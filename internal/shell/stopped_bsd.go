//go:build freebsd || netbsd || openbsd || dragonfly

package shell

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func processStopped(pid int) bool {
	// BSD wait interfaces differ from Linux and Darwin. Query the process state
	// without consuming the exit status owned by exec.Cmd.Wait.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	state, err := exec.CommandContext(ctx, "/bin/ps", "-o", "state=", "-p", strconv.Itoa(pid)).Output()
	return err == nil && strings.HasPrefix(strings.TrimSpace(string(state)), "T")
}
