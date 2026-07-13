//go:build !windows

package ssh

import (
	"errors"
	"syscall"
)

func isConnectionRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}
