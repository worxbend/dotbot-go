package core

import (
	"fmt"
	"path/filepath"

	"dotbot-go/internal/expand"
)

func processOneLink(ctx *Context, target, linkName string, opts linkOptions, globbed bool) bool {
	success := true
	link := resolveLink(ctx, target, linkName, opts)
	if opts.Create {
		success = createParent(ctx, link.linkName) && success
	}
	if !globbed && !opts.IgnoreMissing && !ctx.FS.Exists(link.absoluteTarget) {
		ctx.Log.Warning(fmt.Sprintf("Nonexistent target %s -> %s", link.linkName, link.target))
		return false
	}
	didBackup := false
	didDelete := false
	backupSuccess := true
	if opts.Backup {
		didBackup, backupSuccess = backup(ctx, link)
		if !backupSuccess {
			return false
		}
	}
	if (opts.Force || opts.Relink) && !didBackup {
		var deleteSuccess bool
		didDelete, deleteSuccess = deleteLink(ctx, link, opts)
		success = deleteSuccess && success
	}
	return createLink(ctx, link, opts, opts.IgnoreMissing, didBackup || didDelete) && success
}

func backup(ctx *Context, link linkResolution) (bool, bool) {
	path := link.linkName
	if ctx.FS.Exists(expand.Path(path)) && !ctx.FS.IsSymlink(expand.Path(path)) {
		timestamp := ctx.Clock().Format("20060102-150405")
		pattern := filepath.Base(link.linkPath) + ".dotbot-backup." + timestamp + "-"
		if ctx.Options.DryRun {
			ctx.Log.Action(fmt.Sprintf("Would backup %s in %s*", path, filepath.Join(filepath.Dir(link.linkPath), pattern)))
			return true, true
		}
		// Reserve a private directory atomically; rename cannot replace an older backup.
		backupDir, err := ctx.FS.MkdirTemp(filepath.Dir(link.linkPath), pattern)
		if err != nil {
			ctx.Log.Warning(fmt.Sprintf("Failed to reserve backup for %s", path))
			ctx.Log.Debug(err.Error())
			return false, false
		}
		backupName := filepath.Join(backupDir, filepath.Base(link.linkPath))
		if err := ctx.FS.Rename(expand.Abs(path), expand.Abs(backupName)); err != nil {
			ctx.Log.Warning(fmt.Sprintf("Failed to backup file %s to %s", path, backupName))
			ctx.Log.Debug(err.Error())
			if removeErr := ctx.FS.Remove(backupDir); removeErr != nil {
				ctx.Log.Warning(fmt.Sprintf("Failed to remove empty backup directory %s: %v", backupDir, removeErr))
			}
			return false, false
		}
		ctx.Log.Action(fmt.Sprintf("Backed up file %s to %s", path, backupName))
		return true, true
	}
	return false, true
}

func deleteLink(ctx *Context, link linkResolution, opts linkOptions) (bool, bool) {
	path := link.linkName
	if sameFileConflict(ctx, link) {
		return false, false
	}
	shouldRemove, ok := shouldRemoveLink(ctx, link)
	if !ok {
		return false, false
	}
	if !shouldRemove {
		return false, true
	}
	if ctx.Options.DryRun {
		ctx.Log.Action(fmt.Sprintf("Would remove %s", path))
		return true, true
	}
	removed, err := removeExistingLink(ctx, link, opts)
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

func sameFileConflict(ctx *Context, link linkResolution) bool {
	path := link.linkName
	if !ctx.FS.Exists(expand.Path(path)) || ctx.FS.IsSymlink(expand.Path(path)) {
		return false
	}
	same, err := ctx.FS.SameFile(link.linkPath, link.absoluteTarget)
	if err != nil || !same {
		return false
	}
	ctx.Log.Warning(fmt.Sprintf("%s appears to be the same file as %s.", path, link.absoluteTarget))
	return true
}

func shouldRemoveLink(ctx *Context, link linkResolution) (bool, bool) {
	path := link.linkName
	if ctx.FS.IsSymlink(expand.Path(path)) {
		current, err := ctx.FS.Readlink(expand.Path(path))
		if err != nil {
			ctx.Log.Warning(fmt.Sprintf("Failed to inspect link %s", path))
			ctx.Log.Debug(err.Error())
			return false, false
		}
		return current != link.targetPath, true
	}
	if ctx.FS.Lexists(expand.Path(path)) {
		return true, true
	}
	return false, true
}

func removeExistingLink(ctx *Context, link linkResolution, opts linkOptions) (bool, error) {
	fullpath := link.linkPath
	if ctx.FS.IsSymlink(fullpath) {
		return true, ctx.FS.Remove(fullpath)
	}
	if !opts.Force {
		return false, nil
	}
	if ctx.FS.IsDir(fullpath) {
		return true, ctx.FS.RemoveAll(fullpath)
	}
	return true, ctx.FS.Remove(fullpath)
}

func createLink(ctx *Context, link linkResolution, opts linkOptions, ignoreMissing, assumeGone bool) bool {
	if shouldCreateLink(ctx, link, ignoreMissing, assumeGone) {
		return createNewLink(ctx, link, opts)
	}
	if ctx.FS.IsSymlink(expand.Path(link.linkName)) {
		return handleExistingSymlink(ctx, link, opts)
	}
	return handleExistingNonSymlink(ctx, link, opts)
}

func shouldCreateLink(ctx *Context, link linkResolution, ignoreMissing, assumeGone bool) bool {
	linkExists := ctx.FS.Lexists(expand.Path(link.linkName))
	targetExists := ignoreMissing || ctx.FS.Exists(link.absoluteTarget)
	return (!linkExists || (ctx.Options.DryRun && assumeGone)) && targetExists
}

func createNewLink(ctx *Context, link linkResolution, opts linkOptions) bool {
	if ctx.Options.DryRun {
		ctx.Log.Action(fmt.Sprintf("Would create %s %s -> %s", opts.Type, link.cleanLinkName(), link.targetPath))
		return true
	}
	var err error
	if opts.Type == "symlink" {
		err = ctx.FS.Symlink(link.targetPath, link.linkPath)
	} else {
		err = ctx.FS.Link(link.absoluteTarget, link.linkPath)
	}
	if err != nil {
		ctx.Log.Warning(fmt.Sprintf("Linking failed %s -> %s", link.cleanLinkName(), link.targetPath))
		ctx.Log.Debug(err.Error())
		return false
	}
	ctx.Log.Action(fmt.Sprintf("Creating %s %s -> %s", opts.Type, link.cleanLinkName(), link.targetPath))
	return true
}

func handleExistingSymlink(ctx *Context, link linkResolution, opts linkOptions) bool {
	if opts.Type != "symlink" {
		ctx.Log.Warning(fmt.Sprintf("%s already exists but is a symbolic link, not a hard link", link.cleanLinkName()))
		return false
	}
	current, err := ctx.FS.Readlink(expand.Path(link.linkName))
	if err != nil {
		ctx.Log.Warning(fmt.Sprintf("Failed to inspect link %s", link.cleanLinkName()))
		ctx.Log.Debug(err.Error())
		return false
	}
	if current == link.targetPath {
		ctx.Log.Info(fmt.Sprintf("Link exists %s -> %s", link.cleanLinkName(), link.targetPath))
		return true
	}
	term := "Incorrect"
	if !ctx.FS.Exists(expand.Path(link.linkName)) {
		term = "Invalid"
	}
	ctx.Log.Warning(fmt.Sprintf("%s link %s -> %s", term, link.cleanLinkName(), current))
	return false
}

func handleExistingNonSymlink(ctx *Context, link linkResolution, opts linkOptions) bool {
	if opts.Type == "hardlink" {
		if same, err := ctx.FS.SameFile(link.linkPath, link.absoluteTarget); err == nil && same {
			ctx.Log.Info(fmt.Sprintf("Link exists %s -> %s", link.cleanLinkName(), link.targetPath))
			return true
		}
	}
	ctx.Log.Warning(fmt.Sprintf("%s already exists but is a regular file or directory", link.cleanLinkName()))
	return false
}
