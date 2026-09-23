package parameter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInteractiveResolver(t *testing.T) {
	t.Run("prompts for parameters and reads input", func(t *testing.T) {
		input := strings.NewReader("Hello, World\n8080\n")
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(input, output)

		definitions := []parameter.Definition{
			{
				Name:        "GREETING",
				Description: "The greeting message",
				Required:    true,
				Example:     "Hello",
			},
			{
				Name:        "PORT",
				Description: "Port number",
				Required:    false,
			},
		}

		got, err := resolver.Resolve(definitions, parameter.Values{"GREETING": "CURRENT GREETING HELLO!"})

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
		resolver := parameter.NewInteractiveResolver(input, output)

		got, err := resolver.Resolve([]parameter.Definition{{Name: "OPTIONAL"}}, nil)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("shows current values", func(t *testing.T) {
		input := strings.NewReader("\n")
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(input, output)
		definitions := []parameter.Definition{{
			Name: "GREETING",
		}}

		got, err := resolver.Resolve(definitions, parameter.Values{"GREETING": "Hello"})

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Contains(t, output.String(), `Current: "Hello"`)
	})

	t.Run("shows an explicitly empty current value", func(t *testing.T) {
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(strings.NewReader("\n"), output)
		currentValues := parameter.Values{"GREETING": ""}

		got, err := resolver.Resolve([]parameter.Definition{{Name: "GREETING"}}, currentValues)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Contains(t, output.String(), `Current: ""`)
	})
}
