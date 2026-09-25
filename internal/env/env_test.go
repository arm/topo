package env_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrentValues(t *testing.T) {
	t.Run("process values override file values including with an empty value", func(t *testing.T) {
		for _, value := range []string{"shell=value", ""} {
			t.Run(fmt.Sprintf("value %q", value), func(t *testing.T) {
				t.Setenv("TOPO_TEST_PARAMETER", value)
				path := filepath.Join(t.TempDir(), ".env")
				testutil.RequireWriteFile(t, path, "TOPO_TEST_PARAMETER=file\n")

				values, err := env.CurrentValues([]string{path})

				require.NoError(t, err)
				assert.Contains(t, values, "TOPO_TEST_PARAMETER")
				assert.Equal(t, value, values["TOPO_TEST_PARAMETER"])
			})
		}
	})

	t.Run("returns process values without files", func(t *testing.T) {
		t.Setenv("TOPO_TEST_PARAMETER", "shell=value")

		values, err := env.CurrentValues(nil)

		require.NoError(t, err)
		assert.Equal(t, "shell=value", values["TOPO_TEST_PARAMETER"])
	})

	t.Run("propagates file read errors", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing")

		values, err := env.CurrentValues([]string{path})

		require.ErrorContains(t, err, "failed to read env files")
		assert.Nil(t, values)
	})
}

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
