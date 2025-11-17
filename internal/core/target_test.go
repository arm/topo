package core_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	t.Run("run executes command successfully", func(t *testing.T) {
		mockExec := func(_, _ string) (string, error) {
			return "success", nil
		}
		tc := core.NewTargetConnection("hostname", mockExec)

		out, err := tc.Run("ls")

		assert.NoError(t, err)
		assert.Equal(t, "success", out)
	})

	t.Run("run returns error", func(t *testing.T) {
		mockExec := func(_, _ string) (string, error) {
			return "", errors.New("ssh failed")
		}
		tc := core.NewTargetConnection("hostname", mockExec)

		out, err := tc.Run("ls")

		assert.Error(t, err)
		assert.Empty(t, out)
	})
}

func TestProbe(t *testing.T) {
	t.Run("probe succeeds and collects features", func(t *testing.T) {
		mockExec := func(_, command string) (string, error) {
			if command == "" {
				return "", nil // simulate successful initial connection
			}
			return "Features: fpu asimd", nil
		}

		tc := core.NewTargetConnection("hostname", mockExec)
		ts := tc.Probe()

		assert.NoError(t, ts.ConnectionError)
		assert.Equal(t, []string{"fpu", "asimd"}, ts.Hardware.Features)
	})

	t.Run("probe succeeds but features collection returns empty", func(t *testing.T) {
		mockExec := func(_, command string) (string, error) {
			if command == "" {
				return "", nil
			}
			return "", nil
		}

		tc := core.NewTargetConnection("hostname", mockExec)
		ts := tc.Probe()

		assert.NoError(t, ts.ConnectionError)
		assert.Empty(t, ts.Hardware.Features)
	})

	t.Run("probe fails connection", func(t *testing.T) {
		mockExec := func(_, _ string) (string, error) {
			return "", fmt.Errorf("connection refused")
		}

		tc := core.NewTargetConnection("hostname", mockExec)
		ts := tc.Probe()

		assert.Error(t, ts.ConnectionError)
		assert.EqualError(t, ts.ConnectionError, "connection refused")
	})

	t.Run("probe finds remote cpu", func(t *testing.T) {
		mockExec := func(_, command string) (string, error) {
			if strings.Contains(command, "remoteproc") {
				return "foo\nbar", nil
			}
			return "", nil
		}

		tc := core.NewTargetConnection("hostname", mockExec)
		ts := tc.Probe()

		want := []string{"foo", "bar"}
		assert.Equal(t, want, ts.Hardware.RemoteCPU)
	})
}

func TestBinaryExists(t *testing.T) {
	t.Run("when binary found, returns true", func(t *testing.T) {
		mockExec := func(_, _ string) (string, error) {
			return "/foo/bar", nil
		}
		tc := core.NewTargetConnection("hostname", mockExec)

		got, err := tc.BinaryExists("bar")

		assert.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("invalid format returns an error", func(t *testing.T) {
		mockExec := func(_, _ string) (string, error) {
			return "/foo/bar", nil
		}
		tc := core.NewTargetConnection("hostname", mockExec)

		got, err := tc.BinaryExists("b a r")

		assert.Error(t, err)
		assert.False(t, got)
	})
}
