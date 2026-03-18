package health_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestProbeHealthStatus(t *testing.T) {
	t.Run("probe fails connection", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string, _ []byte, _ ...string) *exec.Cmd {
			return testutil.CmdWithOutput("connection refused", 1)
		}

		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockCommand: mockExec})
		ts := health.ProbeHealthStatus(conn)

		assert.Error(t, ts.ConnectionError)
		assert.ErrorContains(t, ts.ConnectionError, "exit status")
	})

	t.Run("probe finds remote CPUs", func(t *testing.T) {
		mockExec := func(_ ssh.Host, cmdStr string, _ []byte, _ ...string) *exec.Cmd {
			switch {
			case cmdStr == "true":
				return testutil.CmdWithOutput("", 0)
			case strings.Contains(cmdStr, "ls /sys/class/remoteproc"):
				return testutil.CmdWithOutput("remoteproc0\nremoteproc1", 0)
			case strings.Contains(cmdStr, "cat /sys/class/remoteproc"):
				return testutil.CmdWithOutput("foo\nbar", 0)
			default:
				return testutil.CmdWithOutput("unexpected command: "+cmdStr, 1)
			}
		}

		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockCommand: mockExec})
		ts := health.ProbeHealthStatus(conn)

		want := health.HardwareProfile{RemoteCPU: []target.RemoteprocCPU{{Name: "foo"}, {Name: "bar"}}}
		assert.NoError(t, ts.ConnectionError)
		assert.Equal(t, want, ts.Hardware)
	})

	t.Run("probe succeeds when no remoteproc support", func(t *testing.T) {
		mockExec := func(_ ssh.Host, cmdStr string, _ []byte, _ ...string) *exec.Cmd {
			switch cmdStr {
			case "true":
				return testutil.CmdWithOutput("", 0)
			default:
				return testutil.CmdWithOutput("no such directory", 1)
			}
		}

		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockCommand: mockExec})
		ts := health.ProbeHealthStatus(conn)

		assert.NoError(t, ts.ConnectionError)
		assert.Len(t, ts.Hardware.RemoteCPU, 0)
	})
}
