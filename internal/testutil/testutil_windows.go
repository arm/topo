//go:build windows

package testutil

import (
	"syscall"
	"testing"
)

func IsPrivilegeError(t *testing.T, err error) bool {
	t.Helper()
	sysCallErr, ok := err.(syscall.Errno)
	return ok && sysCallErr == syscall.ERROR_PRIVILEGE_NOT_HELD
}
