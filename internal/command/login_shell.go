package command

import (
	"fmt"
	"strings"
)

const loginShellMarker = "__TOPO_COMMAND_START__"

// LoginShellWrapper wraps commands for execution in a login shell and removes
// any output emitted by the shell before the command starts.
type LoginShellWrapper struct {
	marker string
}

// NewLoginShellWrapper returns a wrapper that marks the start of command output.
func NewLoginShellWrapper() *LoginShellWrapper {
	return NewLoginShellWrapperWithMarker(loginShellMarker)
}

// NewLoginShellWrapperWithMarker returns a wrapper that uses marker to identify
// the start of command output.
func NewLoginShellWrapperWithMarker(marker string) *LoginShellWrapper {
	return &LoginShellWrapper{marker: marker}
}

// Wrap returns cmd wrapped for execution in the user's login shell.
func (w *LoginShellWrapper) Wrap(cmd string) string {
	payload := fmt.Sprintf("printf '%s\\n'; printf '%s\\n' >&2; %s", w.marker, w.marker, cmd)
	escaped := shellEscapeForDoubleQuotes(payload)
	return fmt.Sprintf(`/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"%s\""`, escaped)
}

// Unwrap removes output emitted before the wrapped command starts.
func (w *LoginShellWrapper) Unwrap(output string) string {
	_, commandOutput, found := strings.Cut(output, w.marker+"\n")
	if !found {
		return output
	}
	return commandOutput
}

func shellEscapeForDoubleQuotes(s string) string {
	repl := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\\\"`,
		`$`, `\\\$`,
		"`", `\\\`+"`",
	)
	return repl.Replace(s)
}
