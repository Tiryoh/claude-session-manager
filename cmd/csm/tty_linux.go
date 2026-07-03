//go:build linux

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// stdinIsTerminal reports whether stdin is an interactive terminal rather
// than a pipe, redirected file, or another character device such as
// /dev/null (which a plain os.ModeCharDevice check cannot distinguish from
// a real TTY). It issues the same TCGETS ioctl the standard terminal
// helpers use, without pulling in an external dependency.
func stdinIsTerminal() bool {
	var termios syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&termios)))
	return errno == 0
}
