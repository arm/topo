package health_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
)

func TestProbeHealthStatus(t *testing.T) {
	t.Run("probe fails connection", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "", fmt.Errorf("connection refused")
		}

		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})
		ts := health.ProbeHealthStatus(conn)

		assert.Error(t, ts.ConnectionError)
		assert.EqualError(t, ts.ConnectionError, "connection refused")
	})

	t.Run("probe finds remote CPUs", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			switch {
			case command == "true":
				return "", nil
			case strings.Contains(command, "ls /sys/class/remoteproc"):
				return "remoteproc0\nremoteproc1", nil
			case strings.Contains(command, "cat /sys/class/remoteproc"):
				return "foo\nbar", nil
			default:
				return "", errors.New("unexpected command: " + command)
			}
		}

		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})
		ts := health.ProbeHealthStatus(conn)

		want := health.HardwareProfile{RemoteCPU: []target.RemoteprocCPU{{Name: "foo"}, {Name: "bar"}}}
		assert.NoError(t, ts.ConnectionError)
		assert.Equal(t, want, ts.Hardware)
	})

	t.Run("probe succeeds when no remoteproc support", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			switch command {
			case "true":
				return "", nil
			default:
				return "", errors.New("no such directory")
			}
		}

		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})
		ts := health.ProbeHealthStatus(conn)

		assert.NoError(t, ts.ConnectionError)
		assert.Len(t, ts.Hardware.RemoteCPU, 0)
	})
}
