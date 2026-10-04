package deploy_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareRegistry(t *testing.T) {
	config := deploy.RegistryConfig{ContainerName: "test-registry", Port: "12345"}

	t.Run("creates a missing registry with the requested configuration", func(t *testing.T) {
		executor := &registryExecutorFake{}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.NoError(t, err)
		assert.Equal(t, [][]string{
			{"inspect", "test-registry"},
			{
				"run", "--pull=missing", "-d", "--restart", "always",
				"-p", "127.0.0.1:12345:5000", "--name", "test-registry", "registry:2",
			},
		}, executor.runCalls)
		assert.Empty(t, executor.commandCalls)
	})

	t.Run("starts a stopped registry using its configured port", func(t *testing.T) {
		executor := &registryExecutorFake{
			exists: true,
			inspectOutput: `[{"State":{"Running":false},
				"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":"12345"}]}},
				"NetworkSettings":{"Ports":{}}}]`,
		}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.NoError(t, err)
		assert.Equal(t, [][]string{{"inspect", "test-registry"}, {"start", "test-registry"}}, executor.runCalls)
		assert.Equal(t, [][]string{{"inspect", "test-registry"}}, executor.commandCalls)
	})

	t.Run("reuses a running registry using its engine-assigned port", func(t *testing.T) {
		executor := &registryExecutorFake{
			exists: true,
			inspectOutput: `[{"State":{"Running":true},
				"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":"0"}]}},
				"NetworkSettings":{"Ports":{"5000/tcp":[{"HostPort":"12345"}]}}}]`,
		}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.NoError(t, err)
		assert.Equal(t, [][]string{{"inspect", "test-registry"}, {"start", "test-registry"}}, executor.runCalls)
	})

	t.Run("rejects a mismatched port without starting or creating a registry", func(t *testing.T) {
		executor := &registryExecutorFake{
			exists: true,
			inspectOutput: `[{"State":{"Running":true},
				"NetworkSettings":{"Ports":{"5000/tcp":[{"HostPort":"54321"}]}}}]`,
		}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.EqualError(t, err,
			"registry port mismatch (running: 54321, requested: 12345)\n"+
				"you may need to remove the existing registry container test-registry")
		assert.Equal(t, [][]string{{"inspect", "test-registry"}}, executor.runCalls)
	})

	t.Run("reports an inspection command failure without starting the registry", func(t *testing.T) {
		executor := &registryExecutorFake{exists: true, inspectExitCode: 1}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.ErrorContains(t, err, "failed to execute")
		var exitError *exec.ExitError
		assert.ErrorAs(t, err, &exitError)
		assert.Equal(t, [][]string{{"inspect", "test-registry"}}, executor.runCalls)
	})

	t.Run("rejects invalid inspection data without starting the registry", func(t *testing.T) {
		tests := []struct {
			name          string
			inspectOutput string
			wantError     string
		}{
			{"malformed JSON", "not JSON", "failed to inspect existing registry test-registry: decode registry inspect output"},
			{"no containers", "[]", "expected one inspected container, got 0"},
			{"multiple containers", "[{},{}]", "expected one inspected container, got 2"},
			{"missing binding", "[{}]", "container port 5000 is not published"},
			{
				"empty bindings",
				`[{"HostConfig":{"PortBindings":{"5000/tcp":[]}}}]`,
				"container port 5000 is not published",
			},
			{
				"empty host port",
				`[{"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":""}]}}}]`,
				"container port 5000 is not published",
			},
			{
				"running registry without an assigned port",
				`[{"State":{"Running":true},
					"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":"12345"}]}}}]`,
				"container port 5000 is not published",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				executor := &registryExecutorFake{exists: true, inspectOutput: test.inspectOutput}
				progress := term.NewProgress(io.Discard)

				err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

				require.ErrorContains(t, err, test.wantError)
				assert.Equal(t, [][]string{{"inspect", "test-registry"}}, executor.runCalls)
			})
		}
	})

	t.Run("propagates a registry start failure", func(t *testing.T) {
		startError := errors.New("cannot start registry")
		executor := &registryExecutorFake{
			exists:        true,
			inspectOutput: `[{"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":"12345"}]}}}]`,
			runError:      startError,
		}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		assert.ErrorIs(t, err, startError)
	})

	t.Run("adds a diagnostic for a known port conflict and preserves engine output", func(t *testing.T) {
		runError := errors.New("registry run failed")
		executor := &registryExecutorFake{runError: runError, runOutput: "Error: Address ALREADY in use\n"}
		var output bytes.Buffer
		progress := term.NewProgress(&output)
		knownErrors := []string{"already in use", "already allocated"}

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, knownErrors)

		require.ErrorIs(t, err, runError)
		assert.ErrorContains(t, err, "port is already in use, this could be an existing test-registry or another process")
		assert.Contains(t, output.String(), executor.runOutput)
	})

	t.Run("propagates an unrecognised creation failure without a port conflict diagnostic", func(t *testing.T) {
		runError := errors.New("registry run failed")
		executor := &registryExecutorFake{runError: runError, runOutput: "permission denied\n"}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, []string{"address already in use"})

		assert.Same(t, runError, err)
	})

	t.Run("does not diagnose a port conflict when the command succeeds", func(t *testing.T) {
		executor := &registryExecutorFake{runOutput: "address already in use\n"}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, []string{"address already in use"})

		assert.NoError(t, err)
	})
}

type registryExecutorFake struct {
	exists          bool
	inspectOutput   string
	inspectExitCode int
	runError        error
	runOutput       string
	runCalls        [][]string
	commandCalls    [][]string
}

func (fake *registryExecutorFake) Command(_ context.Context, args ...string) *exec.Cmd {
	fake.commandCalls = append(fake.commandCalls, args)
	return testutil.CmdWithOutput(fake.inspectOutput, fake.inspectExitCode)
}

func (fake *registryExecutorFake) RunCommand(_ context.Context, output io.Writer, args ...string) error {
	fake.runCalls = append(fake.runCalls, args)
	if args[0] == "inspect" {
		if !fake.exists {
			return errors.New("container does not exist")
		}
		return nil
	}
	if _, err := io.WriteString(output, fake.runOutput); err != nil {
		return err
	}
	return fake.runError
}
