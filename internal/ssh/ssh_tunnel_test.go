package ssh_test

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHTunnel(t *testing.T) {
	t.Run("NewSSHTunnel", func(t *testing.T) {
		t.Run("it returns start and stop operations with control sockets", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")

			start, _, stop := ssh.NewSSHTunnel(dest, operation.DefaultRegistryPort, true)

			_, ok := start.(*ssh.SSHTunnelStart)
			assert.True(t, ok, "start operation is not of type SSHTunnelStart")
			_, ok = stop.(*ssh.SSHTunnelStop)
			assert.True(t, ok, "stop operation is not of type SSHTunnelStop")
		})

		t.Run("it returns start and stop operations without control sockets", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")

			start, _, stop := ssh.NewSSHTunnel(dest, operation.DefaultRegistryPort, false)

			_, ok := start.(*ssh.SSHTunnelStart)
			assert.True(t, ok, "start operation is not of type SSHTunnelStart")
			_, ok = stop.(*ssh.SSHTunnelProcessStop)
			assert.True(t, ok, "stop operation is not of type SSHTunnelProcessStop")
		})

		t.Run("stop operation has access to start operation process", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")

			start, _, stop := ssh.NewSSHTunnel(dest, operation.DefaultRegistryPort, false)
			startOp, ok := start.(*ssh.SSHTunnelStart)
			require.True(t, ok, "start operation is not of type SSHTunnelStart")

			stopOp, ok := stop.(*ssh.SSHTunnelProcessStop)
			require.True(t, ok, "stop operation is not of type SSHTunnelProcessStop")
			assert.Equal(t, startOp, stopOp.Start, "stop operation process does not match start operation process")
		})

		t.Run("it returns security check operation", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")

			_, securityCheck, _ := ssh.NewSSHTunnel(dest, operation.DefaultRegistryPort, true)

			_, ok := securityCheck.(*ssh.CheckSSHTunnelSecurity)
			assert.True(t, ok, "security check operation is not of type CheckSSHTunnelSecurity")
		})
	})
}

func TestSSHTunnelStart(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it generates correct ssh command", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")
			port := operation.DefaultRegistryPort

			st := ssh.NewSSHTunnelStart(dest, port, true)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -fMS %s -R %s:127.0.0.1:%s ssh://user@remote", ssh.ControlSocketPath(dest.String()), port, port)
			assert.Equal(t, want, got)
		})

		t.Run("it includes port flag when host has custom port", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote:2222")
			port := operation.DefaultRegistryPort

			st := ssh.NewSSHTunnelStart(dest, port, true)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -fMS %s -R %s:127.0.0.1:%s ssh://user@remote:2222", ssh.ControlSocketPath(dest.String()), port, port)
			assert.Equal(t, want, got)
		})

		t.Run("it does not include control socket flag when disabled", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")
			port := operation.DefaultRegistryPort

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
			st := ssh.NewSSHTunnelStart(ssh.NewDestination("user@remote"), operation.DefaultRegistryPort, true)
			var buf strings.Builder

			err := st.Run(&buf)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "opening SSH tunnel:")
			assert.NotContains(t, err.Error(), "port already in use")
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			st := ssh.NewSSHTunnelStart(ssh.NewDestination("user@remote"), operation.DefaultRegistryPort, true)

			got := st.Description()

			assert.Equal(t, "Open registry SSH tunnel", got)
		})
	})
}

func TestCheckSSHTunnelSecurity(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it generates correct curl command", func(t *testing.T) {
			dest := ssh.NewDestination("user@remote")
			port := operation.DefaultRegistryPort

			cs := ssh.NewCheckSSHTunnelSecurity(dest, port)
			got := strings.Join(cs.Command().Args, " ")

			want := fmt.Sprintf("curl remote:%s --max-time 1", port)
			assert.Equal(t, want, got)
		})

		t.Run("it returns nil when target is localhost", func(t *testing.T) {
			dest := ssh.NewDestination("root@localhost")
			port := operation.DefaultRegistryPort

			cs := ssh.NewCheckSSHTunnelSecurity(dest, port)
			got := cs.Command()
			assert.Nil(t, got)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns the expected string", func(t *testing.T) {
			cs := ssh.NewCheckSSHTunnelSecurity(ssh.NewDestination("user@remote"), operation.DefaultRegistryPort)

			got := cs.Description()

			assert.Equal(t, "Check SSH tunnel security", got)
		})
	})
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
