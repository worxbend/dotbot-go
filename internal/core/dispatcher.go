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

// Dispatcher plans and applies ordered configuration tasks.
type Dispatcher struct {
	ctx      *Context
	handlers []Handler
}

// DispatcherConfig provides dependencies and options for a Dispatcher.
type DispatcherConfig struct {
	// BaseDirectory is the repository root used by filesystem and shell handlers.
	BaseDirectory string
	// Options control filtering, dry-run behavior, and failure handling.
	Options Options
	// Logger receives user-facing output; a discard logger is used when nil.
	Logger *log.Logger
	// Handlers overrides the built-in directive handler list when provided.
	Handlers []Handler
	// FS performs filesystem operations; OSFS is used when nil.
	FS fsops.FS
	// Shell runs shell commands; OSRunner is used when nil.
	Shell shell.Runner
	// Clock supplies timestamps for backup names; time.Now is used when nil.
	Clock func() time.Time
}

// NewDispatcher builds a Dispatcher and validates that BaseDirectory exists.
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

// Dispatch applies tasks in order.
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
		for _, taskAction := range task.Actions() {
			action := taskAction.Directive
			data := taskAction.Data
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
			for _, handler := range d.handlers {
				if !handler.CanHandle(action) {
					continue
				}
				if d.ctx.Options.DryRun && !handler.SupportsDryRun() {
					d.ctx.Log.Action(fmt.Sprintf("Skipping dry-run-unaware handler %T", handler))
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

// Validate checks tasks by building a plan without applying operations.
func (d *Dispatcher) Validate(tasks []config.Task) error {
	_, err := d.Plan(tasks)
	return err
}

// Plan expands tasks into operations without applying filesystem or shell changes.
func (d *Dispatcher) Plan(tasks []config.Task) (Plan, error) {
	d.ctx.Defaults = map[string]any{}
	plan := Plan{Operations: []Operation{}}
	for _, task := range tasks {
		for _, taskAction := range task.Actions() {
			action := taskAction.Directive
			data := taskAction.Data
			if d.shouldSkip(action) && action != "defaults" {
				continue
			}
			if action == "defaults" {
				if defaults, ok := asMap(data); ok {
					d.ctx.Defaults = defaults
				} else {
					d.ctx.Defaults = map[string]any{}
				}
				continue
			}
			handler := d.handlerFor(action)
			if handler == nil {
				return plan, fmt.Errorf("action %s not handled", action)
			}
			validator, ok := handler.(validatingHandler)
			if ok {
				if err := validator.Validate(d.ctx, action, data); err != nil {
					return plan, fmt.Errorf("validating action %s: %w", action, err)
				}
			}
			planner, ok := handler.(planningHandler)
			if !ok {
				plan.Operations = append(plan.Operations, Operation{Directive: action})
				continue
			}
			operations, err := planner.Plan(d.ctx, action, data)
			if err != nil {
				return plan, fmt.Errorf("planning action %s: %w", action, err)
			}
			plan.Operations = append(plan.Operations, operations...)
		}
	}
	return plan, nil
}

func (d *Dispatcher) handlerFor(action string) Handler {
	for _, handler := range d.handlers {
		if handler.CanHandle(action) {
			return handler
		}
	}
	return nil
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
