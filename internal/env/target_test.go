package env_test

import (
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/stretchr/testify/assert"
)

func TestTargetEnv(t *testing.T) {
	t.Run("returns a map with the target and hostname", func(t *testing.T) {
		target := "ssh://user@foobar.example:8080"
		want := map[string]string{
			env.TargetVariable:         target,
			env.TargetHostnameVariable: "foobar.example",
		}

		got := env.TargetEnv(target)

		assert.Equal(t, want, got)
	})
}
