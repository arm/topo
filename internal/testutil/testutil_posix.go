//go:build !windows

package testutil

import (
	"testing"
)

func SetHomeDir(t testing.TB, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
}

func IsPrivilegeError(t *testing.T, err error) bool {
	t.Helper()
	return false
}
