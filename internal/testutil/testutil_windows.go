//go:build windows

package testutil

import (
	"os"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

func IsPrivilegeError(t *testing.T, err error) bool {
	t.Helper()
	sysCallErr, ok := err.(syscall.Errno)
	return ok && sysCallErr == syscall.ERROR_PRIVILEGE_NOT_HELD
}

func lockFile(file *os.File) error {
	return windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &windows.Overlapped{})
}
