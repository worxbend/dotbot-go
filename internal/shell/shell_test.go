package shell

import (
	"context"
	"fmt"
	"runtime"
	"testing"
)

func TestShellCommandUsesPlatformShell(t *testing.T) {
	name, args := shellCommand("echo hello")
	if runtime.GOOS == "windows" {
		if name != "cmd" {
			t.Fatalf("name = %q, want cmd", name)
		}
		if len(args) != 2 || args[0] != "/C" || args[1] != "echo hello" {
			t.Fatalf("args = %#v, want [/C echo hello]", args)
		}
		return
	}

	t.Setenv("SHELL", "/bin/custom-sh")
	name, args = shellCommand("echo hello")
	if name != "/bin/custom-sh" {
		t.Fatalf("name = %q, want /bin/custom-sh", name)
	}
	if len(args) != 2 || args[0] != "-c" || args[1] != "echo hello" {
		t.Fatalf("args = %#v, want [-c echo hello]", args)
	}
}

func TestShellCommandFallsBackToSh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows uses cmd rather than SHELL")
	}
	t.Setenv("SHELL", "")

	name, args := shellCommand("echo hello")

	if name != "/bin/sh" {
		t.Fatalf("name = %q, want /bin/sh", name)
	}
	if len(args) != 2 || args[0] != "-c" || args[1] != "echo hello" {
		t.Fatalf("args = %#v, want [-c echo hello]", args)
	}
}

func TestOSRunnerRunReturnsExitCode(t *testing.T) {
	got := OSRunner{}.Run(context.Background(), exitCommand(7), Options{})

	if got != 7 {
		t.Fatalf("exit code = %d, want 7", got)
	}
}

func TestOSRunnerRunHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := OSRunner{}.Run(ctx, exitCommand(0), Options{})

	if got == 0 {
		t.Fatal("expected canceled context to fail command execution")
	}
}

func exitCommand(code int) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("exit /b %d", code)
	}
	return fmt.Sprintf("exit %d", code)
}
