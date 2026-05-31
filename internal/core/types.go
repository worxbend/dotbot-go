package core

import (
	"os"
	"time"

	"dotbot-go/internal/fsops"
	"dotbot-go/internal/log"
	"dotbot-go/internal/shell"
)

type Options struct {
	Only                  []string
	Skip                  []string
	ExitOnFailure         bool
	DryRun                bool
	Verbose               int
	DisableBuiltInPlugins bool
	ShellTimeout          time.Duration
}

type Context struct {
	BaseDirectory string
	Defaults      map[string]any
	Options       Options
	Log           *log.Logger
	FS            fsops.FS
	Shell         shell.Runner
}

type Handler interface {
	CanHandle(directive string) bool
	SupportsDryRun() bool
	Handle(ctx *Context, directive string, data any) (bool, error)
}

func BuiltIns() []Handler {
	return []Handler{
		CreateHandler{},
		CleanHandler{},
		LinkHandler{},
		ShellHandler{},
	}
}

func defaultFS(fs fsops.FS) fsops.FS {
	if fs != nil {
		return fs
	}
	return fsops.OSFS{}
}

func defaultShell(r shell.Runner) shell.Runner {
	if r != nil {
		return r
	}
	return shell.OSRunner{}
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
