package operation_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/require"
)

func osExit(code int) {}

func TestSetupExitCleanup(t *testing.T) {
	t.Run("calls cleanup operation on interrupt signal", func(t *testing.T) {
		testutil.RequireOS(t, "linux")
		cleanupOp := new(mockOperation)
		var buf bytes.Buffer
		cleanupOp.On("Run", &buf).Return(nil)
		p, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)
		operation.SetupExitCleanup(cleanupOp, &buf, osExit)

		err = p.Signal(os.Interrupt)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)

		cleanupOp.AssertExpectations(t)
	})

	t.Run("calls cleanup operation only once when called multiple times", func(t *testing.T) {
		cleanupOp := new(mockOperation)
		var buf bytes.Buffer
		cleanupOp.On("Run", &buf).Return(nil).Once()
		cleanup := operation.SetupExitCleanup(cleanupOp, &buf, osExit)

		cleanup()
		cleanup()
		cleanup()

		cleanupOp.AssertExpectations(t)
	})

	t.Run("still handles signal when operation is nil", func(t *testing.T) {
		testutil.RequireOS(t, "linux")
		var buf bytes.Buffer
		operation.SetupExitCleanup(nil, &buf, osExit)
		p, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)

		require.NotPanics(t, func() {
			err = p.Signal(os.Interrupt)
			require.NoError(t, err)
			time.Sleep(100 * time.Millisecond)
		})
	})
}
