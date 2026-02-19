package target_test

import (
	"errors"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	t.Run("run executes command successfully", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "success", nil
		}
		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})

		out, err := conn.Run("ls")

		assert.NoError(t, err)
		assert.Equal(t, "success", out)
	})

	t.Run("run returns error", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "", errors.New("ssh failed")
		}
		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})

		out, err := conn.Run("ls")

		assert.Error(t, err)
		assert.Empty(t, out)
	})
}

func TestBinaryExists(t *testing.T) {
	t.Run("when binary found returns true", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "/foo/bar", nil
		}
		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})

		got, err := conn.BinaryExists("bar")

		assert.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("invalid format returns an error", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "/foo/bar", nil
		}
		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})

		got, err := conn.BinaryExists("b a r")

		assert.Error(t, err)
		assert.False(t, got)
	})
}
