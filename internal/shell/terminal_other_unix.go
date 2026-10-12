//go:build unix && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package shell

import (
	"os"
)

func terminalProcessGroup(*os.File) (int, error)  { return 0, os.ErrInvalid }
func setTerminalProcessGroup(*os.File, int) error { return os.ErrInvalid }
