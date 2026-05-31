package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"dotbot-go/internal/expand"
)

type CleanHandler struct{}

func (CleanHandler) CanHandle(directive string) bool { return directive == "clean" }
func (CleanHandler) SupportsDryRun() bool            { return true }

func (CleanHandler) Handle(ctx *Context, directive string, data any) (bool, error) {
	success := true
	defaults, _ := asMap(ctx.Defaults["clean"])
	force := false
	recursive := false
	if defaults != nil {
		force = boolValue(defaults, "force", force)
		recursive = boolValue(defaults, "recursive", recursive)
	}
	if m, ok := asMap(data); ok {
		for target, options := range m {
			localForce, localRecursive := force, recursive
			if om, ok := asMap(options); ok {
				localForce = boolValue(om, "force", localForce)
				localRecursive = boolValue(om, "recursive", localRecursive)
			}
			success = cleanTarget(ctx, target, localForce, localRecursive) && success
		}
		return finish(ctx, success, "All targets have been cleaned", "Some targets were not successfully cleaned"), nil
	}
	items, ok := asList(data)
	if !ok {
		return false, fmt.Errorf("clean directive must be a list or map")
	}
	for _, item := range items {
		target, ok := asString(item)
		if !ok {
			success = false
			continue
		}
		success = cleanTarget(ctx, target, force, recursive) && success
	}
	return finish(ctx, success, "All targets have been cleaned", "Some targets were not successfully cleaned"), nil
}

func cleanTarget(ctx *Context, target string, force, recursive bool) bool {
	dir := expand.Abs(target)
	if !ctx.FS.IsDir(dir) {
		ctx.Log.Debug(fmt.Sprintf("Ignoring nonexistent directory %s", target))
		return true
	}
	names, err := ctx.FS.ListDir(dir)
	if err != nil {
		ctx.Log.Warning(fmt.Sprintf("Failed to list directory %s", dir))
		ctx.Log.Debug(err.Error())
		return false
	}
	success := true
	for _, name := range names {
		path := filepath.Join(dir, name)
		if recursive && ctx.FS.IsDir(path) && !ctx.FS.IsSymlink(path) {
			success = cleanTarget(ctx, path, force, recursive) && success
		}
		if !ctx.FS.Exists(path) && ctx.FS.IsSymlink(path) {
			targetPath, err := ctx.FS.Readlink(path)
			if err != nil {
				success = false
				continue
			}
			pointsAt := filepath.Join(filepath.Dir(path), targetPath)
			if inDirectory(path, ctx.BaseDirectory) || force {
				if ctx.Options.DryRun {
					ctx.Log.Action(fmt.Sprintf("Would remove invalid link %s -> %s", path, pointsAt))
				} else {
					ctx.Log.Action(fmt.Sprintf("Removing invalid link %s -> %s", path, pointsAt))
					if err := ctx.FS.Remove(path); err != nil {
						ctx.Log.Warning(fmt.Sprintf("Failed to remove invalid link %s", path))
						ctx.Log.Debug(err.Error())
						success = false
					}
				}
			} else {
				ctx.Log.Info(fmt.Sprintf("Link %s -> %s not removed.", path, pointsAt))
			}
		}
	}
	return success
}

func inDirectory(path, directory string) bool {
	dir, err := filepath.EvalSymlinks(directory)
	if err != nil {
		dir = directory
	}
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		p = path
	}
	dir = filepath.Clean(dir) + string(filepath.Separator)
	p = filepath.Clean(p)
	return strings.HasPrefix(p, dir)
}
