package command

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Error struct {
	Command string
	Stderr  string
	Err     error
}

func (err *Error) Error() string {
	return fmt.Sprintf("command %q failed: %v", err.Command, err.Err)
}

func (err *Error) Unwrap() error {
	return err.Err
}

func NewError(cmd *exec.Cmd, err error) *Error {
	stderr := ""
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		stderr = string(exitErr.Stderr)
	}
	return &Error{
		Command: strings.Join(cmd.Args, " "),
		Err:     err,
		Stderr:  stderr,
	}
}
