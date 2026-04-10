package ssh

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrSSH               = errors.New("SSH failed")
	ErrAuthFailed        = fmt.Errorf("%w: authentication failed", ErrSSH)
	ErrConnectionFailed  = fmt.Errorf("%w: connection failed", ErrSSH)
	ErrConnectionTimeout = fmt.Errorf("%w: connection timed out", ErrSSH)
	ErrHostKeyUnknown    = fmt.Errorf("%w: host key is not known", ErrSSH)
	ErrHostKeyChanged    = fmt.Errorf("%w: host key has changed", ErrSSH)
)

// ClassifyStderr inspects SSH stderr output and returns a typed error when a
// known failure pattern is detected, or nil if the output is unrecognised.
func ClassifyStderr(stderr string) error {
	lower := strings.ToLower(stderr)
	if strings.Contains(lower, "host key verification failed") {
		if strings.Contains(lower, "has changed") {
			return ErrHostKeyChanged
		}
		return ErrHostKeyUnknown
	}
	if strings.Contains(lower, "timed out") ||
		strings.Contains(lower, "connection timeout") ||
		strings.Contains(lower, "did not properly respond after a period of time") {
		return ErrConnectionTimeout
	}
	if strings.Contains(lower, "publickey") ||
		strings.Contains(lower, "authentication") ||
		strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "password") {
		return ErrAuthFailed
	}
	if strings.Contains(lower, "connection refused") {
		return ErrConnectionFailed
	}
	return nil
}
