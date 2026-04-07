package runner_test

import (
	"context"
	"testing"
	"time"

	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/require"
)

func TestLocal(t *testing.T) {
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
}
