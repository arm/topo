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
				"test -d /sys/class/remoteproc":    {},
				"cat /sys/class/remoteproc/*/name": {Output: "virtio0\nvirtio1"},
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

	t.Run("returns empty when remoteproc directory does not exist", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"test -d /sys/class/remoteproc": {Output: "", Err: errors.New("exit status 1")},
			},
		}

		got, err := probe.Remoteproc(context.Background(), r)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns error when listing remoteproc directory fails", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"test -d /sys/class/remoteproc": {Output: "", Err: runner.ErrTimeout},
			},
		}

		_, err := probe.Remoteproc(context.Background(), r)

		assert.ErrorIs(t, err, runner.ErrTimeout)
	})

	t.Run("returns empty when no remoteproc names found", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"test -d /sys/class/remoteproc":    {},
				"cat /sys/class/remoteproc/*/name": {Output: ""},
			},
		}

		got, err := probe.Remoteproc(context.Background(), r)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns empty when reading names fails with non-timeout error", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"test -d /sys/class/remoteproc":    {},
				"cat /sys/class/remoteproc/*/name": {Err: errors.New("No such file or directory")},
			},
		}

		got, err := probe.Remoteproc(context.Background(), r)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns error when reading names times out", func(t *testing.T) {
		r := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"test -d /sys/class/remoteproc":    {},
				"cat /sys/class/remoteproc/*/name": {Err: runner.ErrTimeout},
			},
		}

		_, err := probe.Remoteproc(context.Background(), r)

		assert.ErrorIs(t, err, runner.ErrTimeout)
	})
}
