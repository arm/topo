package parameter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInteractiveProvider(t *testing.T) {
	t.Run("prompts for parameters and reads input", func(t *testing.T) {
		input := strings.NewReader("Hello, World\n8080\n")
		output := &bytes.Buffer{}
		provider := parameter.NewInteractiveProvider(input, output)

		definitions := []parameter.Definition{
			{
				Name:          "GREETING",
				Description:   "The greeting message",
				Required:      true,
				Example:       "Hello",
				CurrentValues: []string{"CURRENT GREETING HELLO!"},
			},
			{
				Name:        "PORT",
				Description: "Port number",
				Required:    false,
			},
		}

		got, err := provider.Provide(definitions)

		require.NoError(t, err)
		want := parameter.Values{
			"GREETING": "Hello, World",
			"PORT":     "8080",
		}
		assert.Equal(t, want, got)
		assert.Contains(t, output.String(), "The greeting message")
		assert.Contains(t, output.String(), "Example: Hello")
		assert.Contains(t, output.String(), "GREETING (required, leave blank to keep current)>")
	})

	t.Run("skips empty inputs", func(t *testing.T) {
		input := strings.NewReader("\n")
		output := &bytes.Buffer{}
		provider := parameter.NewInteractiveProvider(input, output)

		got, err := provider.Provide([]parameter.Definition{{Name: "OPTIONAL"}})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("shows current values", func(t *testing.T) {
		input := strings.NewReader("\n")
		output := &bytes.Buffer{}
		provider := parameter.NewInteractiveProvider(input, output)
		definitions := []parameter.Definition{{
			Name:          "GREETING",
			CurrentValues: []string{"Hello", ""},
		}}

		got, err := provider.Provide(definitions)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Contains(t, output.String(), `Current: ["Hello",""]`)
	})
}
