package env_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/stretchr/testify/assert"
)

func TestIsEnvVarTruthy(t *testing.T) {
	t.Run("returns true if env variable is set to truthy value", func(t *testing.T) {
		truthy_values := []string{
			"1",
			"On",
			"TRUE",
			"Yes",
			"enabled",
			"tRuE",
			"true",
			"y",
			"yes",
		}

		for _, value := range truthy_values {
			description := fmt.Sprintf("%q is considered truthy", value)
			env_var := "SOME_VAR"
			t.Run(description, func(t *testing.T) {
				t.Setenv(env_var, value)

				assert.True(t, env.IsVarTruthy(env_var))
			})
		}
	})

	t.Run("returns false if env variable is not set", func(t *testing.T) {
		assert.False(t, env.IsVarTruthy("NOT_SET"))
	})

	t.Run("returns false if env variable is set to falsy value", func(t *testing.T) {
		assert.False(t, env.IsVarTruthy("false"))
	})
}
