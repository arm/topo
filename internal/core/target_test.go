package core_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	t.Run("run executes command successfully", func(t *testing.T) {
		mockExec := func(_, _ string) (string, error) {
			return "success", nil
		}
		target := core.MakeTarget("hostname", mockExec)

		out, err := target.Run("ls")

		assert.NoError(t, err)
		assert.Equal(t, "success", out)
	})

	t.Run("run returns error", func(t *testing.T) {
		mockExec := func(_, _ string) (string, error) {
			return "", errors.New("ssh failed")
		}
		target := core.MakeTarget("hostname", mockExec)

		out, err := target.Run("ls")

		assert.Error(t, err)
		assert.Empty(t, out)
	})
}

func TestMakeTarget(t *testing.T) {
	t.Run("make target succeeds and collects features", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			if command == "" {
				return "", nil // simulate successful initial connection
			}
			return "Features: fpu asimd", nil
		}

		target := core.MakeTarget("hostname", mockExec)

		assert.NoError(t, target.ConnectionError)
		assert.Equal(t, []string{"fpu", "asimd"}, target.Features)
	})

	t.Run("make target succeeds but fails to collect features", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			if command == "" {
				return "", nil
			}
			return "", nil
		}

		target := core.MakeTarget("hostname", mockExec)

		assert.NoError(t, target.ConnectionError)
		assert.Empty(t, target.Features)
	})

	t.Run("make target fails connection", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			return "", fmt.Errorf("connection refused")
		}

		target := core.MakeTarget("hostname", mockExec)

		assert.Error(t, target.ConnectionError)
		assert.EqualError(t, target.ConnectionError, "connection refused")
	})
}

func TestBinaryExists(t *testing.T) {
	t.Run("when binary found, returns true", func(t *testing.T) {
		mockExec := func(_, _ string) (string, error) {
			return "/foo/bar", nil
		}
		target := core.MakeTarget("hostname", mockExec)

		got, err := target.BinaryExists("bar")

		assert.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("invalid format returns an error", func(t *testing.T) {
		mockExec := func(_, _ string) (string, error) {
			return "/foo/bar", nil
		}
		target := core.MakeTarget("hostname", mockExec)

		got, err := target.BinaryExists("b a r")

		assert.Error(t, err)
		assert.False(t, got)
	})
}
