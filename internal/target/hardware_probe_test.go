package target_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeHardware(t *testing.T) {
	t.Run("returns model name and features", func(t *testing.T) {
		r := &runner.Fake{
			Binaries: []string{"lscpu"},
			Commands: map[string]runner.FakeResult{
				command.WrapInLoginShell("lscpu --json"):             {Output: testutil.LsCpuOutputRaw},
				command.WrapInLoginShell("ls /sys/class/remoteproc"): {Output: ""},
				command.WrapInLoginShell("cat /proc/meminfo"):        {Output: "MemTotal:       16384000 kB"},
			},
		}

		got, err := target.ProbeHardware(context.Background(), r)

		require.NoError(t, err)
		want := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{
					Model:    "Cortex-A55",
					Cores:    2,
					Features: []string{"fp", "asimd"},
				},
			},
			TotalMemoryKb: int64(16384000),
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns error when lscpu not found", func(t *testing.T) {
		r := &runner.Fake{}

		_, err := target.ProbeHardware(context.Background(), r)

		assert.ErrorContains(t, err, `"lscpu" not found in $PATH`)
	})

	t.Run("returns error when lscpu output is invalid JSON", func(t *testing.T) {
		r := &runner.Fake{
			Binaries: []string{"lscpu"},
			Commands: map[string]runner.FakeResult{
				command.WrapInLoginShell("lscpu --json"): {Output: "not json"},
			},
		}

		_, err := target.ProbeHardware(context.Background(), r)

		assert.ErrorContains(t, err, "collecting CPU info")
	})
}
