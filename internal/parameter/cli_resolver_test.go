package parameter_test

import (
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIResolver(t *testing.T) {
	t.Run("preserves an explicitly empty value", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver([]string{"GREETING="}, nil)
		require.NoError(t, err)
		parameters := []parameter.Parameter{{Name: "GREETING"}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"GREETING": new("")}, got)
	})

	t.Run("parses valid parameters", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver([]string{"GREETING=Hello", "PORT=8080"}, nil)
		require.NoError(t, err)

		parameters := []parameter.Parameter{
			{Name: "GREETING"},
			{Name: "PORT"},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		want := parameter.Changes{
			"GREETING": new("Hello"),
			"PORT":     new("8080"),
		}
		assert.Equal(t, want, got)
	})

	t.Run("allows values with equals signs", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver([]string{"CONNECTION_STRING=host=localhost;port=5432"}, nil)
		require.NoError(t, err)

		parameters := []parameter.Parameter{
			{Name: "CONNECTION_STRING"},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		want := parameter.Changes{
			"CONNECTION_STRING": new("host=localhost;port=5432"),
		}
		assert.Equal(t, want, got)
	})

	t.Run("errors on invalid format", func(t *testing.T) {
		_, err := parameter.NewCLIResolver([]string{"INVALID"}, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid parameter format")
	})

	t.Run("errors on unknown parameter", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver([]string{"UNKNOWN=value"}, nil)
		require.NoError(t, err)

		parameters := []parameter.Parameter{
			{Name: "GREETING"},
		}

		_, err = resolver.Resolve(parameters)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown parameter: UNKNOWN")
	})

	t.Run("returns values for all known parameters", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver([]string{"PORT=8080", "GREETING=Hello", "NAME=Topo"}, nil)
		require.NoError(t, err)

		parameters := []parameter.Parameter{
			{Name: "NAME"},
			{Name: "GREETING"},
			{Name: "PORT"},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		want := parameter.Changes{
			"NAME":     new("Topo"),
			"GREETING": new("Hello"),
			"PORT":     new("8080"),
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns removals as nil values", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver(nil, []string{"FOO"})
		require.NoError(t, err)

		got, err := resolver.Resolve([]parameter.Parameter{{Name: "FOO"}})

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"FOO": nil}, got)
	})

	t.Run("rejects setting and removing the same parameter", func(t *testing.T) {
		_, err := parameter.NewCLIResolver([]string{"FOO="}, []string{"FOO"})

		require.ErrorContains(t, err, "cannot both set and unset parameter: FOO")
	})

	t.Run("rejects unknown removal names", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver(nil, []string{"UNKNOWN"})
		require.NoError(t, err)

		_, err = resolver.Resolve(nil)

		require.ErrorContains(t, err, "unknown parameter: UNKNOWN")
	})
}
