package core

import (
	"fmt"
	"os"
	"path/filepath"

	"dotbot-go/internal/config"
	"dotbot-go/internal/log"
)

type Dispatcher struct {
	ctx      *Context
	handlers []Handler
}

func NewDispatcher(baseDirectory string, opts Options, logger *log.Logger, handlers []Handler) (*Dispatcher, error) {
	abs, err := filepath.Abs(baseDirectory)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("nonexistent base directory")
	}
	ctx := &Context{
		BaseDirectory: abs,
		Defaults:      map[string]any{},
		Options:       opts,
		Log:           logger,
		FS:            defaultFS(nil),
		Shell:         defaultShell(nil),
	}
	if handlers == nil {
		handlers = BuiltIns()
	}
	return &Dispatcher{ctx: ctx, handlers: handlers}, nil
}

func (d *Dispatcher) Dispatch(tasks []config.Task) (bool, error) {
	success := true
	for _, task := range tasks {
		for action, data := range task {
			if d.shouldSkip(action) && action != "defaults" {
				d.ctx.Log.Info(fmt.Sprintf("Skipping action %s", action))
				continue
			}
			handled := false
			if action == "defaults" {
				if defaults, ok := asMap(data); ok {
					d.ctx.Defaults = defaults
				} else {
					d.ctx.Defaults = map[string]any{}
				}
				handled = true
			}
			if action == "plugins" {
				d.ctx.Log.Warning("Go migration does not support Python plugin loading")
				success = false
				handled = true
				if d.ctx.Options.ExitOnFailure {
					d.ctx.Log.Error("Action plugins failed")
					return false, nil
				}
			}
			for _, handler := range d.handlers {
				if !handler.CanHandle(action) {
					continue
				}
				if d.ctx.Options.DryRun && !handler.SupportsDryRun() {
					d.ctx.Log.Action(fmt.Sprintf("Skipping dry-run-unaware plugin %T", handler))
					handled = true
					continue
				}
				localSuccess, err := handler.Handle(d.ctx, action, data)
				if err != nil {
					d.ctx.Log.Error(fmt.Sprintf("An error was encountered while executing action %s", action))
					d.ctx.Log.Debug(err.Error())
					if d.ctx.Options.ExitOnFailure {
						return false, err
					}
					success = false
				}
				if !localSuccess && d.ctx.Options.ExitOnFailure {
					d.ctx.Log.Error(fmt.Sprintf("Action %s failed", action))
					return false, nil
				}
				success = success && localSuccess
				handled = true
			}
			if !handled {
				success = false
				d.ctx.Log.Error(fmt.Sprintf("Action %s not handled", action))
				if d.ctx.Options.ExitOnFailure {
					return false, nil
				}
			}
		}
	}
	return success, nil
}

func (d *Dispatcher) shouldSkip(action string) bool {
	if len(d.ctx.Options.Only) > 0 && !contains(d.ctx.Options.Only, action) {
		return true
	}
	return contains(d.ctx.Options.Skip, action)
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
