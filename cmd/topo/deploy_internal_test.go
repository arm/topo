package main

import (
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryConfig(t *testing.T) {
	t.Run("enables skipping the remote port check from the deploy flag", func(t *testing.T) {
		flag := deployCmd.Flags().Lookup("skip-remote-port-check")
		require.NotNil(t, flag)
		originalValue := flag.Value.String()
		originalChanged := flag.Changed
		t.Cleanup(func() {
			require.NoError(t, flag.Value.Set(originalValue))
			flag.Changed = originalChanged
		})
		require.NoError(t, flag.Value.Set("true"))
		flag.Changed = true

		got := newRegistryConfig(deployCmd, "12345", "linux")

		want := &deploy.RegistryConfig{
			Port:                "12345",
			SkipRemotePortCheck: true,
			UseControlSockets:   true,
		}
		assert.Equal(t, want, got)
	})
}
