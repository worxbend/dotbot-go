package core

import (
	"context"
	"os"
	"time"

	"dotbot-go/internal/fsops"
	"dotbot-go/internal/log"
	"dotbot-go/internal/shell"
)

// Options controls directive filtering and execution behavior.
type Options struct {
	// Only limits dispatch and planning to the named directives.
	Only []string
	// Skip excludes the named directives from dispatch and planning.
	Skip []string
	// ExitOnFailure stops dispatch after the first handler failure.
	ExitOnFailure bool
	// DryRun previews supported handlers without mutating the filesystem or shell.
	DryRun bool
	// Verbose increases handler detail; values greater than one expose shell output.
	Verbose int
	// ShellTimeout bounds each shell command when greater than zero.
	ShellTimeout time.Duration
}

// Context carries shared state and adapters for directive handlers.
type Context struct {
	// RunContext is the cancellation context used by shell commands and dispatch.
	RunContext context.Context
	// BaseDirectory is the absolute repository root for relative operations.
	BaseDirectory string
	// Defaults holds the currently active defaults directive values.
	Defaults map[string]any
	// Options are the execution controls for this dispatch or plan.
	Options Options
	// Log writes user-facing handler messages.
	Log *log.Logger
	// FS performs filesystem operations.
	FS fsops.FS
	// Shell runs shell directive commands.
	Shell shell.Runner
	// Clock supplies timestamps for backup names.
	Clock func() time.Time
}

// Handler executes one directive type.
type Handler interface {
	// CanHandle reports whether this handler owns directive.
	CanHandle(directive string) bool
	// SupportsDryRun reports whether Handle can safely preview work.
	SupportsDryRun() bool
	// Handle applies directive data and returns whether the action succeeded.
	Handle(ctx *Context, directive string, data any) (bool, error)
}

type validatingHandler interface {
	Handler
	Validate(ctx *Context, directive string, data any) error
}

type planningHandler interface {
	Handler
	Plan(ctx *Context, directive string, data any) ([]Operation, error)
}

// BuiltIns returns the built-in directive handlers in dispatch order.
func BuiltIns() []Handler {
	return []Handler{
		CreateHandler{},
		CleanHandler{},
		LinkHandler{},
		ShellHandler{},
	}
}

func defaultTimeout(d time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return 10 * time.Minute
}

func fileMode(v any) (os.FileMode, bool) {
	switch t := v.(type) {
	case int:
		return os.FileMode(t), true
	case int64:
		return os.FileMode(t), true
	case uint64:
		return os.FileMode(t), true
	case float64:
		return os.FileMode(t), true
	default:
		return 0, false
	}
}
