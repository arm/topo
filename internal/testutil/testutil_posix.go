//go:build !windows

package testutil

import (
	"os"
	"syscall"
	"testing"
)

func IsPrivilegeError(t *testing.T, err error) bool {
	t.Helper()
	return false
}

func lockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX) // #nosec G115 -- POSIX file descriptors fit in int.
}
