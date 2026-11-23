package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginDirectory(t *testing.T) {
	t.Run("uses the explicitly supplied discovery directory", func(t *testing.T) {
		cmd := &cobra.Command{}
		addPluginDirectoryFlag(cmd)
		require.NoError(t, cmd.ParseFlags([]string{"--plugin-dir", "./custom-plugins"}))

		got, err := pluginDirectory(cmd)

		require.NoError(t, err)
		assert.Equal(t, "./custom-plugins", got)
	})
}
