package ssh_test

import (
	"fmt"
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

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -fMS %s -R %s:127.0.0.1:%s ssh://user@remote", ssh.ControlSocketPath(dest.String()), port, port)
			assert.Equal(t, want, got)
		})

		t.Run("it does not include control socket flag when disabled", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")
			port := "1338"

			st := ssh.NewSSHTunnelStart(dest, port, false)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -R %s:127.0.0.1:%s ssh://user@remote", port, port)
			assert.Equal(t, want, got)
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("it returns port in use error when port is taken", func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close() //nolint:errcheck
			_, port, err := net.SplitHostPort(listener.Addr().String())
			require.NoError(t, err)
			var buf strings.Builder
			st := ssh.NewSSHTunnelStart(ssh.NewDestination("user@remote"), port, true)

			err = st.Run(&buf)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "port already in use: "+port)
		})

		t.Run("it returns generic tunnel error when port is free", func(t *testing.T) {
			st := ssh.NewSSHTunnelStart(ssh.NewDestination("user@remote"), "99010", true)
			var buf strings.Builder

			err := st.Run(&buf)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to open ssh tunnel:")
			assert.NotContains(t, err.Error(), "port already in use")
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
}

func TestIsRemotePortDefinitelyNotListening(t *testing.T) {
	t.Run("it returns true when nothing answers on the remote port", func(t *testing.T) {
		port := reserveFreePort(t)

		assert.True(t, ssh.IsRemotePortDefinitelyNotListening("127.0.0.1", port))
	})

	t.Run("it returns false when a TCP listener is bound on the remote port", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = listener.Close() })
		_, port, err := net.SplitHostPort(listener.Addr().String())
		require.NoError(t, err)

		assert.False(t, ssh.IsRemotePortDefinitelyNotListening("127.0.0.1", port))
	})

	t.Run("it returns false when curl exits with any other non-7 error", func(t *testing.T) {
		// `.invalid` is reserved by RFC 6761 to never resolve, so curl
		// exits 6 ("Couldn't resolve host") rather than 7. We can't
		// certify the tunnel is safe, so we must fail.
		assert.False(t, ssh.IsRemotePortDefinitelyNotListening("nonexistent.invalid", "12345"))
	})
}

func reserveFreePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
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
