//go:build windows

package testutil

import (
	"path/filepath"
	"syscall"
	"testing"
)

func SetHomeDir(t testing.TB, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	volume := filepath.VolumeName(dir)
	if volume == "" {
		return
	}
	path := dir[len(volume):]
	t.Setenv("HOMEDRIVE", volume)
	if path == "" {
		path = "\\"
	}
	t.Setenv("HOMEPATH", path)
}

func IsPrivilegeError(t *testing.T, err error) bool {
	t.Helper()
	sysCallErr, ok := err.(syscall.Errno)
	return ok && sysCallErr == syscall.ERROR_PRIVILEGE_NOT_HELD
}
