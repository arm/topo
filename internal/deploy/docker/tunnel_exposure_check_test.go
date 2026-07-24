package docker_test

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckTunnelExposure(t *testing.T) {
	t.Run("fails when the SSH hostname cannot be resolved", func(t *testing.T) {
		err := docker.CheckTunnelExposure(context.Background(), io.Discard, ssh.Destination{
			Host: "not-a-host",
		}, "12345")

		assert.ErrorContains(t, err, "cannot conclusively rule out network access to registry port 12345")
		assert.ErrorContains(t, err, `Could not resolve hostname not-a-host`)
		assert.ErrorContains(t, err, "use `--skip-remote-port-check` if you understand the security risk")
	})

	t.Run("succeeds when the remote port refuses the connection", func(t *testing.T) {
		target := testutil.StartContainer(t, testutil.SshServerContainer)
		dest := ssh.NewDestination(target.SSHDestination)
		port := "12345"
		openSocket(t, dest, "127.0.0.1", port)
		var output strings.Builder

		err := docker.CheckTunnelExposure(context.Background(), &output, dest, port)

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Registry port %s is bound to remote loopback only\n", port), output.String())
	})

	t.Run("fails when the remote port accepts a connection", func(t *testing.T) {
		target := testutil.StartContainer(t, testutil.SshServerContainer)
		dest := ssh.NewDestination(target.SSHDestination)
		port := "12345"
		openSocket(t, dest, "0.0.0.0", port)

		err := docker.CheckTunnelExposure(context.Background(), io.Discard, dest, port)

		assert.EqualError(t, err, fmt.Sprintf("the remote SSH server is exposing forwarded registry port %s beyond remote loopback; configure the SSH server to bind remote forwards to loopback only, or use `--skip-remote-port-check` if you understand that the registry may be reachable without SSH authentication", port))
	})
}

func openSocket(t *testing.T, dest ssh.Destination, address string, port string) {
	t.Helper()
	cmd := exec.Command("ssh", dest.String(), fmt.Sprintf("nc -lk -p %s -s %s -e cat", port, address))
	err := cmd.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})

	err = waitForOpenSocket(dest, address, port)
	require.NoError(t, err)
}

func waitForOpenSocket(dest ssh.Destination, address string, port string) error {
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		// #nosec G204 -- ignore as its a test helper
		cmd := exec.Command("ssh", dest.String(), fmt.Sprintf("nc -z -w 1 %s %s", address, port))
		output, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("%w output: %s", err, strings.TrimSpace(string(output)))
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("port %s not ready: %w", port, lastErr)
}
