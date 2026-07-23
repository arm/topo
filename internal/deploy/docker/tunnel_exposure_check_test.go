package docker_test

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"

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
		listenOnPort(t, dest, "127.0.0.1", port)
		var output strings.Builder

		err := docker.CheckTunnelExposure(context.Background(), &output, dest, port)

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Registry port %s is bound to remote loopback only\n", port), output.String())
	})

	t.Run("fails when the remote port accepts a connection", func(t *testing.T) {
		target := testutil.StartContainer(t, testutil.SshServerContainer)
		dest := ssh.NewDestination(target.SSHDestination)
		port := "12345"
		listenOnPort(t, dest, "0.0.0.0", port)

		err := docker.CheckTunnelExposure(context.Background(), io.Discard, dest, port)

		assert.EqualError(t, err, fmt.Sprintf("the remote SSH server is exposing forwarded registry port %s beyond remote loopback; configure the SSH server to bind remote forwards to loopback only, or use `--skip-remote-port-check` if you understand that the registry may be reachable without SSH authentication", port))
	})
}

func listenOnPort(t *testing.T, dest ssh.Destination, address string, port string) {
	t.Helper()
	cmd := exec.Command("ssh", dest.String(), fmt.Sprintf("nc -l -p %s -s %s", port, address))
	err := cmd.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})
}
