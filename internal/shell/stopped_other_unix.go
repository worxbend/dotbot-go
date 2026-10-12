//go:build unix && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package shell

func processStopped(int) bool { return false }
