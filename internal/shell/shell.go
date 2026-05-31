package shell

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type Options struct {
	CWD          string
	EnableStdin  bool
	EnableStdout bool
	EnableStderr bool
	Timeout      time.Duration
}

type Runner interface {
	Run(ctx context.Context, command string, opts Options) int
}

type OSRunner struct{}

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
	devnullW, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if devnullW != nil {
		defer devnullW.Close()
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
	err := cmd.Run()
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
