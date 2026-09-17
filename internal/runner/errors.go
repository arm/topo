package runner

import "fmt"

type CommandError struct {
	Command string
	Stderr  string
	Err     error
}

func (err *CommandError) Error() string {
	return fmt.Sprintf("command failed: %v", err.Err)
}

func (err *CommandError) Unwrap() error {
	return err.Err
}
