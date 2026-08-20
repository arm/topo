package main

import (
	"os"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireTarget(t *testing.T) {
	t.Run("sets the resolved target environment", func(t *testing.T) {
		t.Setenv(env.TargetVariable, "ssh://stale.example")
		t.Setenv(env.TargetHostnameVariable, "stale.example")
		command := &cobra.Command{}
		addTargetFlag(command)
		require.NoError(t, command.Flags().Set("target", "user@target.example"))

		got, err := requireTarget(command)

		require.NoError(t, err)
		assert.Equal(t, "user@target.example", got)
		assert.Equal(t, "ssh://user@target.example", os.Getenv(env.TargetVariable))
		assert.Equal(t, "target.example", os.Getenv(env.TargetHostnameVariable))
	})

	t.Run("derives the hostname when the target comes from the environment", func(t *testing.T) {
		t.Setenv(env.TargetVariable, "user@environment-target.example")
		command := &cobra.Command{}
		addTargetFlag(command)

		got, err := requireTarget(command)

		require.NoError(t, err)
		assert.Equal(t, "user@environment-target.example", got)
		assert.Equal(t, "ssh://user@environment-target.example", os.Getenv(env.TargetVariable))
		assert.Equal(t, "environment-target.example", os.Getenv(env.TargetHostnameVariable))
	})
}
