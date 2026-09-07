package parameter_test

import (
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIResolver(t *testing.T) {
	t.Run("parses valid parameters", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver([]string{"GREETING=Hello", "PORT=8080"})
		require.NoError(t, err)

		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}

		got, err := resolver.Resolve(definitions)

		require.NoError(t, err)
		want := parameter.Values{
			"GREETING": "Hello",
			"PORT":     "8080",
		}
		assert.Equal(t, want, got)
	})

	t.Run("allows values with equals signs", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver([]string{"CONNECTION_STRING=host=localhost;port=5432"})
		require.NoError(t, err)

		definitions := []parameter.Definition{
			{Name: "CONNECTION_STRING", Required: true},
		}

		got, err := resolver.Resolve(definitions)

		require.NoError(t, err)
		want := parameter.Values{
			"CONNECTION_STRING": "host=localhost;port=5432",
		}
		assert.Equal(t, want, got)
	})

	t.Run("errors on invalid format", func(t *testing.T) {
		_, err := parameter.NewCLIResolver([]string{"INVALID"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid parameter format")
	})

	t.Run("errors on unknown parameter", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver([]string{"UNKNOWN=value"})
		require.NoError(t, err)

		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
		}

		_, err = resolver.Resolve(definitions)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown parameter: UNKNOWN")
	})

	t.Run("returns values for all known parameters", func(t *testing.T) {
		resolver, err := parameter.NewCLIResolver([]string{"PORT=8080", "GREETING=Hello", "NAME=Topo"})
		require.NoError(t, err)

		definitions := []parameter.Definition{
			{Name: "NAME", Required: true},
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: true},
		}

		got, err := resolver.Resolve(definitions)

		require.NoError(t, err)
		want := parameter.Values{
			"NAME":     "Topo",
			"GREETING": "Hello",
			"PORT":     "8080",
		}
		assert.Equal(t, want, got)
	})
}
