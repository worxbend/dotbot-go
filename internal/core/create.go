package core

import (
	"fmt"
	"os"

	"dotbot-go/internal/expand"
)

// CreateHandler implements the create directive.
type CreateHandler struct{}

// CanHandle reports whether directive is create.
func (CreateHandler) CanHandle(directive string) bool { return directive == "create" }

// SupportsDryRun reports that create can preview directory creation.
func (CreateHandler) SupportsDryRun() bool { return true }

// Validate checks create directive data without touching the filesystem.
func (CreateHandler) Validate(ctx *Context, directive string, data any) error {
	entries, err := createEntries(data)
	if err != nil {
		return err
	}
	defaults, _ := asMap(ctx.Defaults["create"])
	for _, entry := range entries {
		if _, err := createMode(defaults, entry.options); err != nil {
			return fmt.Errorf("create directive mode for %s: %w", entry.path, err)
		}
	}
	return nil
}

// Plan expands create directive data into directory operations.
func (h CreateHandler) Plan(ctx *Context, directive string, data any) ([]Operation, error) {
	entries, err := createEntries(data)
	if err != nil {
		return nil, err
	}
	operations := make([]Operation, 0, len(entries))
	for _, entry := range entries {
		operations = append(operations, Operation{Directive: directive, Target: entry.path})
	}
	return operations, nil
}

// Handle creates directories requested by the create directive.
func (CreateHandler) Handle(ctx *Context, directive string, data any) (bool, error) {
	entries, err := createEntries(data)
	if err != nil {
		return false, err
	}
	success := true
	defaults, _ := asMap(ctx.Defaults["create"])
	for _, entry := range entries {
		mode, err := createMode(defaults, entry.options)
		if err != nil {
			return false, fmt.Errorf("create directive mode for %s: %w", entry.path, err)
		}
		success = createPath(ctx, entry.path, mode) && success
	}
	return finish(ctx, success, "All paths have been set up", "Some paths were not successfully set up"), nil
}

type createEntry struct {
	path    string
	options map[string]any
}

func createEntries(data any) ([]createEntry, error) {
	if paths, ok := asList(data); ok {
		entries := make([]createEntry, 0, len(paths))
		for _, item := range paths {
			path, ok := asString(item)
			if !ok {
				return nil, fmt.Errorf("create directive item must be a string")
			}
			entries = append(entries, createEntry{path: path})
		}
		return entries, nil
	}
	if m, ok := asMap(data); ok {
		entries := make([]createEntry, 0, len(m))
		for _, path := range sortedKeys(m) {
			options := m[path]
			if options == nil {
				entries = append(entries, createEntry{path: path})
				continue
			}
			optionMap, ok := asMap(options)
			if !ok {
				return nil, fmt.Errorf("create directive options for %s must be a map", path)
			}
			entries = append(entries, createEntry{path: path, options: optionMap})
		}
		return entries, nil
	}
	return nil, fmt.Errorf("create directive must be a list or map")
}

func createMode(defaults, options map[string]any) (os.FileMode, error) {
	mode := os.FileMode(0o777)
	if defaults != nil {
		var err error
		mode, err = parseMode(defaults["mode"], mode)
		if err != nil {
			return 0, fmt.Errorf("default mode: %w", err)
		}
	}
	if options != nil {
		var err error
		mode, err = parseMode(options["mode"], mode)
		if err != nil {
			return 0, err
		}
	}
	return mode, nil
}

func createPath(ctx *Context, path string, mode os.FileMode) bool {
	abs := expand.Abs(path)
	if ctx.FS.Exists(abs) {
		ctx.Log.Info(fmt.Sprintf("Path exists %s", abs))
		return true
	}
	ctx.Log.Debug(fmt.Sprintf("Trying to create path %s with mode %d", abs, mode))
	if ctx.Options.DryRun {
		ctx.Log.Action(fmt.Sprintf("Would create path %s", abs))
		return true
	}
	ctx.Log.Action(fmt.Sprintf("Creating path %s", abs))
	if err := ctx.FS.MkdirAll(abs, mode); err != nil {
		ctx.Log.Warning(fmt.Sprintf("Failed to create path %s", abs))
		ctx.Log.Debug(err.Error())
		return false
	}
	if err := ctx.FS.Chmod(abs, mode); err != nil {
		ctx.Log.Warning(fmt.Sprintf("Failed to set mode for path %s", abs))
		ctx.Log.Debug(err.Error())
		return false
	}
	return true
}

func finish(ctx *Context, success bool, okMessage, failMessage string) bool {
	if success {
		ctx.Log.Info(okMessage)
	} else {
		ctx.Log.Error(failMessage)
	}
	return success
}
