package core

import (
	"fmt"
	"path/filepath"

	"dotbot-go/internal/expand"
)

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

func backup(ctx *Context, path string) (bool, bool) {
	if ctx.FS.Exists(expand.Path(path)) && !ctx.FS.IsSymlink(expand.Path(path)) {
		timestamp := ctx.Clock().Format("20060102-150405")
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
		current, err := ctx.FS.Readlink(expand.Path(path))
		if err != nil {
			ctx.Log.Warning(fmt.Sprintf("Failed to inspect link %s", path))
			ctx.Log.Debug(err.Error())
			return false, false
		}
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
			current, err := ctx.FS.Readlink(expand.Path(linkName))
			if err != nil {
				ctx.Log.Warning(fmt.Sprintf("Failed to inspect link %s", filepath.Clean(linkName)))
				ctx.Log.Debug(err.Error())
				return false
			}
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
