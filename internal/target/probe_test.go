package target_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractArmFeatures(t *testing.T) {
	t.Run("extracts mapped Arm features and ignores unrecognised", func(t *testing.T) {
		ts := target.HostProcessor{
			Features: []string{"fp", "asimd", "sve2", "sme"},
		}

		res := ts.ExtractArmFeatures()

		want := []string{"NEON", "SVE2", "SME"}
		assert.Equal(t, want, res)
	})

	t.Run("returns empty slice if no matching features", func(t *testing.T) {
		ts := target.HostProcessor{
			Features: []string{"fp", "crc32"},
		}

		res := ts.ExtractArmFeatures()

		assert.Empty(t, res)
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

		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})
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

		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})
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

		conn := target.NewConnection("hostname", mockExec, target.ConnectionOptions{})
		_, err := conn.ProbeHardware()

		assert.ErrorContains(t, err, "collecting CPU info")
	})
}

func TestCreateCPUProfile(t *testing.T) {
	t.Run("parses lscpu with sockets", func(t *testing.T) {
		input := []target.LscpuOutputField{
			{Field: "Vendor ID:", Data: "ARM"},
			{Field: "Model name:", Data: "Cortex-A72"},
			{Field: "Core(s) per socket:", Data: "4"},
			{Field: "Socket(s):", Data: "2"},
			{Field: "Flags:", Data: "fp asimd evtstrm"},
		}

		got, err := target.CreateCPUProfile(input)

		require.NoError(t, err)
		require.Len(t, got, 1)
		want := target.HostProcessor{
			ModelName: "Cortex-A72",
			Cores:     8,
			Features:  []string{"fp", "asimd", "evtstrm"},
		}
		assert.Equal(t, want, got[0])
	})

	t.Run("parses lscpu with clusters", func(t *testing.T) {
		input := []target.LscpuOutputField{
			{Field: "Vendor ID:", Data: "ARM"},
			{Field: "Model name:", Data: "Cortex-A55"},
			{Field: "Core(s) per cluster:", Data: "2"},
			{Field: "Socket(s):", Data: "-"},
			{Field: "Cluster(s):", Data: "1"},
			{Field: "Flags:", Data: "fp asimd"},
		}

		got, err := target.CreateCPUProfile(input)

		require.NoError(t, err)
		require.Len(t, got, 1)
		want := target.HostProcessor{
			ModelName: "Cortex-A55",
			Cores:     2,
			Features:  []string{"fp", "asimd"},
		}
		assert.Equal(t, want, got[0])
	})

	t.Run("parses multiple processors", func(t *testing.T) {
		input := []target.LscpuOutputField{
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

		got, err := target.CreateCPUProfile(input)

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "Cortex-A55", got[0].ModelName)
		assert.Equal(t, 4, got[0].Cores)
		assert.Equal(t, "Cortex-A78", got[1].ModelName)
		assert.Equal(t, 2, got[1].Cores)
	})

	t.Run("returns empty when no model name field", func(t *testing.T) {
		input := []target.LscpuOutputField{
			{Field: "Architecture:", Data: "aarch64"},
		}

		got, err := target.CreateCPUProfile(input)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns error when cores per socket is not a number", func(t *testing.T) {
		input := []target.LscpuOutputField{
			{Field: "Model name:", Data: "Cortex-A55"},
			{Field: "Core(s) per socket:", Data: "abc"},
			{Field: "Socket(s):", Data: "1"},
		}

		_, err := target.CreateCPUProfile(input)

		assert.Error(t, err)
	})
}
