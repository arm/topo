//go:build windows

package testutil

import "testing"

func acquireFlock(t *testing.T, lockPath string) func() {
	t.Helper()
	t.Fatal("file locking is not implemented on Windows")
	return func() {}
}
