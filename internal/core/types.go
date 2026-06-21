package core

import (
	"context"
	"os"
	"time"

	"dotbot-go/internal/fsops"
	"dotbot-go/internal/log"
	"dotbot-go/internal/shell"
)

type Options struct {
	Only          []string
	Skip          []string
	ExitOnFailure bool
	DryRun        bool
	Verbose       int
	ShellTimeout  time.Duration
}

type Context struct {
	RunContext    context.Context
	BaseDirectory string
	Defaults      map[string]any
	Options       Options
	Log           *log.Logger
	FS            fsops.FS
	Shell         shell.Runner
	Clock         func() time.Time
}

type Handler interface {
	CanHandle(directive string) bool
	SupportsDryRun() bool
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

func fileMode(v any, fallback os.FileMode) os.FileMode {
	switch t := v.(type) {
	case int:
		return os.FileMode(t)
	case int64:
		return os.FileMode(t)
	case uint64:
		return os.FileMode(t)
	case float64:
		return os.FileMode(t)
	default:
		return fallback
	}
}
