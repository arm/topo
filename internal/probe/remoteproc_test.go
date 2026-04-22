package probe_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeRemoteproc(t *testing.T) {
	t.Run("returns remote processors", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"ls /sys/class/remoteproc 2>/dev/null || true": {Output: "remoteproc0\nremoteproc1"},
				"cat /sys/class/remoteproc/*/name":             {Output: "virtio0\nvirtio1"},
			},
		}

		got, err := probe.Remoteproc(context.Background(), r)

		require.NoError(t, err)
		want := []probe.RemoteprocCPU{
			{Name: "virtio0"},
			{Name: "virtio1"},
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns error when listing remoteproc directory fails", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"ls /sys/class/remoteproc 2>/dev/null || true": {Output: "", Err: errors.New("connection lost")},
			},
		}

		_, err := probe.Remoteproc(context.Background(), r)

		assert.ErrorContains(t, err, "connection lost")
	})

	t.Run("returns empty when remoteproc directory is empty", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"ls /sys/class/remoteproc 2>/dev/null || true": {Output: ""},
			},
		}

		got, err := probe.Remoteproc(context.Background(), r)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns error when reading names fails", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"ls /sys/class/remoteproc 2>/dev/null || true": {Output: "remoteproc0"},
				"cat /sys/class/remoteproc/*/name":             {Err: errors.New("permission denied")},
			},
		}

		_, err := probe.Remoteproc(context.Background(), r)

		assert.ErrorContains(t, err, "permission denied")
	})
}
