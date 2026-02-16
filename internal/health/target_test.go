package health_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	t.Run("probe succeeds and collects CPU info", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			switch {
			case command == "true":
				return "", nil
			case strings.Contains(command, "command -v"):
				return "/usr/bin/lscpu", nil
			case command == "lscpu --json":
				return testutil.LsCpuOutputRaw, nil
			default:
				return "", errors.New("unexpected command: " + command)
			}
		}

		conn := health.NewConnection("hostname", mockExec)
		ts := conn.Probe()

		assert.NoError(t, ts.ConnectionError)
		require.Len(t, ts.Hardware.HostProcessor, 1)
		assert.Equal(t, "Cortex-A55", ts.Hardware.HostProcessor[0].ModelName)
		assert.Equal(t, 2, ts.Hardware.HostProcessor[0].Cores)
		assert.Equal(t, []string{"fp", "asimd"}, ts.Hardware.HostProcessor[0].Features)
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

	t.Run("probe finds remote CPUs", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			switch {
			case command == "true":
				return "", nil
			case strings.Contains(command, "command -v"):
				return "/usr/bin/lscpu", nil
			case command == "lscpu --json":
				return testutil.LsCpuOutputRaw, nil
			case strings.Contains(command, "ls /sys/class/remoteproc"):
				return "remoteproc0\nremoteproc1", nil
			case strings.Contains(command, "cat /sys/class/remoteproc"):
				return "foo\nbar", nil
			default:
				return "", errors.New("unexpected command: " + command)
			}
		}

		conn := health.NewConnection("hostname", mockExec)
		ts := conn.Probe()

		want := []health.RemoteprocCPU{{Name: "foo"}, {Name: "bar"}}
		assert.Equal(t, want, ts.Hardware.RemoteCPU)
	})

	t.Run("probe succeeds when no remoteproc support", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			switch {
			case command == "true":
				return "", nil
			case strings.Contains(command, "command -v"):
				return "/usr/bin/lscpu", nil
			case command == "lscpu --json":
				return testutil.LsCpuOutputRaw, nil
			default:
				return "", errors.New("no such directory")
			}
		}

		conn := health.NewConnection("hostname", mockExec)
		ts := conn.Probe()

		assert.NoError(t, ts.ConnectionError)
		assert.Nil(t, ts.Hardware.RemoteCPU)
		require.Len(t, ts.Hardware.HostProcessor, 1)
		assert.Equal(t, "Cortex-A55", ts.Hardware.HostProcessor[0].ModelName)
		assert.Equal(t, 2, ts.Hardware.HostProcessor[0].Cores)
	})
}

func TestProbeHardware(t *testing.T) {
	t.Run("returns model name and features", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			switch {
			case strings.Contains(command, "command -v"):
				return "/usr/bin/lscpu", nil
			case command == "lscpu --json":
				return testutil.LsCpuOutputRaw, nil
			default:
				return "", errors.New("not found")
			}
		}

		conn := health.NewConnection("hostname", mockExec)
		hw, err := conn.ProbeHardware()

		require.NoError(t, err)
		require.Len(t, hw.HostProcessor, 1)
		assert.Equal(t, "Cortex-A55", hw.HostProcessor[0].ModelName)
		assert.Equal(t, 2, hw.HostProcessor[0].Cores)
		assert.Equal(t, []string{"fp", "asimd"}, hw.HostProcessor[0].Features)
	})

	t.Run("returns error when lscpu not found", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			return "", errors.New("not found")
		}

		conn := health.NewConnection("hostname", mockExec)
		_, err := conn.ProbeHardware()

		assert.ErrorContains(t, err, "lscpu not found")
	})

	t.Run("returns error when lscpu output is invalid JSON", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string) (string, error) {
			switch {
			case strings.Contains(command, "command -v"):
				return "/usr/bin/lscpu", nil
			case command == "lscpu --json":
				return "not json", nil
			default:
				return "", errors.New("not found")
			}
		}

		conn := health.NewConnection("hostname", mockExec)
		_, err := conn.ProbeHardware()

		assert.ErrorContains(t, err, "collecting CPU info")
	})
}

func TestCreateCPUProfile(t *testing.T) {
	t.Run("parses lscpu with sockets", func(t *testing.T) {
		input := []health.LscpuOutputField{
			{Field: "Vendor ID:", Data: "ARM"},
			{Field: "Model name:", Data: "Cortex-A72"},
			{Field: "Core(s) per socket:", Data: "4"},
			{Field: "Socket(s):", Data: "2"},
			{Field: "Flags:", Data: "fp asimd evtstrm"},
		}

		got, err := health.CreateCPUProfile(input)

		require.NoError(t, err)
		require.Len(t, got, 1)
		want := health.HostProcessor{
			ModelName: "Cortex-A72",
			Cores:     8,
			Features:  []string{"fp", "asimd", "evtstrm"},
		}
		assert.Equal(t, want, got[0])
	})

	t.Run("parses lscpu with clusters", func(t *testing.T) {
		input := []health.LscpuOutputField{
			{Field: "Vendor ID:", Data: "ARM"},
			{Field: "Model name:", Data: "Cortex-A55"},
			{Field: "Core(s) per cluster:", Data: "2"},
			{Field: "Socket(s):", Data: "-"},
			{Field: "Cluster(s):", Data: "1"},
			{Field: "Flags:", Data: "fp asimd"},
		}

		got, err := health.CreateCPUProfile(input)

		require.NoError(t, err)
		require.Len(t, got, 1)
		want := health.HostProcessor{
			ModelName: "Cortex-A55",
			Cores:     2,
			Features:  []string{"fp", "asimd"},
		}
		assert.Equal(t, want, got[0])
	})

	t.Run("parses multiple processors", func(t *testing.T) {
		input := []health.LscpuOutputField{
			{Field: "Vendor ID:", Data: "ARM"},
			{Field: "Model name:", Data: "Cortex-A55"},
			{Field: "Core(s) per socket:", Data: "4"},
			{Field: "Socket(s):", Data: "1"},
			{Field: "Flags:", Data: "fp asimd"},
			{Field: "Model name:", Data: "Cortex-A78"},
			{Field: "Core(s) per socket:", Data: "2"},
			{Field: "Socket(s):", Data: "1"},
			{Field: "Flags:", Data: "fp asimd sve"},
		}

		got, err := health.CreateCPUProfile(input)

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "Cortex-A55", got[0].ModelName)
		assert.Equal(t, 4, got[0].Cores)
		assert.Equal(t, "Cortex-A78", got[1].ModelName)
		assert.Equal(t, 2, got[1].Cores)
	})

	t.Run("returns empty when no model name field", func(t *testing.T) {
		input := []health.LscpuOutputField{
			{Field: "Architecture:", Data: "aarch64"},
		}

		got, err := health.CreateCPUProfile(input)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns error when cores per socket is not a number", func(t *testing.T) {
		input := []health.LscpuOutputField{
			{Field: "Model name:", Data: "Cortex-A55"},
			{Field: "Core(s) per socket:", Data: "abc"},
			{Field: "Socket(s):", Data: "1"},
		}

		_, err := health.CreateCPUProfile(input)

		assert.Error(t, err)
	})

	t.Run("returns error when Socket(s)/Cluster(s) is not found", func(t *testing.T) {
		input := []health.LscpuOutputField{
			{Field: "Model name:", Data: "Cortex-A55"},
			{Field: "Core(s) per socket:", Data: "1"},
		}

		_, err := health.CreateCPUProfile(input)

		assert.ErrorContains(t, err, "could not determine CPU units")
	})
}

func TestBinaryExists(t *testing.T) {
	t.Run("when binary found returns true", func(t *testing.T) {
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
