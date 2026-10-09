package parameter_test

import (
	"testing"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInteractiveResolver(t *testing.T) {
	t.Run("resolves parameters in order", func(t *testing.T) {
		resolver := parameter.NewInteractiveResolver(promptValues("hello", "8080"), term.NewPalette(false))
		parameters := []parameter.Parameter{{Name: "GREETING", ExistingValue: new("old")}, {Name: "PORT"}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"GREETING": new("hello"), "PORT": new("8080")}, got)
	})

	t.Run("shows parameter details", func(t *testing.T) {
		var prompt term.Prompt
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false))
		parameters := []parameter.Parameter{{Name: "GREETING", Description: "The greeting message", Example: "Hello"}}

		_, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"1/1 GREETING", "", "    The greeting message", `    Example: "Hello"`, "", "✓ Press enter to continue.",
		}, prompt.Content(""))
	})

	t.Run("does not create a change for an empty input on an absent parameter", func(t *testing.T) {
		resolver := parameter.NewInteractiveResolver(promptValues(""), term.NewPalette(false))

		got, err := resolver.Resolve([]parameter.Parameter{{Name: "OPTIONAL"}})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("does not create a change for an accepted existing value", func(t *testing.T) {
		resolver := parameter.NewInteractiveResolver(acceptPromptInitial, term.NewPalette(false))
		parameters := []parameter.Parameter{{Name: "GREETING", ExistingValue: new("hello")}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("retains an existing empty value when accepted", func(t *testing.T) {
		resolver := parameter.NewInteractiveResolver(acceptPromptInitial, term.NewPalette(false))
		parameters := []parameter.Parameter{{Name: "GREETING", ExistingValue: new("")}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("unsets a cleared nonempty value", func(t *testing.T) {
		resolver := parameter.NewInteractiveResolver(promptValues(""), term.NewPalette(false))
		parameters := []parameter.Parameter{{Name: "GREETING", ExistingValue: new("hello")}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"GREETING": nil}, got)
	})

	t.Run("shows reference paths and expressions", func(t *testing.T) {
		var prompt term.Prompt
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false))
		parameters := []parameter.Parameter{{Name: "GREETING", References: []parameter.Reference{
			{Path: "app", Expression: "${GREETING:-default}/${GREETING:-other}"},
		}}}

		_, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Contains(t, prompt.Content("hello"), `      app: "hello/hello"`)
		assert.Contains(t, prompt.Content("hello"), "           ${GREETING:-default}/${GREETING:-other}")
	})

	t.Run("allows empty input without required references", func(t *testing.T) {
		var prompt term.Prompt
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false))
		parameters := []parameter.Parameter{{Name: "GREETING", References: []parameter.Reference{
			{Expression: "${GREETING:-hello}/${GREETING:-${OTHER:?required}}"},
		}}}

		_, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.True(t, prompt.Validate(""))
	})

	t.Run("rejects missing required input", func(t *testing.T) {
		var prompt term.Prompt
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false))
		parameters := []parameter.Parameter{{Name: "REQUIRED", References: []parameter.Reference{
			{Expression: "${REQUIRED:?required}"},
		}}}

		_, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.False(t, prompt.Validate(""))
	})
}

func promptValues(values ...string) func(term.Prompt) (string, error) {
	return func(term.Prompt) (string, error) {
		value := values[0]
		values = values[1:]
		return value, nil
	}
}

func recordPrompt(captured *term.Prompt) func(term.Prompt) (string, error) {
	return func(prompt term.Prompt) (string, error) {
		*captured = prompt
		return acceptPromptInitial(prompt)
	}
}

func acceptPromptInitial(prompt term.Prompt) (string, error) {
	return prompt.Initial, nil
}
