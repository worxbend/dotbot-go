//go:build linux && integration

package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestOSRunnerPreservesInteractiveTerminal(t *testing.T) {
	for _, name := range []string{"input", "interrupt", "suspend"} {
		t.Run(name, func(t *testing.T) { testInteractiveTerminal(t, name) })
	}
}

func testInteractiveTerminal(t *testing.T, scenario string) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = master.Close() }()
	var unlock int32
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); errno != 0 {
		t.Fatal(errno)
	}
	var number uint32
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&number))); errno != 0 {
		t.Fatal(errno)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = slave.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInteractiveShellHelper$")
	dir := t.TempDir()
	cmd.Env = append(os.Environ(), "DOTBOT_SHELL_PTY_HELPER=1", "DOTBOT_SHELL_TEST_DIR="+dir,
		"DOTBOT_SHELL_TEST_CASE="+scenario, "SHELL=/bin/sh")
	cmd.Stdin = slave
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(dir + "/ready"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("interactive shell did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if scenario == "interrupt" {
		if _, err := master.Write([]byte{3}); err != nil {
			t.Fatal(err)
		}
	} else if scenario == "suspend" {
		if _, err := master.Write([]byte{26}); err != nil {
			t.Fatal(err)
		}
		for !processStopped(cmd.Process.Pid) {
			if time.Now().After(deadline) {
				t.Fatal("CLI did not suspend with its interactive child")
			}
			time.Sleep(10 * time.Millisecond)
		}
		if err := syscall.Kill(cmd.Process.Pid, syscall.SIGCONT); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := io.WriteString(master, "hello\n"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("interactive command failed: %v; %s", err, stderr.String())
	}
}

func TestInteractiveShellHelper(t *testing.T) {
	if os.Getenv("DOTBOT_SHELL_PTY_HELPER") != "1" {
		return
	}
	before, err := terminalProcessGroup(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	code := (OSRunner{}).Run(context.Background(), `echo ready > ready; read answer; test "$answer" = hello`, Options{
		CWD:         os.Getenv("DOTBOT_SHELL_TEST_DIR"),
		EnableStdin: true,
		Timeout:     2 * time.Second,
	})
	want := 0
	if os.Getenv("DOTBOT_SHELL_TEST_CASE") == "interrupt" {
		want = 130
	}
	if code != want {
		t.Fatalf("terminal read exit code = %d", code)
	}
	after, err := terminalProcessGroup(os.Stdin)
	if err != nil || after != before {
		t.Fatalf("terminal ownership not restored: before = %d, after = %d, error = %v", before, after, err)
	}
}
