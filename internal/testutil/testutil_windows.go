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

func AcquireFlock(t *testing.T, path string) func() {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("failed to open lock file: %v", err)
	}

	err = windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		1, 0,
		&windows.Overlapped{},
	)
	if err != nil {
		_ = f.Close()
		t.Fatalf("failed to acquire lock: %v", err)
	}

	return func() {
		_ = windows.UnlockFileEx(
			windows.Handle(f.Fd()),
			0,
			1, 0,
			&windows.Overlapped{},
		)
		_ = f.Close()
	}
}
