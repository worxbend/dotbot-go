package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"dotbot-go/internal/expand"
	sh "dotbot-go/internal/shell"
)

type LinkHandler struct{}

func (LinkHandler) CanHandle(directive string) bool { return directive == "link" }
func (LinkHandler) SupportsDryRun() bool            { return true }

type linkOptions struct {
	Relative      bool
	Canonicalize  bool
	Type          string
	Force         bool
	Relink        bool
	Create        bool
	Glob          bool
	Backup        bool
	Prefix        string
	If            string
	IgnoreMissing bool
	Exclude       []string
}

func (LinkHandler) Handle(ctx *Context, directive string, data any) (bool, error) {
	links, ok := asMap(data)
	if !ok {
		return false, fmt.Errorf("link directive must be a map")
	}
	defaults, _ := asMap(ctx.Defaults["link"])
	def := defaultLinkOptions(defaults)
	if def.Type != "symlink" && def.Type != "hardlink" {
		ctx.Log.Warning(fmt.Sprintf("The default link type is not recognized: '%s'", def.Type))
		return false, nil
	}
	success := true
	for linkName, target := range links {
		opts := def
		linkName = os.ExpandEnv(expand.NormSlash(linkName))
		path := ""
		if targetMap, ok := asMap(target); ok {
			opts = mergeLinkOptions(opts, targetMap)
			if opts.Type != "symlink" && opts.Type != "hardlink" {
				ctx.Log.Warning(fmt.Sprintf("The link type is not recognized: '%s'", opts.Type))
				success = false
				continue
			}
			path = defaultTarget(linkName, targetMap["path"])
		} else {
			path = defaultTarget(linkName, target)
		}
		path = filepath.Clean(expand.Path(path))
		if opts.If != "" && !testSuccess(ctx, opts.If) {
			ctx.Log.Info(fmt.Sprintf("Skipping %s", linkName))
			continue
		}
		if opts.Glob && hasGlobChars(path) {
			matches := createGlobResults(ctx, path, opts.Exclude)
			ctx.Log.Debug(fmt.Sprintf("Globs from '%s': %v", path, matches))
			for _, fullItem := range matches {
				globItem := globLinkItem(path, fullItem)
				if opts.Prefix != "" {
					globItem = opts.Prefix + globItem
				}
				globLinkName := filepath.Join(linkName, globItem)
				success = processOneLink(ctx, fullItem, globLinkName, opts, true) && success
			}
			continue
		}
		success = processOneLink(ctx, path, linkName, opts, false) && success
	}
	return finish(ctx, success, "All links have been set up", "Some links were not successfully set up"), nil
}

func defaultLinkOptions(defaults map[string]any) linkOptions {
	opts := linkOptions{Canonicalize: true, Type: "symlink"}
	if defaults == nil {
		return opts
	}
	return mergeLinkOptions(opts, defaults)
}

func mergeLinkOptions(opts linkOptions, values map[string]any) linkOptions {
	opts.Relative = boolValue(values, "relative", opts.Relative)
	if v, ok := values["canonicalize"]; ok {
		if b, ok := v.(bool); ok {
			opts.Canonicalize = b
		}
	} else {
		opts.Canonicalize = boolValue(values, "canonicalize-path", opts.Canonicalize)
	}
	opts.Type = stringValue(values, "type", opts.Type)
	opts.Force = boolValue(values, "force", opts.Force)
	opts.Relink = boolValue(values, "relink", opts.Relink)
	opts.Create = boolValue(values, "create", opts.Create)
	opts.Glob = boolValue(values, "glob", opts.Glob)
	opts.Backup = boolValue(values, "backup", opts.Backup)
	opts.Prefix = stringValue(values, "prefix", opts.Prefix)
	opts.If = stringValue(values, "if", opts.If)
	opts.IgnoreMissing = boolValue(values, "ignore-missing", opts.IgnoreMissing)
	if v, ok := values["exclude"]; ok {
		opts.Exclude = stringSlice(v)
	}
	return opts
}

func processOneLink(ctx *Context, target, linkName string, opts linkOptions, globbed bool) bool {
	success := true
	if opts.Create {
		success = createParent(ctx, linkName) && success
	}
	if !globbed && !opts.IgnoreMissing && !ctx.FS.Exists(filepath.Join(baseDir(ctx, opts.Canonicalize), target)) {
		ctx.Log.Warning(fmt.Sprintf("Nonexistent target %s -> %s", linkName, target))
		return false
	}
	didBackup := false
	didDelete := false
	backupSuccess := true
	if opts.Backup {
		didBackup, backupSuccess = backup(ctx, linkName)
		success = backupSuccess && success
	}
	if (opts.Force || opts.Relink) && !(didBackup && backupSuccess) {
		var deleteSuccess bool
		didDelete, deleteSuccess = deleteLink(ctx, target, linkName, opts)
		success = deleteSuccess && success
	}
	return createLink(ctx, target, linkName, opts, opts.IgnoreMissing, didBackup || didDelete) && success
}

func testSuccess(ctx *Context, command string) bool {
	ret := ctx.Shell.Run(context.Background(), command, sh.Options{
		CWD:     ctx.BaseDirectory,
		Timeout: defaultTimeout(ctx.Options.ShellTimeout),
	})
	if ret != 0 {
		ctx.Log.Debug(fmt.Sprintf("Test '%s' returned false", command))
	}
	return ret == 0
}

func baseDir(ctx *Context, canonical bool) string {
	if !canonical {
		return ctx.BaseDirectory
	}
	if real, err := ctx.FS.Realpath(ctx.BaseDirectory); err == nil {
		return real
	}
	return ctx.BaseDirectory
}

func createParent(ctx *Context, path string) bool {
	parent := filepath.Dir(expand.Abs(path))
	if ctx.FS.Exists(parent) {
		return true
	}
	ctx.Log.Debug(fmt.Sprintf("Try to create parent: %s", parent))
	if ctx.Options.DryRun {
		ctx.Log.Action(fmt.Sprintf("Would create directory %s", parent))
		return true
	}
	if err := ctx.FS.MkdirAll(parent, 0o777); err != nil {
		ctx.Log.Warning(fmt.Sprintf("Failed to create directory %s", parent))
		ctx.Log.Debug(err.Error())
		return false
	}
	ctx.Log.Action(fmt.Sprintf("Creating directory %s", parent))
	return true
}

func backup(ctx *Context, path string) (bool, bool) {
	if ctx.FS.Exists(expand.Path(path)) && !ctx.FS.IsSymlink(expand.Path(path)) {
		timestamp := time.Now().Format("20060102-150405")
		backupName := path + ".dotbot-backup." + timestamp
		ctx.Log.Debug(fmt.Sprintf("Try to backup file %s to %s", path, backupName))
		if ctx.Options.DryRun {
			ctx.Log.Action(fmt.Sprintf("Would backup %s to %s", path, backupName))
			return true, true
		}
		if err := ctx.FS.Rename(expand.Abs(path), expand.Abs(backupName)); err != nil {
			ctx.Log.Warning(fmt.Sprintf("Failed to backup file %s to %s", path, backupName))
			ctx.Log.Debug(err.Error())
			return false, false
		}
		ctx.Log.Action(fmt.Sprintf("Backed up file %s to %s", path, backupName))
		return true, true
	}
	return false, true
}

func deleteLink(ctx *Context, target, path string, opts linkOptions) (bool, bool) {
	removed := false
	targetAbs := filepath.Join(baseDir(ctx, opts.Canonicalize), target)
	fullpath := expand.Abs(path)
	if ctx.FS.Exists(expand.Path(path)) && !ctx.FS.IsSymlink(expand.Path(path)) {
		if same, err := ctx.FS.SameFile(fullpath, targetAbs); err == nil && same {
			ctx.Log.Warning(fmt.Sprintf("%s appears to be the same file as %s.", path, targetAbs))
			return false, false
		}
	}
	targetPath := targetAbs
	if opts.Relative {
		targetPath = relativePath(targetAbs, fullpath)
	}
	shouldRemove := false
	if ctx.FS.IsSymlink(expand.Path(path)) {
		current, _ := ctx.FS.Readlink(expand.Path(path))
		shouldRemove = current != targetPath
	} else if ctx.FS.Lexists(expand.Path(path)) {
		shouldRemove = true
	}
	if !shouldRemove {
		return false, true
	}
	if ctx.Options.DryRun {
		ctx.Log.Action(fmt.Sprintf("Would remove %s", path))
		return true, true
	}
	var err error
	if ctx.FS.IsSymlink(fullpath) {
		err = ctx.FS.Remove(fullpath)
		removed = true
	} else if opts.Force {
		if ctx.FS.IsDir(fullpath) {
			err = ctx.FS.RemoveAll(fullpath)
		} else {
			err = ctx.FS.Remove(fullpath)
		}
		removed = true
	}
	if err != nil {
		ctx.Log.Warning(fmt.Sprintf("Failed to remove %s", path))
		ctx.Log.Debug(err.Error())
		return removed, false
	}
	if removed {
		ctx.Log.Action(fmt.Sprintf("Removing %s", path))
	}
	return removed, true
}

func createLink(ctx *Context, target, linkName string, opts linkOptions, ignoreMissing, assumeGone bool) bool {
	linkPath := expand.Abs(linkName)
	absoluteTarget := filepath.Join(baseDir(ctx, opts.Canonicalize), target)
	targetPath := absoluteTarget
	if opts.Relative {
		targetPath = relativePath(absoluteTarget, linkPath)
	}
	linkExists := ctx.FS.Lexists(expand.Path(linkName))
	if (!linkExists || (ctx.Options.DryRun && assumeGone)) && (ignoreMissing || ctx.FS.Exists(absoluteTarget)) {
		if ctx.Options.DryRun {
			ctx.Log.Action(fmt.Sprintf("Would create %s %s -> %s", opts.Type, filepath.Clean(linkName), targetPath))
			return true
		}
		var err error
		if opts.Type == "symlink" {
			err = ctx.FS.Symlink(targetPath, linkPath)
		} else {
			err = ctx.FS.Link(absoluteTarget, linkPath)
		}
		if err != nil {
			ctx.Log.Warning(fmt.Sprintf("Linking failed %s -> %s", filepath.Clean(linkName), targetPath))
			ctx.Log.Debug(err.Error())
			return false
		}
		ctx.Log.Action(fmt.Sprintf("Creating %s %s -> %s", opts.Type, filepath.Clean(linkName), targetPath))
		return true
	}
	if ctx.FS.IsSymlink(expand.Path(linkName)) {
		if opts.Type == "symlink" {
			current, _ := ctx.FS.Readlink(expand.Path(linkName))
			if current == targetPath {
				ctx.Log.Info(fmt.Sprintf("Link exists %s -> %s", filepath.Clean(linkName), targetPath))
				return true
			}
			term := "Incorrect"
			if !ctx.FS.Exists(expand.Path(linkName)) {
				term = "Invalid"
			}
			ctx.Log.Warning(fmt.Sprintf("%s link %s -> %s", term, filepath.Clean(linkName), current))
			return false
		}
		ctx.Log.Warning(fmt.Sprintf("%s already exists but is a symbolic link, not a hard link", filepath.Clean(linkName)))
		return false
	}
	if opts.Type == "hardlink" {
		if same, err := ctx.FS.SameFile(linkPath, absoluteTarget); err == nil && same {
			ctx.Log.Info(fmt.Sprintf("Link exists %s -> %s", filepath.Clean(linkName), targetPath))
			return true
		}
	}
	ctx.Log.Warning(fmt.Sprintf("%s already exists but is a regular file or directory", filepath.Clean(linkName)))
	return false
}

func relativePath(target, linkName string) string {
	linkDir := filepath.Dir(linkName)
	rel, err := filepath.Rel(linkDir, target)
	if err != nil {
		return target
	}
	return rel
}

func hasGlobChars(path string) bool {
	return strings.ContainsAny(path, "?*[")
}

func createGlobResults(ctx *Context, pattern string, exclude []string) []string {
	include := glob(ctx, pattern)
	excluded := map[string]bool{}
	for _, ex := range exclude {
		for _, item := range glob(ctx, ex) {
			excluded[item] = true
		}
	}
	var out []string
	for _, item := range include {
		if !excluded[item] {
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func glob(ctx *Context, pattern string) []string {
	var matches []string
	if strings.Contains(pattern, "**") {
		root := pattern[:strings.Index(pattern, "**")]
		root = strings.TrimRight(root, string(filepath.Separator))
		if root == "" {
			root = "."
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			ok, _ := doublestarMatch(pattern, path)
			if ok {
				if !strings.HasSuffix(pattern, string(filepath.Separator)) && d.IsDir() {
					return nil
				}
				matches = append(matches, filepath.Clean(path))
			}
			return nil
		})
		return matches
	}
	found, err := filepath.Glob(pattern)
	if err != nil {
		ctx.Log.Debug(err.Error())
		return nil
	}
	for _, item := range found {
		matches = append(matches, filepath.Clean(item))
	}
	return matches
}

func doublestarMatch(pattern, path string) (bool, error) {
	if !strings.Contains(pattern, "**") {
		return filepath.Match(pattern, path)
	}
	parts := strings.Split(pattern, "**")
	prefix := filepath.Clean(strings.TrimRight(parts[0], string(filepath.Separator)))
	suffix := ""
	if len(parts) > 1 {
		suffix = strings.TrimLeft(parts[1], string(filepath.Separator))
	}
	if prefix != "." && prefix != "" {
		cleanPath := filepath.Clean(path)
		if cleanPath != prefix && !strings.HasPrefix(cleanPath, prefix+string(filepath.Separator)) {
			return false, nil
		}
	}
	if suffix == "" {
		return true, nil
	}
	return filepath.Match(suffix, filepath.Base(path))
}

func globLinkItem(pattern, item string) string {
	dir := filepath.Dir(commonPrefix(pattern, item))
	if dir == "." || dir == string(filepath.Separator) || dir == "" {
		return item
	}
	rel := strings.TrimPrefix(item, dir+string(filepath.Separator))
	return rel
}

func commonPrefix(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}
