package command_test

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancellationStopsChildren(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	// go run acts as the wrapper; the program is its child.
	cmd := exec.CommandContext(ctx, "go", "run", "./testdata/cancellation")
	command.ConfigureCancellation(cmd)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	t.Cleanup(func() { cancel(); _ = cmd.Wait() })
	require.NoError(t, cmd.Start())
	output := bufio.NewScanner(stdout)
	require.True(t, output.Scan(), "child did not start")
	require.Equal(t, "ready", output.Text())

	cancel()
	// Read before Wait, which closes stdout even if a child is still running.
	survived := output.Scan()
	_ = cmd.Wait()

	assert.NoError(t, output.Err())
	assert.False(t, survived, "child survived cancellation")
}
