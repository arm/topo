package core

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectFeatures(t *testing.T) {
	t.Run("collect features successfully", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			return "Features: fpu asimd sve", nil
		}
		target := Target{"", nil, nil, mockExec}

		err := target.collectFeatures()
		assert.NoError(t, err)
		assert.Equal(t, []string{"Features:", "fpu", "asimd", "sve"}, target.features)
	})

	t.Run("collect features with malformed output", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			return "", nil
		}
		target := Target{"", nil, nil, mockExec}

		err := target.collectFeatures()
		assert.NoError(t, err)
		assert.Empty(t, target.features)
	})

	t.Run("collect features returns an error", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			return "", fmt.Errorf("failed to get features")
		}
		target := Target{"", nil, nil, mockExec}

		err := target.collectFeatures()
		assert.EqualError(t, err, "failed to get features")
	})
}

func TestRun(t *testing.T) {
	t.Run("run executes command successfully", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			return "success", nil
		}
		target := Target{"device1", nil, nil, mockExec}

		out, err := target.Run("ls")
		assert.NoError(t, err)
		assert.Equal(t, "success", out)
	})

	t.Run("run returns error", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			return "", errors.New("ssh failed")
		}
		target := Target{"device2", nil, nil, mockExec}

		out, err := target.Run("ls")
		assert.Error(t, err)
		assert.Empty(t, out)
	})
}

func TestMakeTarget(t *testing.T) {
	t.Run("make target succeeds and collects features", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			if command == "" {
				return "", nil // simulate successful initial connection
			}
			return "Features: fpu asimd", nil
		}

		target := MakeTarget("device3", mockExec)
		assert.NoError(t, target.connectionError)
		assert.Equal(t, []string{"Features:", "fpu", "asimd"}, target.features)
	})

	t.Run("make target fails connection", func(t *testing.T) {
		mockExec := func(target, command string) (string, error) {
			return "", fmt.Errorf("connection refused")
		}

		target := MakeTarget("device4", mockExec)
		assert.Error(t, target.connectionError)
		assert.EqualError(t, target.connectionError, "connection refused")
	})
}
