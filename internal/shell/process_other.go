//go:build !unix && !windows

package shell

import "os/exec"

func configureProcessCancellation(cmd *exec.Cmd) func() {
	// Other platforms retain os/exec's direct-process cancellation.
	return func() {}
}

func runCommand(cmd *exec.Cmd, _ func()) error { return cmd.Run() }
func processExitCode(err *exec.ExitError) int  { return err.ExitCode() }
