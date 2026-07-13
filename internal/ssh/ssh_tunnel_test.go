package ssh_test

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHTunnel(t *testing.T) {
	t.Run("NewSSHTunnel", func(t *testing.T) {
		t.Run("it returns start and stop operations with control sockets", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")

			start, _, stop := ssh.NewSSHTunnel(dest, "91232", true)

			_, ok := start.(*ssh.SSHTunnelStart)
			assert.True(t, ok, "start operation is not of type SSHTunnelStart")
			_, ok = stop.(*ssh.SSHTunnelStop)
			assert.True(t, ok, "stop operation is not of type SSHTunnelStop")
		})

		t.Run("it returns start and stop operations without control sockets", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")

			start, _, stop := ssh.NewSSHTunnel(dest, "12201", false)

			_, ok := start.(*ssh.SSHTunnelStart)
			assert.True(t, ok, "start operation is not of type SSHTunnelStart")
			_, ok = stop.(*ssh.SSHTunnelProcessStop)
			assert.True(t, ok, "stop operation is not of type SSHTunnelProcessStop")
		})

		t.Run("stop operation has access to start operation process", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")

			start, _, stop := ssh.NewSSHTunnel(dest, "07070", false)
			startOp, ok := start.(*ssh.SSHTunnelStart)
			require.True(t, ok, "start operation is not of type SSHTunnelStart")

			stopOp, ok := stop.(*ssh.SSHTunnelProcessStop)
			require.True(t, ok, "stop operation is not of type SSHTunnelProcessStop")
			assert.Equal(t, startOp, stopOp.Start, "stop operation process does not match start operation process")
		})

		t.Run("it returns security check operation", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")

			_, securityCheck, _ := ssh.NewSSHTunnel(dest, "44553", true)

			_, ok := securityCheck.(*ssh.CheckRemoteForwardNotExposed)
			assert.True(t, ok, "security check operation is not of type CheckRemoteForwardNotExposed")
		})
	})
}

func TestSSHTunnelStart(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it generates correct ssh command", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")
			port := "1337"

			st := ssh.NewSSHTunnelStart(dest, port, true)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -fMS %s -R 127.0.0.1:%s:127.0.0.1:%s ssh://user@remote", ssh.ControlSocketPath(dest.String()), port, port)
			assert.Equal(t, want, got)
		})

		t.Run("it does not include control socket flag when disabled", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")
			port := "1338"

			st := ssh.NewSSHTunnelStart(dest, port, false)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -R 127.0.0.1:%s:127.0.0.1:%s ssh://user@remote", port, port)
			assert.Equal(t, want, got)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			st := ssh.NewSSHTunnelStart(ssh.NewDestination("user@remote"), "12345", true)

			got := st.Description()

			assert.Equal(t, "Open registry SSH tunnel", got)
		})
	})
}

func TestCheckRemoteForwardNotExposed(t *testing.T) {
	t.Run("Description", func(t *testing.T) {
		t.Run("it returns the expected string", func(t *testing.T) {
			cs := ssh.NewCheckRemoteForwardNotExposed(ssh.NewDestination("user@remote"), "12345")

			got := cs.Description()

			assert.Equal(t, "Check tunnel port is not exposed on remote network", got)
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("it skips the check for localhost", func(t *testing.T) {
			check := ssh.NewCheckRemoteForwardNotExposed(ssh.PlainLocalhost, "invalid")
			var output strings.Builder

			err := check.Run(&output)

			assert.NoError(t, err)
			assert.Empty(t, output.String())
		})

		t.Run("it fails when the SSH hostname cannot be resolved", func(t *testing.T) {
			check := ssh.NewCheckRemoteForwardNotExposed(ssh.Destination{}, "12345")

			err := check.Run(io.Discard)

			assert.ErrorContains(t, err, `could not resolve SSH hostname for "ssh://"`)
			assert.ErrorContains(t, err, "use `--skip-remote-port-check` if you understand the security risk")
		})

		t.Run("it succeeds when the remote port refuses the connection", func(t *testing.T) {
			port := reserveFreePort(t, "0.0.0.0")
			check := ssh.NewCheckRemoteForwardNotExposed(ssh.NewDestination("0.0.0.0"), port)
			var output strings.Builder

			err := check.Run(&output)

			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("Port %s is bound to remote loopback only\n", port), output.String())
		})

		t.Run("it fails when the remote port accepts a connection", func(t *testing.T) {
			listener, err := net.Listen("tcp", "0.0.0.0:0")
			require.NoError(t, err)
			t.Cleanup(func() { _ = listener.Close() })
			_, port, err := net.SplitHostPort(listener.Addr().String())
			require.NoError(t, err)
			check := ssh.NewCheckRemoteForwardNotExposed(ssh.NewDestination("0.0.0.0"), port)

			err = check.Run(io.Discard)

			assert.EqualError(t, err, fmt.Sprintf("remote sshd might be exposing the forwarded port %s on its network (likely GatewayPorts=yes); the local registry may be reachable without SSH auth", port))
			assert.NotContains(t, err.Error(), "--skip-remote-port-check")
		})
	})
}

func reserveFreePort(t *testing.T, host string) string {
	t.Helper()
	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	require.NoError(t, listener.Close())
	return port
}

func TestSSHTunnelStop(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it generates correct ssh command", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")

			st := ssh.NewSSHTunnelStop(dest)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -S %s -O exit ssh://user@remote", ssh.ControlSocketPath(dest.String()))
			assert.Equal(t, want, got)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			st := ssh.NewSSHTunnelStop(ssh.NewDestination("user@remote"))

			got := st.Description()

			assert.Equal(t, "Close registry SSH tunnel", got)
		})
	})
}

func TestSSHTunnelProcessStop(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("windows", func(t *testing.T) {
			testutil.RequireOS(t, "windows")

			t.Run("it generates correct kill command without target process", func(t *testing.T) {
				st := ssh.NewSSHTunnelProcessStop(nil)
				got := strings.Join(st.Command().Args, " ")

				want := fmt.Sprintf("taskkill /PID %s /F", ssh.TunnelPIDPlaceholder)
				assert.Equal(t, want, got)
			})

			t.Run("it generates correct kill command with target process", func(t *testing.T) {
				start := &ssh.SSHTunnelStart{Process: &os.Process{Pid: 12345}}

				st := ssh.NewSSHTunnelProcessStop(start)
				got := strings.Join(st.Command().Args, " ")

				want := fmt.Sprintf("taskkill /PID %d /F", start.Process.Pid)
				assert.Equal(t, want, got)
			})
		})

		t.Run("linux", func(t *testing.T) {
			testutil.RequireOS(t, "linux")

			t.Run("it generates correct kill command without target process", func(t *testing.T) {
				st := ssh.NewSSHTunnelProcessStop(nil)
				got := strings.Join(st.Command().Args, " ")

				want := fmt.Sprintf("kill -9 %s", ssh.TunnelPIDPlaceholder)
				assert.Equal(t, want, got)
			})

			t.Run("it generates correct kill command with target process", func(t *testing.T) {
				start := &ssh.SSHTunnelStart{Process: &os.Process{Pid: 12345}}

				st := ssh.NewSSHTunnelProcessStop(start)
				got := strings.Join(st.Command().Args, " ")

				want := fmt.Sprintf("kill -9 %d", start.Process.Pid)
				assert.Equal(t, want, got)
			})
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			st := ssh.NewSSHTunnelProcessStop(nil)

			got := st.Description()

			assert.Equal(t, "Close registry SSH tunnel", got)
		})
	})
}
