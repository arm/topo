package ssh_test

import (
	"bytes"
	"fmt"
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
			host := ssh.SSHDestination("user@remote")

			start, _, stop := ssh.NewSSHTunnel(host, operation.DefaultRegistryPort, true)

			_, ok := start.(*ssh.SSHTunnelStart)
			assert.True(t, ok, "start operation is not of type SSHTunnelStart")
			_, ok = stop.(*ssh.SSHTunnelStop)
			assert.True(t, ok, "stop operation is not of type SSHTunnelStop")
		})

		t.Run("it returns start and stop operations without control sockets", func(t *testing.T) {
			host := ssh.SSHDestination("user@remote")

			start, _, stop := ssh.NewSSHTunnel(host, operation.DefaultRegistryPort, false)

			_, ok := start.(*ssh.SSHTunnelStart)
			assert.True(t, ok, "start operation is not of type SSHTunnelStart")
			_, ok = stop.(*ssh.SSHTunnelProcessStop)
			assert.True(t, ok, "stop operation is not of type SSHTunnelProcessStop")
		})

		t.Run("stop operation has access to start operation process", func(t *testing.T) {
			host := ssh.SSHDestination("user@remote")

			start, _, stop := ssh.NewSSHTunnel(host, operation.DefaultRegistryPort, false)
			startOp, ok := start.(*ssh.SSHTunnelStart)
			require.True(t, ok, "start operation is not of type SSHTunnelStart")

			stopOp, ok := stop.(*ssh.SSHTunnelProcessStop)
			require.True(t, ok, "stop operation is not of type SSHTunnelProcessStop")
			assert.Equal(t, startOp, stopOp.Start, "stop operation process does not match start operation process")
		})

		t.Run("it returns security check operation", func(t *testing.T) {
			host := ssh.SSHDestination("user@remote")

			_, securityCheck, _ := ssh.NewSSHTunnel(host, operation.DefaultRegistryPort, true)

			_, ok := securityCheck.(*ssh.CheckSSHTunnelSecurity)
			assert.True(t, ok, "security check operation is not of type CheckSSHTunnelSecurity")
		})
	})
}

func TestSSHTunnelStart(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it generates correct ssh command", func(t *testing.T) {
			host := ssh.SSHDestination("user@remote")
			port := operation.DefaultRegistryPort

			st := mustStartFromHost(t, host, port, true)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -fMS %s -R %s:127.0.0.1:%s user@remote", st.TargetHost.ControlSocketPath(), port, port)
			assert.Equal(t, want, got)
		})

		t.Run("it includes port flag when host has custom port", func(t *testing.T) {
			host := ssh.SSHDestination("user@remote:2222")
			port := operation.DefaultRegistryPort

			st := mustStartFromHost(t, host, port, true)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -p 2222 -fMS %s -R %s:127.0.0.1:%s user@remote", st.TargetHost.ControlSocketPath(), port, port)
			assert.Equal(t, want, got)
		})

		t.Run("it does not include control socket flag when disabled", func(t *testing.T) {
			host := ssh.SSHDestination("user@remote")
			port := operation.DefaultRegistryPort

			st := mustStartFromHost(t, host, port, false)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -R %s:127.0.0.1:%s user@remote", port, port)
			assert.Equal(t, want, got)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("it outputs the ssh command", func(t *testing.T) {
			var buf bytes.Buffer
			host := ssh.SSHDestination("user@remote")
			port := operation.DefaultRegistryPort

			st := mustStartFromHost(t, host, port, true)
			err := st.DryRun(&buf)
			got := strings.TrimSpace(buf.String())

			require.NoError(t, err)
			wantSuffix := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -fMS %s -R %s:127.0.0.1:%s user@remote", st.TargetHost.ControlSocketPath(), port, port)
			assert.True(t, strings.HasSuffix(got, wantSuffix),
				"DryRun output %q does not end with %q", got, wantSuffix)
		})

		t.Run("it includes port flag when host has custom port", func(t *testing.T) {
			var buf bytes.Buffer
			host := ssh.SSHDestination("user@remote:2222")
			port := operation.DefaultRegistryPort

			st := mustStartFromHost(t, host, port, true)
			err := st.DryRun(&buf)
			got := strings.TrimSpace(buf.String())

			require.NoError(t, err)
			wantSuffix := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -p 2222 -fMS %s -R %s:127.0.0.1:%s user@remote", st.TargetHost.ControlSocketPath(), port, port)
			assert.True(t, strings.HasSuffix(got, wantSuffix),
				"DryRun output %q does not end with %q", got, wantSuffix)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			st := mustStartFromHost(t, ssh.SSHDestination("user@remote"), operation.DefaultRegistryPort, true)

			got := st.Description()

			assert.Equal(t, "Open registry SSH tunnel", got)
		})
	})
}

func TestSSHTunnelStartEdgeCases(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it handles host without user", func(t *testing.T) {
			host := ssh.SSHDestination("remote-server")
			port := operation.DefaultRegistryPort

			st := mustStartFromHost(t, host, port, true)
			got := strings.Join(st.Command().Args, " ")

			socketPathPattern := `(?i).*[/\\]topo-tunnel-[a-f0-9]+`
			want := fmt.Sprintf(`^ssh -N -o ExitOnForwardFailure=yes -fMS %s -R %s:127\.0\.0\.1:%s (?:[^@ ]+@)?remote-server$`, socketPathPattern, port, port)
			assert.Regexp(t, want, got)
		})

		t.Run("it handles host without user but with port", func(t *testing.T) {
			host := ssh.SSHDestination("remote-server:2222")
			port := operation.DefaultRegistryPort

			st := mustStartFromHost(t, host, port, true)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -p 2222 -fMS %s -R %s:127.0.0.1:%s remote-server", st.TargetHost.ControlSocketPath(), port, port)
			assert.Equal(t, want, got)
		})

		t.Run("it handles IP address", func(t *testing.T) {
			host := ssh.SSHDestination("user@192.168.1.100")
			port := operation.DefaultRegistryPort

			st := mustStartFromHost(t, host, port, true)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -fMS %s -R %s:127.0.0.1:%s user@192.168.1.100", st.TargetHost.ControlSocketPath(), port, port)
			assert.Equal(t, want, got)
		})

		t.Run("it handles IP address with port", func(t *testing.T) {
			host := ssh.SSHDestination("user@192.168.1.100:2222")
			port := operation.DefaultRegistryPort

			st := mustStartFromHost(t, host, port, true)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -N -o ExitOnForwardFailure=yes -p 2222 -fMS %s -R %s:127.0.0.1:%s user@192.168.1.100", st.TargetHost.ControlSocketPath(), port, port)
			assert.Equal(t, want, got)
		})
	})
}

func mustStartFromHost(t *testing.T, host ssh.SSHDestination, port string, useControlSockets bool) *ssh.SSHTunnelStart {
	t.Helper()
	start, _, _ := ssh.NewSSHTunnel(host, port, useControlSockets)
	startOp, ok := start.(*ssh.SSHTunnelStart)
	require.True(t, ok, "start operation is not of type SSHTunnelStart")
	return startOp
}

func mustStopFromHost(t *testing.T, host ssh.SSHDestination, port string) *ssh.SSHTunnelStop {
	t.Helper()
	_, _, stop := ssh.NewSSHTunnel(host, port, true)
	stopOp, ok := stop.(*ssh.SSHTunnelStop)
	require.True(t, ok, "stop operation is not of type SSHTunnelStop")
	return stopOp
}

func mustCheckFromHost(t *testing.T, host ssh.SSHDestination, port string) *ssh.CheckSSHTunnelSecurity {
	t.Helper()
	_, check, _ := ssh.NewSSHTunnel(host, port, true)
	checkOp, ok := check.(*ssh.CheckSSHTunnelSecurity)
	require.True(t, ok, "security check operation is not of type CheckSSHTunnelSecurity")
	return checkOp
}

func TestCheckSSHTunnelSecurity(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it generates correct curl command", func(t *testing.T) {
			host := ssh.SSHDestination("user@remote")
			port := operation.DefaultRegistryPort

			cs := mustCheckFromHost(t, host, port)
			got := strings.Join(cs.Command().Args, " ")

			want := fmt.Sprintf("curl remote:%s --max-time 1", port)
			assert.Equal(t, want, got)
		})

		t.Run("it returns nil when target is localhost", func(t *testing.T) {
			host := ssh.SSHDestination("root@localhost")
			port := operation.DefaultRegistryPort

			cs := mustCheckFromHost(t, host, port)
			got := cs.Command()
			assert.Nil(t, got)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("it outputs the curl command", func(t *testing.T) {
			var buf bytes.Buffer
			host := ssh.SSHDestination("user@remote")
			port := operation.DefaultRegistryPort

			cs := mustCheckFromHost(t, host, port)
			err := cs.DryRun(&buf)
			got := strings.TrimSpace(buf.String())
			require.NoError(t, err)
			want := fmt.Sprintf("curl remote:%s --max-time 1", port)
			assert.Equal(t, want, got)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns the expected string", func(t *testing.T) {
			cs := mustCheckFromHost(t, ssh.SSHDestination("user@remote"), operation.DefaultRegistryPort)

			got := cs.Description()

			assert.Equal(t, "Check SSH tunnel security", got)
		})
	})
}

func TestSSHTunnelStop(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it generates correct ssh command", func(t *testing.T) {
			host := ssh.SSHDestination("user@remote")

			st := mustStopFromHost(t, host, operation.DefaultRegistryPort)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -S %s -O exit user@remote", st.TargetHost.ControlSocketPath())
			assert.Equal(t, want, got)
		})

		t.Run("it includes port flag when host has custom port", func(t *testing.T) {
			host := ssh.SSHDestination("user@remote:2222")

			st := mustStopFromHost(t, host, operation.DefaultRegistryPort)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -p 2222 -S %s -O exit user@remote", st.TargetHost.ControlSocketPath())
			assert.Equal(t, want, got)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("generates the correct ssh command", func(t *testing.T) {
			var buf bytes.Buffer
			host := ssh.SSHDestination("user@remote")

			st := mustStopFromHost(t, host, operation.DefaultRegistryPort)
			err := st.DryRun(&buf)
			got := strings.TrimSpace(buf.String())

			require.NoError(t, err)
			wantSuffix := fmt.Sprintf("ssh -S %s -O exit user@remote", st.TargetHost.ControlSocketPath())
			assert.True(t, strings.HasSuffix(got, wantSuffix),
				"DryRun output %q does not end with %q", got, wantSuffix)
		})

		t.Run("it includes port flag when host has custom port", func(t *testing.T) {
			var buf bytes.Buffer
			host := ssh.SSHDestination("user@remote:2222")

			st := mustStopFromHost(t, host, operation.DefaultRegistryPort)
			err := st.DryRun(&buf)
			got := strings.TrimSpace(buf.String())

			require.NoError(t, err)
			wantSuffix := fmt.Sprintf("ssh -p 2222 -S %s -O exit user@remote", st.TargetHost.ControlSocketPath())
			assert.True(t, strings.HasSuffix(got, wantSuffix),
				"DryRun output %q does not end with %q", got, wantSuffix)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			st := mustStopFromHost(t, ssh.SSHDestination("user@remote"), operation.DefaultRegistryPort)

			got := st.Description()

			assert.Equal(t, "Close registry SSH tunnel", got)
		})
	})
}

func TestSSHTunnelStopEdgeCases(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("handles host without user", func(t *testing.T) {
			host := ssh.SSHDestination("remote-server")

			st := mustStopFromHost(t, host, operation.DefaultRegistryPort)
			got := strings.Join(st.Command().Args, " ")

			socketPathPattern := `(?i).*[/\\]topo-tunnel-[a-f0-9]+`
			want := fmt.Sprintf(`^ssh -S %s -O exit (?:[^@ ]+@)?remote-server$`, socketPathPattern)
			assert.Regexp(t, want, got)
		})

		t.Run("handles host without user but with port", func(t *testing.T) {
			host := ssh.SSHDestination("remote-server:2222")

			st := mustStopFromHost(t, host, operation.DefaultRegistryPort)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -p 2222 -S %s -O exit remote-server", st.TargetHost.ControlSocketPath())
			assert.Equal(t, want, got)
		})

		t.Run("handles IP address", func(t *testing.T) {
			host := ssh.SSHDestination("user@192.168.1.100")

			st := mustStopFromHost(t, host, operation.DefaultRegistryPort)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -S %s -O exit user@192.168.1.100", st.TargetHost.ControlSocketPath())
			assert.Equal(t, want, got)
		})

		t.Run("handles IP address with port", func(t *testing.T) {
			host := ssh.SSHDestination("user@192.168.1.100:2222")

			st := mustStopFromHost(t, host, operation.DefaultRegistryPort)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -p 2222 -S %s -O exit user@192.168.1.100", st.TargetHost.ControlSocketPath())
			assert.Equal(t, want, got)
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

	t.Run("DryRun", func(t *testing.T) {
		t.Run("windows", func(t *testing.T) {
			testutil.RequireOS(t, "windows")

			t.Run("generates the correct kill command without target process", func(t *testing.T) {
				var buf bytes.Buffer

				st := ssh.NewSSHTunnelProcessStop(nil)
				err := st.DryRun(&buf)
				got := strings.TrimSpace(buf.String())

				require.NoError(t, err)
				wantSuffix := fmt.Sprintf("taskkill /PID %s /F", ssh.TunnelPIDPlaceholder)
				assert.True(t, strings.HasSuffix(got, wantSuffix),
					"DryRun output %q does not end with %q", got, wantSuffix)
			})

			t.Run("it includes port flag when host has custom port", func(t *testing.T) {
				var buf bytes.Buffer
				start := &ssh.SSHTunnelStart{Process: &os.Process{Pid: 12345}}

				st := ssh.NewSSHTunnelProcessStop(start)
				err := st.DryRun(&buf)
				got := strings.TrimSpace(buf.String())

				require.NoError(t, err)
				wantSuffix := fmt.Sprintf("taskkill /PID %d /F", start.Process.Pid)
				assert.True(t, strings.HasSuffix(got, wantSuffix),
					"DryRun output %q does not end with %q", got, wantSuffix)
			})
		})

		t.Run("linux", func(t *testing.T) {
			testutil.RequireOS(t, "linux")

			t.Run("generates the correct kill command without target process", func(t *testing.T) {
				var buf bytes.Buffer

				st := ssh.NewSSHTunnelProcessStop(nil)
				err := st.DryRun(&buf)
				got := strings.TrimSpace(buf.String())

				require.NoError(t, err)
				wantSuffix := fmt.Sprintf("kill -9 %s", ssh.TunnelPIDPlaceholder)
				assert.True(t, strings.HasSuffix(got, wantSuffix),
					"DryRun output %q does not end with %q", got, wantSuffix)
			})

			t.Run("it includes port flag when host has custom port", func(t *testing.T) {
				var buf bytes.Buffer
				start := &ssh.SSHTunnelStart{Process: &os.Process{Pid: 12345}}

				st := ssh.NewSSHTunnelProcessStop(start)
				err := st.DryRun(&buf)
				got := strings.TrimSpace(buf.String())

				require.NoError(t, err)
				wantSuffix := fmt.Sprintf("kill -9 %d", start.Process.Pid)
				assert.True(t, strings.HasSuffix(got, wantSuffix),
					"DryRun output %q does not end with %q", got, wantSuffix)
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
