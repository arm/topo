package ssh

import (
	"errors"
	"strings"
)

type sshError struct {
	message string
}

func (err sshError) Error() string {
	return err.message
}

func (err sshError) Is(target error) bool {
	return target == ErrSSH
}

var (
	ErrSSH               = errors.New("ssh failed")
	ErrAuthFailed        = sshError{message: "authentication failed"}
	ErrTooManyAuthFails  = sshError{message: "too many authentication failures"}
	ErrConnectionFailed  = sshError{message: "connection failed"}
	ErrConnectionTimeout = sshError{message: "connection timed out"}
	ErrHostKeyUnknown    = sshError{message: "host key is not known"}
	ErrHostKeyChanged    = sshError{message: "host key has changed"}
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
	if strings.Contains(lower, "timed out") {
		return ErrConnectionTimeout
	}
	if strings.Contains(lower, "permission denied") {
		return ErrAuthFailed
	}
	if strings.Contains(lower, "too many authentication failures") {
		return ErrTooManyAuthFails
	}
	if strings.Contains(lower, "connection refused") {
		return ErrConnectionFailed
	}
	if strings.Contains(lower, "could not resolve hostname") {
		return ErrConnectionFailed
	}
	return nil
}
