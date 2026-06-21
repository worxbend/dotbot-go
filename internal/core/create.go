package core

import (
	"fmt"
	"os"

	"dotbot-go/internal/expand"
)

type CreateHandler struct{}

func (CreateHandler) CanHandle(directive string) bool { return directive == "create" }
func (CreateHandler) SupportsDryRun() bool            { return true }

func (CreateHandler) Validate(ctx *Context, directive string, data any) error {
	if paths, ok := asList(data); ok {
		for _, item := range paths {
			if _, ok := asString(item); !ok {
				return fmt.Errorf("create directive item must be a string")
			}
		}
		return nil
	}
	if m, ok := asMap(data); ok {
		for _, path := range sortedKeys(m) {
			options := m[path]
			if options == nil {
				continue
			}
			if _, ok := asMap(options); !ok {
				return fmt.Errorf("create directive options for %s must be a map", path)
			}
		}
		return nil
	}
	return fmt.Errorf("create directive must be a list or map")
}

func (h CreateHandler) Plan(ctx *Context, directive string, data any) ([]Operation, error) {
	if err := h.Validate(ctx, directive, data); err != nil {
		return nil, err
	}
	operations := []Operation{}
	if paths, ok := asList(data); ok {
		for _, item := range paths {
			path, _ := asString(item)
			operations = append(operations, Operation{Directive: directive, Target: path})
		}
		return operations, nil
	}
	m, _ := asMap(data)
	for _, path := range sortedKeys(m) {
		operations = append(operations, Operation{Directive: directive, Target: path})
	}
	return operations, nil
}

func (CreateHandler) Handle(ctx *Context, directive string, data any) (bool, error) {
	success := true
	defaults, _ := asMap(ctx.Defaults["create"])
	paths, ok := asList(data)
	if !ok {
		if m, isMap := asMap(data); isMap {
			for _, key := range sortedKeys(m) {
				options := m[key]
				mode := os.FileMode(0o777)
				if defaults != nil {
					mode = parseMode(defaults["mode"], mode)
				}
				if optionMap, ok := asMap(options); ok && optionMap != nil {
					mode = parseMode(optionMap["mode"], mode)
				}
				success = createPath(ctx, key, mode) && success
			}
			return finish(ctx, success, "All paths have been set up", "Some paths were not successfully set up"), nil
		}
		return false, fmt.Errorf("create directive must be a list or map")
	}
	mode := os.FileMode(0o777)
	if defaults != nil {
		mode = parseMode(defaults["mode"], mode)
	}
	for _, item := range paths {
		path, ok := asString(item)
		if !ok {
			success = false
			continue
		}
		success = createPath(ctx, path, mode) && success
	}
	return finish(ctx, success, "All paths have been set up", "Some paths were not successfully set up"), nil
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
