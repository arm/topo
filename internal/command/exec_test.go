package command_test

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommand(t *testing.T) {
	t.Run("captures stdout in writer on success", func(t *testing.T) {
		cmd := exec.Command("echo", "hello")
		var buf bytes.Buffer

		err := command.RunCommand(cmd, &buf)

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "hello")
	})

	t.Run("returns error with stderr on failure", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "echo oops >&2; exit 1")
		var buf bytes.Buffer

		err := command.RunCommand(cmd, &buf)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed:")
		assert.Contains(t, err.Error(), "stderr: oops")
	})

	t.Run("includes command args in error", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "exit 1")
		var buf bytes.Buffer

		err := command.RunCommand(cmd, &buf)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "sh -c exit 1")
	})
}

func TestStartCommand(t *testing.T) {
	t.Run("starts command without blocking", func(t *testing.T) {
		cmd := exec.Command("echo", "hello")
		var buf bytes.Buffer

		err := command.StartCommand(cmd, &buf)

		require.NoError(t, err)
		assert.NotNil(t, cmd.Process)
		_ = cmd.Wait()
	})

	t.Run("returns error for invalid command containing binary name", func(t *testing.T) {
		cmd := exec.Command("nonexistent-binary-xyz")
		var buf bytes.Buffer

		err := command.StartCommand(cmd, &buf)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent-binary-xyz")
	})
}
