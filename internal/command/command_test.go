package command_test

import (
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
)

func TestWrapInLoginShell(t *testing.T) {
	t.Run("wraps command in login shell", func(t *testing.T) {
		got := command.WrapInLoginShell("echo $PATH")

		want := `/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"echo \\\$PATH\""`
		assert.Equal(t, want, got)
	})
}

func TestBinaryLookupCommand(t *testing.T) {
	t.Run("returns error for invalid binary", func(t *testing.T) {
		got, err := command.BinaryLookupCommand("bad name")

		assert.Error(t, err)
		assert.Empty(t, got)
	})
}
