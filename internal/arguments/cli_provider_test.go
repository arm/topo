package arguments

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIProvider(t *testing.T) {
	t.Run("parses valid arguments", func(t *testing.T) {
		provider, err := NewCLIProvider([]string{"GREETING=Hello", "PORT=8080"})
		require.NoError(t, err)

		specs := []service.ArgSpec{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}

		got, err := provider.Provide(specs)

		require.NoError(t, err)
		want := map[string]string{
			"GREETING": "Hello",
			"PORT":     "8080",
		}
		assert.Equal(t, want, got)
	})

	t.Run("allows values with equals signs", func(t *testing.T) {
		provider, err := NewCLIProvider([]string{"CONNECTION_STRING=host=localhost;port=5432"})
		require.NoError(t, err)

		specs := []service.ArgSpec{
			{Name: "CONNECTION_STRING", Required: true},
		}

		got, err := provider.Provide(specs)

		require.NoError(t, err)
		assert.Equal(t, "host=localhost;port=5432", got["CONNECTION_STRING"])
	})

	t.Run("errors on invalid format", func(t *testing.T) {
		_, err := NewCLIProvider([]string{"INVALID"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid argument format")
	})

	t.Run("errors on unknown argument", func(t *testing.T) {
		provider, err := NewCLIProvider([]string{"UNKNOWN=value"})
		require.NoError(t, err)

		specs := []service.ArgSpec{
			{Name: "GREETING", Required: true},
		}

		_, err = provider.Provide(specs)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown argument: UNKNOWN")
	})

	t.Run("returns correct name", func(t *testing.T) {
		provider, _ := NewCLIProvider([]string{})
		assert.Equal(t, "cli", provider.Name())
	})
}
