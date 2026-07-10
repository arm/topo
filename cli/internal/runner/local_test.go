package runner_test

import (
	"context"
	"testing"
	"time"

	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocal(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		t.Run("cancelled context returns timeout error", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			r := runner.NewLocal()

			_, err := r.Run(ctx, "sleep 10")

			require.ErrorIs(t, err, runner.ErrTimeout)
		})

		t.Run("expired deadline returns timeout error", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			defer cancel()
			time.Sleep(5 * time.Millisecond)
			r := runner.NewLocal()

			_, err := r.Run(ctx, "sleep 10")

			require.ErrorIs(t, err, runner.ErrTimeout)
		})

		t.Run("executes binary directly without shell interpretation", func(t *testing.T) {
			r := runner.NewLocal()
			got, err := r.Run(context.Background(), "echo hello && echo world")

			assert.NoError(t, err)
			assert.Equal(t, "hello && echo world\n", got)
		})
	})

	t.Run("BinaryExists", func(t *testing.T) {
		t.Run("returns nil for a binary that exists", func(t *testing.T) {
			r := runner.NewLocal()

			err := r.BinaryExists(context.Background(), "ls")

			assert.NoError(t, err)
		})

		t.Run("returns error mentioning binary name when not found", func(t *testing.T) {
			r := runner.NewLocal()

			err := r.BinaryExists(context.Background(), "definitely-not-a-real-binary")

			assert.EqualError(t, err, `"definitely-not-a-real-binary" not found in $PATH`)
		})
	})
}
