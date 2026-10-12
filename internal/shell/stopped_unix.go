//go:build linux || darwin

package shell

import (
	"syscall"
	"unsafe"
)

func processStopped(pid int) bool {
	// siginfo_t starts with si_signo on these platforms. The remaining fields
	// are opaque here; reserve enough space and alignment for every layout.
	info := struct {
		signo int32
		_     int32
		_     [15]uint64
	}{}
	const processID = 1 // POSIX P_PID
	_, _, errno := syscall.Syscall6(syscall.SYS_WAITID, processID, uintptr(pid), uintptr(unsafe.Pointer(&info)),
		uintptr(syscall.WSTOPPED|syscall.WNOHANG), 0, 0)
	return errno == 0 && info.signo != 0
}
