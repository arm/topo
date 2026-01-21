//go:build windows

package testutil

import (
	"syscall"
	"testing"
)

func acquireFlock(t *testing.T, lockPath string) func() {
	t.Helper()
	t.Fatal("file locking is not implemented on Windows")
	return func() {}
}

func IsPrivilegeError(t *testing.T, err error) bool {
	t.Helper()
	sysCallErr, ok := err.(syscall.Errno)
	return ok && sysCallErr == syscall.ERROR_PRIVILEGE_NOT_HELD
}
