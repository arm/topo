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
				Name:          "GREETING",
				Description:   "The greeting message",
				Example:       "Hello",
				ExistingValue: new("CURRENT GREETING HELLO!"),
			},
			{
				Name:        "PORT",
				Description: "Port number",
			},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		want := parameter.Changes{
			"GREETING": new("Hello, World"),
			"PORT":     new("8080"),
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
			Name:          "GREETING",
			ExistingValue: new("Hello"),
		}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Contains(t, output.String(), `Current: "Hello"`)
	})

	t.Run("shows an explicitly empty current value", func(t *testing.T) {
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(strings.NewReader("\n"), output)
		parameters := []parameter.Parameter{{Name: "GREETING", ExistingValue: new("")}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Contains(t, output.String(), `Current: ""`)
	})

	t.Run("shows paths and complete usage expressions", func(t *testing.T) {
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(strings.NewReader("\n"), output)
		parameters := []parameter.Parameter{{
			Name: "GREETING",
			References: []parameter.Reference{
				{Path: "services.app.build.args.GREETING", Expression: "${GREETING:-hello}/${GREETING:-hi}"},
				{Path: "services.app.command[1]", Expression: "prefix-${GREETING:-${OTHER}}"},
			},
		}}

		_, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Contains(t, output.String(), `References:
      services.app.build.args.GREETING: "${GREETING:-hello}/${GREETING:-hi}"
      services.app.command[1]: "prefix-${GREETING:-${OTHER}}"`)
	})

	t.Run("allows skipping parameters with no required references", func(t *testing.T) {
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(strings.NewReader("\nnext\n"), output)
		parameters := []parameter.Parameter{
			{Name: "GREETING", References: []parameter.Reference{{Expression: "${GREETING:-hello}/${GREETING:-${OTHER:?required}}"}}},
			{Name: "NEXT"},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"NEXT": new("next")}, got)
	})

	t.Run("re-prompts when a required value is missing", func(t *testing.T) {
		output := &bytes.Buffer{}
		resolver := parameter.NewInteractiveResolver(strings.NewReader("\n \t\nprovided\nnext\n"), output)
		parameters := []parameter.Parameter{
			{Name: "REQUIRED", References: []parameter.Reference{{Expression: "${REQUIRED:?required}"}}},
			{Name: "NEXT"},
		}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"REQUIRED": new("provided"), "NEXT": new("next")}, got)
		assert.Equal(t, 2, strings.Count(output.String(), "✗ A value is required."))
	})
}
