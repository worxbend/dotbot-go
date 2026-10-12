//go:build unix

package shell

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func configureProcessCancellation(cmd *exec.Cmd) func() {
	// A separate process group keeps cancellation from killing the CLI itself
	// and includes children launched by the shell.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	restore := func() {}
	if stdin, ok := cmd.Stdin.(*os.File); ok {
		if group, err := terminalProcessGroup(stdin); err == nil {
			// Interactive children must own the terminal while they run; otherwise
			// a terminal read stops the new group with SIGTTIN.
			cmd.SysProcAttr.Foreground = true
			cmd.SysProcAttr.Ctty = int(stdin.Fd())
			restore = func() {
				if cmd.Process == nil {
					return
				}
				current, err := terminalProcessGroup(stdin)
				if err != nil || current != cmd.Process.Pid {
					return
				}
				ignored := signal.Ignored(syscall.SIGTTOU)
				if !ignored {
					signal.Ignore(syscall.SIGTTOU)
					defer signal.Reset(syscall.SIGTTOU)
				}
				if err := setTerminalProcessGroup(stdin, group); err != nil {
					// The terminal may have closed while the command was running.
					return
				}
			}
		}
	}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	return restore
}

func processExitCode(err *exec.ExitError) int {
	if status, ok := err.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return err.ExitCode()
}

func runCommand(cmd *exec.Cmd, restoreTerminal func()) error {
	if !cmd.SysProcAttr.Foreground {
		return cmd.Run()
	}
	// Observe stopped children without reaping them; exec.Cmd retains ownership
	// of the exit status. Suspend the CLI too so the invoking shell can manage
	// the entire interactive job with Ctrl+Z, fg and bg.
	changes := make(chan os.Signal, 1)
	signal.Notify(changes, syscall.SIGCHLD)
	defer signal.Stop(changes)
	if err := cmd.Start(); err != nil {
		return err
	}
	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()
	cancelAndWait := func(err error) error {
		cancelErr := cmd.Cancel()
		return errors.Join(err, cancelErr, <-finished)
	}
	stdin := cmd.Stdin.(*os.File)
	for {
		select {
		case err := <-finished:
			return err
		case <-changes:
			if !processStopped(cmd.Process.Pid) {
				continue
			}
			if group, err := terminalProcessGroup(stdin); err == nil && group == cmd.Process.Pid {
				restoreTerminal()
			}
			if err := syscall.Kill(os.Getpid(), syscall.SIGSTOP); err != nil {
				continue
			}
			// On fg, return terminal ownership to the child. On bg, leave the
			// foreground shell alone; a later terminal read will stop this job.
			if group, err := terminalProcessGroup(stdin); err == nil && group == syscall.Getpgrp() {
				if err := setTerminalProcessGroup(stdin, cmd.Process.Pid); err != nil {
					return cancelAndWait(err)
				}
			}
			if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGCONT); err != nil && !errors.Is(err, syscall.ESRCH) {
				return cancelAndWait(err)
			}
		}
	}
}
