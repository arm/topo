package probe_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeMemory(t *testing.T) {
	t.Run("parses MemTotal from /proc/meminfo", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"cat /proc/meminfo": {Output: "MemTotal:       16384000 kB\nMemFree:        8192000 kB"},
			},
		}

		got, err := probe.Memory(context.Background(), r)

		require.NoError(t, err)
		assert.Equal(t, int64(16384000), got)
	})

	t.Run("returns error when MemTotal not found", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"cat /proc/meminfo": {Output: "MemFree:        8192000 kB"},
			},
		}

		_, err := probe.Memory(context.Background(), r)

		assert.Error(t, err)
	})

	t.Run("returns error when value is invalid", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"cat /proc/meminfo": {Output: "MemTotal:       notanumber"},
			},
		}

		_, err := probe.Memory(context.Background(), r)

		assert.Error(t, err)
	})

	t.Run("returns error when cat fails", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"cat /proc/meminfo": {Err: assert.AnError},
			},
		}

		_, err := probe.Memory(context.Background(), r)

		assert.Error(t, err)
	})
}
