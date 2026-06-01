package core

import (
	"fmt"
	"path/filepath"

	"dotbot-go/internal/expand"
	sh "dotbot-go/internal/shell"
)

func testSuccess(ctx *Context, command string) bool {
	ret := ctx.Shell.Run(ctx.RunContext, command, sh.Options{
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

func relativePath(target, linkName string) string {
	linkDir := filepath.Dir(linkName)
	rel, err := filepath.Rel(linkDir, target)
	if err != nil {
		return target
	}
	return rel
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
