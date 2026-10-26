package fact_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/fact"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryProvider(t *testing.T) {
	t.Run("Facts reports total memory in KiB", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			"cat /proc/meminfo": {Output: "MemFree: 1024 kB\nMemTotal:       16384000 kB\n"},
		}}
		want := fact.Report{Groups: []fact.Group{{
			ID: fact.GroupSystemMemory,
			Facts: []fact.Fact{{
				Key:         fact.FactMemoryTotal,
				Value:       fact.Value{Type: "integer", Integer: 16384000},
				Label:       "Total memory",
				Description: "Kernel-reported MemTotal in KiB",
			}},
		}}}

		got, err := (fact.MemoryProvider{}).Facts(context.Background(), r)

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("returns error when MemTotal not found", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"cat /proc/meminfo": {Output: "MemFree:        8192000 kB"},
			},
		}

		_, err := (fact.MemoryProvider{}).Facts(context.Background(), r)

		assert.Error(t, err)
	})

	t.Run("returns error when value is invalid", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"cat /proc/meminfo": {Output: "MemTotal:       notanumber"},
			},
		}

		_, err := (fact.MemoryProvider{}).Facts(context.Background(), r)

		assert.Error(t, err)
	})

	t.Run("returns error when cat fails", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"cat /proc/meminfo": {Err: assert.AnError},
			},
		}

		_, err := (fact.MemoryProvider{}).Facts(context.Background(), r)

		assert.Error(t, err)
	})
}
