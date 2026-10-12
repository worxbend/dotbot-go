//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package shell

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

func terminalProcessGroup(file *os.File) (int, error) {
	var group int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), syscall.TIOCGPGRP, uintptr(unsafe.Pointer(&group)))
	runtime.KeepAlive(file)
	if errno != 0 {
		return 0, errno
	}
	return int(group), nil
}

func setTerminalProcessGroup(file *os.File, group int) error {
	value := int32(group)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&value)))
	runtime.KeepAlive(file)
	if errno != 0 {
		return errno
	}
	return nil
}
