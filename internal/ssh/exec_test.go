package ssh_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommandTrimsLoginShellOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX shell script")
	}

	binDir := t.TempDir()
	fakeSSH := filepath.Join(binDir, "ssh")
	err := os.WriteFile(fakeSSH, []byte(`#!/bin/sh
printf 'login shell stdout\n__TOPO_COMMAND_START__\ncommand stdout\n'
printf 'login shell stderr\n__TOPO_COMMAND_START__\ncommand stderr\n' >&2
exit "${FAKE_SSH_EXIT_STATUS:-0}"
`), 0o700)
	require.NoError(t, err)
	t.Setenv("PATH", binDir)

	t.Run("successful command", func(t *testing.T) {
		stdout, stderr, err := ssh.RunCommand(context.Background(), ssh.NewDestination("example.com"), "ignored", nil)

		require.NoError(t, err)
		assert.Equal(t, "command stdout\n", stdout)
		assert.Equal(t, "command stderr\n", stderr)
	})

	t.Run("failed command", func(t *testing.T) {
		t.Setenv("FAKE_SSH_EXIT_STATUS", "1")

		stdout, stderr, err := ssh.RunCommand(context.Background(), ssh.NewDestination("example.com"), "ignored", nil)

		require.Error(t, err)
		assert.Equal(t, "command stdout\n", stdout)
		assert.Equal(t, "command stderr\n", stderr)
		assert.NotContains(t, err.Error(), "login shell stderr")
		assert.Contains(t, err.Error(), "command stderr")
	})
}
