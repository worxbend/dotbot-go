package shell

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// Options controls how a shell command is executed.
type Options struct {
	// CWD is the working directory for the command.
	CWD string
	// EnableStdin connects the command to os.Stdin.
	EnableStdin bool
	// EnableStdout connects the command to os.Stdout.
	EnableStdout bool
	// EnableStderr connects the command to os.Stderr.
	EnableStderr bool
	// Timeout cancels the command after the duration when greater than zero.
	Timeout time.Duration
}

// Runner executes shell command strings for the shell directive.
type Runner interface {
	// Run executes command and returns its process exit code.
	Run(ctx context.Context, command string, opts Options) int
}

// OSRunner runs commands through the host platform shell.
type OSRunner struct{}

// Run executes command through the host platform shell.
func (OSRunner) Run(ctx context.Context, command string, opts Options) int {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	name, args := shellCommand(command)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = opts.CWD
	if opts.EnableStdin {
		cmd.Stdin = os.Stdin
	} else {
		cmd.Stdin = nil
	}
	devnullW, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		devnullW = nil
	}
	if devnullW != nil {
		defer func() {
			_ = devnullW.Close()
		}()
	}
	if opts.EnableStdout {
		cmd.Stdout = os.Stdout
	} else if devnullW != nil {
		cmd.Stdout = devnullW
	} else {
		cmd.Stdout = io.Discard
	}
	if opts.EnableStderr {
		cmd.Stderr = os.Stderr
	} else if devnullW != nil {
		cmd.Stderr = devnullW
	} else {
		cmd.Stderr = io.Discard
	}
	err = cmd.Run()
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	return sh, []string{"-c", command}
}
