//go:build windows

package ssh

import (
	"errors"
	"syscall"
)

func isConnectionRefused(err error) bool {
	const windowsConnectionRefused syscall.Errno = 10061
	return errors.Is(err, windowsConnectionRefused)
}
