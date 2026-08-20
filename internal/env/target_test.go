package env_test

import (
	"os"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetTargetEnv(t *testing.T) {
	t.Run("sets the resolved target environment", func(t *testing.T) {
		t.Setenv(env.TargetVariable, "ssh://stale.example")
		t.Setenv(env.TargetHostnameVariable, "stale.example")

		err := env.SetTargetEnv("user@target.example")

		require.NoError(t, err)
		assert.Equal(t, "ssh://user@target.example", os.Getenv(env.TargetVariable))
		assert.Equal(t, "target.example", os.Getenv(env.TargetHostnameVariable))
	})
}
