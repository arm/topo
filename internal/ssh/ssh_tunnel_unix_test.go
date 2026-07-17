//go:build unix

package ssh_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHTunnel(t *testing.T) {
	t.Run("opens and closes through a control socket", func(t *testing.T) {
		logPath := installFakeSSH(t)
		destination := ssh.NewDestination("user@remote")

		tunnel, openErr := ssh.OpenSSHTunnel(context.Background(), io.Discard, destination, "1337", true)
		require.NoError(t, openErr)
		closeErr := tunnel.Close(context.Background(), io.Discard)

		require.NoError(t, closeErr)
		invocations := readSSHInvocations(t, logPath, 2)
		want := []string{
			fmt.Sprintf("-N -o ExitOnForwardFailure=yes -fMS %s -R 127.0.0.1:1337:127.0.0.1:1337 %s", ssh.ControlSocketPath(destination.String()), destination.String()),
			fmt.Sprintf("-S %s -O exit %s", ssh.ControlSocketPath(destination.String()), destination.String()),
		}
		assert.Equal(t, want, invocations)
	})

	t.Run("writes stdout and stderr to supplied writer", func(t *testing.T) {
		installFakeSSH(t)
		destination := ssh.NewDestination("user@remote")
		var output bytes.Buffer

		tunnel, err := ssh.OpenSSHTunnel(context.Background(), &output, destination, "1337", true)

		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tunnel.Close(context.Background(), io.Discard))
		})
		assert.Contains(t, output.String(), "ssh stdout")
		assert.Contains(t, output.String(), "ssh stderr")
	})

	t.Run("closes a running tunnel process", func(t *testing.T) {
		logPath := installFakeSSH(t)
		destination := ssh.NewDestination("user@remote")

		tunnel, openErr := ssh.OpenSSHTunnel(context.Background(), io.Discard, destination, "1338", false)
		require.NoError(t, openErr)
		pid := readSSHPID(t, logPath+".pid")
		closeErr := tunnel.Close(context.Background(), io.Discard)

		require.NoError(t, closeErr)
		// ESRCH means no process exists with the supplied PID.
		processErr := syscall.Kill(pid, 0)
		assert.ErrorIs(t, processErr, syscall.ESRCH)
	})

	t.Run("close is idempotent", func(t *testing.T) {
		logPath := installFakeSSH(t)
		destination := ssh.NewDestination("user@remote")

		tunnel, openErr := ssh.OpenSSHTunnel(context.Background(), nil, destination, "1339", true)
		require.NoError(t, openErr)
		firstCloseErr := tunnel.Close(context.Background(), nil)
		secondCloseErr := tunnel.Close(context.Background(), nil)

		require.NoError(t, firstCloseErr)
		require.NoError(t, secondCloseErr)
		invocations := readSSHInvocations(t, logPath, 2)
		assert.Len(t, invocations, 2)
	})
}

func installFakeSSH(t *testing.T) string {
	t.Helper()

	directory := t.TempDir()
	logPath := filepath.Join(directory, "ssh.log")
	fakeSSHBin := filepath.Join(directory, "ssh")
	const script = `#!/bin/sh
printf '%s\n' "$*" >> "$TOPO_TEST_SSH_LOG"
printf 'ssh stdout\n'
printf 'ssh stderr\n' >&2
case " $* " in
  *" -O exit "*) exit 0 ;;
  *" -fMS "*) exit 0 ;;
  *)
    printf '%s\n' "$$" > "$TOPO_TEST_SSH_LOG.pid"
    exec sleep 3600
    ;;
esac
`

	require.NoError(t, os.WriteFile(fakeSSHBin, []byte(script), 0o700))
	t.Setenv("TOPO_TEST_SSH_LOG", logPath)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func readSSHPID(t *testing.T, pidPath string) int {
	t.Helper()

	var pid int
	require.Eventually(t, func() bool {
		content, err := os.ReadFile(pidPath)
		if err != nil {
			return false
		}
		pid, err = strconv.Atoi(strings.TrimSpace(string(content)))
		return err == nil
	}, time.Second, 10*time.Millisecond)
	return pid
}

func readSSHInvocations(t *testing.T, logPath string, count int) []string {
	t.Helper()

	var invocations []string
	require.Eventually(t, func() bool {
		content, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		invocations = strings.Split(strings.TrimSpace(string(content)), "\n")
		return len(invocations) >= count
	}, time.Second, 10*time.Millisecond)
	return invocations
}
