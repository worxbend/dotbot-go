package core

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"dotbot-go/internal/config"
	"dotbot-go/internal/fsops"
	"dotbot-go/internal/log"
	"dotbot-go/internal/shell"
)

type Dispatcher struct {
	ctx      *Context
	handlers []Handler
}

type DispatcherConfig struct {
	BaseDirectory string
	Options       Options
	Logger        *log.Logger
	Handlers      []Handler
	FS            fsops.FS
	Shell         shell.Runner
	Clock         func() time.Time
}

func NewDispatcher(cfg DispatcherConfig) (*Dispatcher, error) {
	abs, err := filepath.Abs(cfg.BaseDirectory)
	if err != nil {
		return nil, err
	}
	fs := cfg.FS
	if fs == nil {
		fs = fsops.OSFS{}
	}
	if _, err := fs.Stat(abs); err != nil {
		return nil, fmt.Errorf("nonexistent base directory: %w", err)
	}
	runner := cfg.Shell
	if runner == nil {
		runner = shell.OSRunner{}
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	ctx := &Context{
		RunContext:    context.Background(),
		BaseDirectory: abs,
		Defaults:      map[string]any{},
		Options:       cfg.Options,
		Log:           defaultLogger(cfg.Logger),
		FS:            fs,
		Shell:         runner,
		Clock:         clock,
	}
	handlers := cfg.Handlers
	if handlers == nil {
		handlers = BuiltIns()
	}
	return &Dispatcher{ctx: ctx, handlers: handlers}, nil
}

func defaultLogger(logger *log.Logger) *log.Logger {
	if logger != nil {
		return logger
	}
	return log.New(io.Discard)
}

func (d *Dispatcher) Dispatch(ctx context.Context, tasks []config.Task) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	d.ctx.RunContext = ctx
	success := true
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return false, err
		}
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
					if d.ctx.Options.ExitOnFailure {
						return false, fmt.Errorf("executing action %s: %w", action, err)
					}
					d.ctx.Log.Error(fmt.Sprintf("An error was encountered while executing action %s", action))
					d.ctx.Log.Debug(err.Error())
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
