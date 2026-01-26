//go:build !windows

package testutil

import (
	"testing"
)

func IsPrivilegeError(t *testing.T, err error) bool {
	t.Helper()
	return false
}
