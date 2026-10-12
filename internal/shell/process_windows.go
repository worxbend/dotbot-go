package shell

import (
	"os/exec"
	"strconv"
)

func configureProcessCancellation(cmd *exec.Cmd) func() {
	cmd.Cancel = func() error {
		// taskkill /T includes descendants; /F handles shells that ignore signals.
		if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run(); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	return func() {}
}

func runCommand(cmd *exec.Cmd, _ func()) error { return cmd.Run() }
func processExitCode(err *exec.ExitError) int  { return err.ExitCode() }
