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

		parameters := []parameter.Parameter{
			{
				Name:        "GREETING",
				Description: "The greeting message",
				Required:    true,
				Example:     "Hello",
				Value:       new("CURRENT GREETING HELLO!"),
			},
			{
				Name:        "PORT",
				Description: "Port number",
				Required:    false,
			},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		want := parameter.Values{
			"GREETING": "Hello, World",
			"PORT":     "8080",
		}
		assert.Equal(t, want, got)
		assert.Contains(t, output.String(), "The greeting message")
		assert.Contains(t, output.String(), "Example: \"Hello\"")
		assert.Contains(t, output.String(), "1/2 GREETING")
		assert.Contains(t, output.String(), "i Leave empty to keep the current value.")
	})

	t.Run("skips empty inputs", func(t *testing.T) {
		input := strings.NewReader("\n")
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(input, output)

		got, err := resolver.Resolve([]parameter.Parameter{{Name: "OPTIONAL"}})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("shows current values", func(t *testing.T) {
		input := strings.NewReader("\n")
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(input, output)
		parameters := []parameter.Parameter{{
			Name:  "GREETING",
			Value: new("Hello"),
		}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Contains(t, output.String(), `Current: "Hello"`)
	})

	t.Run("shows an explicitly empty current value", func(t *testing.T) {
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(strings.NewReader("\n"), output)
		parameters := []parameter.Parameter{{Name: "GREETING", Value: new("")}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Contains(t, output.String(), `Current: ""`)
	})

	t.Run("re-prompts when a required value is missing", func(t *testing.T) {
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(strings.NewReader("\n \t\nprovided\nnext\n"), output)
		parameters := []parameter.Parameter{
			{Name: "REQUIRED", Required: true},
			{Name: "NEXT"},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Values{"REQUIRED": "provided", "NEXT": "next"}, got)
		assert.Equal(t, 2, strings.Count(output.String(), "✗ A value is required."))
	})
}
