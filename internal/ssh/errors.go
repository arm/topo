package ssh

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrSSH              = errors.New("SSH failed")
	ErrAuthFailed       = fmt.Errorf("%w: authentication failed", ErrSSH)
	ErrConnectionFailed = fmt.Errorf("%w: connection failed", ErrSSH)
)

// ClassifyStderr inspects SSH stderr output and returns a typed error when a
// known failure pattern is detected, or nil if the output is unrecognised.
func ClassifyStderr(stderr string) error {
	lower := strings.ToLower(stderr)
	if strings.Contains(lower, "publickey") ||
		strings.Contains(lower, "authentication") {
		return ErrAuthFailed
	}
	if strings.Contains(lower, "connection refused") {
		return ErrConnectionFailed
	}
	return nil
}
