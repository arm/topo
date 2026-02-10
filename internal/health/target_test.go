package health_test

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/stretchr/testify/assert"
)

//go:embed data/lscpu_output.json
var lsCpuOutputRaw []byte

func TestRun(t *testing.T) {
	t.Run("run executes command successfully", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "success", nil
		}
		conn := health.NewConnection("hostname", mockExec)

		out, err := conn.Run("ls")

		assert.NoError(t, err)
		assert.Equal(t, "success", out)
	})

	t.Run("run returns error", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "", errors.New("ssh failed")
		}
		conn := health.NewConnection("hostname", mockExec)

		out, err := conn.Run("ls")

		assert.Error(t, err)
		assert.Empty(t, out)
	})
}

func TestProbe(t *testing.T) {
	var lsCpuOutput health.LscpuOutput
	err := json.Unmarshal(lsCpuOutputRaw, &lsCpuOutput)
	assert.NoError(t, err, "failed to unmarshal lscpu output")

	t.Run("probe succeeds and collects features", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			if command == "" {
				return "", nil // simulate successful initial connection
			}
			return string(lsCpuOutputRaw), nil
		}

		conn := health.NewConnection("hostname", mockExec)
		ts := conn.Probe()

		assert.NoError(t, ts.ConnectionError)
		assert.Equal(t, []string{"fpu", "asimd"}, ts.Hardware.HostCPU.Features)
	})

	t.Run("probe succeeds but features collection returns empty", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			if command == "" {
				return "", nil
			}
			return "", nil
		}

		conn := health.NewConnection("hostname", mockExec)
		ts := conn.Probe()

		assert.NoError(t, ts.ConnectionError)
		assert.Empty(t, ts.Hardware.HostCPU.Features)
	})

	t.Run("probe fails connection", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "", fmt.Errorf("connection refused")
		}

		conn := health.NewConnection("hostname", mockExec)
		ts := conn.Probe()

		assert.Error(t, ts.ConnectionError)
		assert.EqualError(t, ts.ConnectionError, "connection refused")
	})

	t.Run("probe finds remote cpu", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			if strings.Contains(command, "remoteproc") {
				return "foo\nbar", nil
			}
			return "", nil
		}

		conn := health.NewConnection("hostname", mockExec)
		ts := conn.Probe()

		want := []health.RemoteProcCPU{{Name: "foo"}, {Name: "bar"}}
		assert.Equal(t, want, ts.Hardware.RemoteCPU)
	})
}

func TestProbeHardware(t *testing.T) {
	t.Run("probe includes core count and model name", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			if command == "" {
				return "", nil // simulate successful initial connection
			}
			return string(lsCpuOutputRaw), nil
		}

		conn := health.NewConnection("hostname", mockExec)
		ts, err := conn.ProbeHardware()

		assert.NoError(t, err)
		assert.Equal(t, 2, ts.HostCPU.Cores)
		assert.Equal(t, "Cortex-A55", ts.HostCPU.ModelName)
	})
}

func TestBinaryExists(t *testing.T) {
	t.Run("when binary found, returns true", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "/foo/bar", nil
		}
		conn := health.NewConnection("hostname", mockExec)

		got, err := conn.BinaryExists("bar")

		assert.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("invalid format returns an error", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string) (string, error) {
			return "/foo/bar", nil
		}
		conn := health.NewConnection("hostname", mockExec)

		got, err := conn.BinaryExists("b a r")

		assert.Error(t, err)
		assert.False(t, got)
	})
}
