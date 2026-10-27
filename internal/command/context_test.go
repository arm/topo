package command_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"testing"
	"time"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancellationStopsChildren(t *testing.T) {
	binary, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cmd := exec.CommandContext(ctx, binary, "-test.run=^TestCancellationProcess$")
	command.ConfigureCancellation(cmd)
	cmd.Env = append(os.Environ(), "TOPO_CANCEL_TEST_PROCESS=parent")
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

// TestCancellationProcess acts as a command wrapper and its child process.
func TestCancellationProcess(t *testing.T) {
	switch os.Getenv("TOPO_CANCEL_TEST_PROCESS") {
	case "parent":
		cmd := exec.Command(os.Args[0], "-test.run=^TestCancellationProcess$")
		cmd.Env = append(os.Environ(), "TOPO_CANCEL_TEST_PROCESS=child")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	case "child":
		// Keep the pipe open after the wrapper exits; forced cleanup must stop us.
		signal.Ignore(os.Interrupt)
		fmt.Println("ready")
		time.Sleep(10 * time.Second)
		fmt.Println("survived")
	}
}
