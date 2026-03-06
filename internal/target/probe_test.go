package target_test

import (
	"os/exec"
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
		mockExec := func(_ ssh.Host, command string, _ []byte, _ ...string) *exec.Cmd {
			switch {
			case strings.Contains(command, "command -v"):
				return testutil.CmdWithOutput("/usr/bin/lscpu", 0)
			case command == "lscpu --json":
				return testutil.CmdWithOutput(testutil.LsCpuOutputRaw, 0)
			case strings.Contains(command, "meminfo"):
				return testutil.CmdWithOutput("MemTotal:       16384000 kB", 0)
			default:
				return testutil.CmdWithOutput("not found", 1)
			}
		}

		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockExec: mockExec})
		hw, err := conn.ProbeHardware()

		require.NoError(t, err)
		require.Len(t, hw.HostProcessor, 1)
		assert.Equal(t, "Cortex-A55", hw.HostProcessor[0].Model)
		assert.Equal(t, 2, hw.HostProcessor[0].Cores)
		assert.Equal(t, []string{"fp", "asimd"}, hw.HostProcessor[0].Features)
		assert.Equal(t, int64(16384000), hw.TotalMemoryKb)
	})

	t.Run("returns error when lscpu not found", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string, _ []byte, _ ...string) *exec.Cmd {
			return testutil.CmdWithOutput("not found", 1)
		}

		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockExec: mockExec})
		_, err := conn.ProbeHardware()

		assert.ErrorContains(t, err, `"lscpu" executable file not found in $PATH`)
	})

	t.Run("returns error when lscpu output is invalid JSON", func(t *testing.T) {
		mockExec := func(_ ssh.Host, command string, _ []byte, _ ...string) *exec.Cmd {
			switch {
			case strings.Contains(command, "command -v"):
				return testutil.CmdWithOutput("/usr/bin/lscpu", 0)
			case command == "lscpu --json":
				return testutil.CmdWithOutput("not json", 0)
			case strings.Contains(command, "meminfo"):
				return testutil.CmdWithOutput("MemTotal:       16384000 kB", 0)
			default:
				return testutil.CmdWithOutput("not found", 1)
			}
		}

		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockExec: mockExec})
		_, err := conn.ProbeHardware()

		assert.ErrorContains(t, err, "collecting CPU info")
	})
}

func TestFindKeyValueInString(t *testing.T) {
	t.Run("finds key and parses value", func(t *testing.T) {
		text := `MemTotal:       16384000 kB
MemFree:        8192000 kB`

		got, err := target.FindKeyValueInString("MemTotal", text)

		require.NoError(t, err)
		assert.Equal(t, int64(16384000), got)
	})

	t.Run("returns error when key not found", func(t *testing.T) {
		text := `MemTotal:       16384000 kB`

		got, err := target.FindKeyValueInString("MissingKey", text)

		assert.Error(t, err)
		assert.Equal(t, int64(0), got)
	})

	t.Run("returns error when value is invalid", func(t *testing.T) {
		text := `MemTotal:       notanumber`

		_, err := target.FindKeyValueInString("MemTotal", text)

		assert.Error(t, err)
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
			Model:    "Cortex-A72",
			Cores:    8,
			Features: []string{"fp", "asimd", "evtstrm"},
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
			Model:    "Cortex-A55",
			Cores:    2,
			Features: []string{"fp", "asimd"},
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
		assert.Equal(t, "Cortex-A55", got[0].Model)
		assert.Equal(t, 4, got[0].Cores)
		assert.Equal(t, "Cortex-A78", got[1].Model)
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
