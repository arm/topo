package command_test

import (
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
)

func TestLoginShellWrapper(t *testing.T) {
	t.Run("Wrap", func(t *testing.T) {
		t.Run("wraps the command in a login shell with markers", func(t *testing.T) {
			wrapper := command.NewLoginShellWrapperWithMarker("*FOO*")

			got := wrapper.Wrap("echo $PATH")

			want := `/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"printf '*FOO*\\n'; printf '*FOO*\\n' >&2; echo \\\$PATH\""`
			assert.Equal(t, want, got)
		})
	})

	t.Run("Unwrap", func(t *testing.T) {
		wrapper := command.NewLoginShellWrapperWithMarker("*FOO*")

		t.Run("strips output emitted before the marker", func(t *testing.T) {
			got := wrapper.Unwrap("login shell output\n*FOO*\nhello, some output")

			assert.Equal(t, "hello, some output", got)
		})

		t.Run("returns output without a marker unchanged", func(t *testing.T) {
			output := "ssh failed before starting the command"

			got := wrapper.Unwrap(output)

			assert.Equal(t, output, got)
		})

		t.Run("preserves subsequent markers in command output", func(t *testing.T) {
			got := wrapper.Unwrap("login shell output\n*FOO*\nfirst line\n*FOO*\nlast line")

			assert.Equal(t, "first line\n*FOO*\nlast line", got)
		})
	})
}
