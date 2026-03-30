package command_test

import (
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapInLoginShell(t *testing.T) {
	t.Run("wraps command in login shell", func(t *testing.T) {
		got := command.WrapInLoginShell("echo $PATH")

		want := `/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"echo \\\$PATH\""`
		assert.Equal(t, want, got)
	})
}

func TestBinaryLookupCommand(t *testing.T) {
	t.Run("returns wrapped command for valid binary", func(t *testing.T) {
		got, err := command.BinaryLookupCommand("docker")

		require.NoError(t, err)
		assert.Equal(t, command.UnsafeBinaryLookupCommand("docker"), got)
	})

	t.Run("returns error for invalid binary", func(t *testing.T) {
		got, err := command.BinaryLookupCommand("bad name")

		assert.Error(t, err)
		assert.Empty(t, got)
	})
}

func TestUnsafeBinaryLookupCommand(t *testing.T) {
	t.Run("returns wrapped command without validation", func(t *testing.T) {
		got := command.UnsafeBinaryLookupCommand("docker")

		assert.Equal(t, command.WrapInLoginShell("command -v docker"), got)
	})
}
