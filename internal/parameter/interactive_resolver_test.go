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
		resolver := parameter.NewInteractiveResolver(promptValues("hello", "8080"), term.NewPalette(false), nil)
		parameters := []parameter.Parameter{{Name: "GREETING", ExistingValue: new("old")}, {Name: "PORT"}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"GREETING": new("hello"), "PORT": new("8080")}, got)
	})

	t.Run("shows parameter details", func(t *testing.T) {
		var prompt term.Prompt
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false), nil)
		parameters := []parameter.Parameter{{Name: "GREETING", Description: "The greeting message", Example: "Hello"}}

		_, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, []string{
			"   1/1 GREETING", "", "       The greeting message", `       Example: "Hello"`, "",
		}, prompt.Content)
	})

	t.Run("does not create a change for an empty input on an absent parameter", func(t *testing.T) {
		resolver := parameter.NewInteractiveResolver(promptValues(""), term.NewPalette(false), nil)

		got, err := resolver.Resolve([]parameter.Parameter{{Name: "OPTIONAL"}})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("does not create a change for an accepted existing value", func(t *testing.T) {
		resolver := parameter.NewInteractiveResolver(acceptPromptInitial, term.NewPalette(false), nil)
		parameters := []parameter.Parameter{{Name: "GREETING", ExistingValue: new("hello")}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("retains an existing empty value when accepted", func(t *testing.T) {
		resolver := parameter.NewInteractiveResolver(acceptPromptInitial, term.NewPalette(false), nil)
		parameters := []parameter.Parameter{{Name: "GREETING", ExistingValue: new("")}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("unsets a cleared nonempty value", func(t *testing.T) {
		resolver := parameter.NewInteractiveResolver(promptValues(""), term.NewPalette(false), nil)
		parameters := []parameter.Parameter{{Name: "GREETING", ExistingValue: new("hello")}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"GREETING": nil}, got)
	})

	t.Run("shows reference paths and expressions", func(t *testing.T) {
		var prompt term.Prompt
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false), nil)
		parameters := []parameter.Parameter{{Name: "GREETING", References: []parameter.Reference{
			{Path: "app", Expression: "${GREETING:-default}/${GREETING:-other}"},
		}}}

		_, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Contains(t, prompt.Content, `         app: "${GREETING:-default}/${GREETING:-other}"`)
	})

	t.Run("allows empty input without required references", func(t *testing.T) {
		var prompt term.Prompt
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false), nil)
		parameters := []parameter.Parameter{{Name: "GREETING", References: []parameter.Reference{
			{Expression: "${GREETING:-hello}/${GREETING:-${OTHER:?required}}"},
		}}}

		_, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.True(t, prompt.Validate(""))
		assert.Equal(t, "   Press Enter to continue · Ctrl+C to cancel", prompt.Hint("", false))
	})

	t.Run("rejects missing required input", func(t *testing.T) {
		var prompt term.Prompt
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false), nil)
		parameters := []parameter.Parameter{{Name: "REQUIRED", References: []parameter.Reference{
			{Expression: "${REQUIRED:?required}"},
		}}}

		_, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.False(t, prompt.Validate(""))
		assert.Equal(t, "   Enter a value to continue · Ctrl+C to cancel", prompt.Hint("", false))
		assert.Equal(t, "   Press Enter to continue · Ctrl+C to cancel", prompt.Hint("value", false))
		assert.Equal(t, "   ✗ REQUIRED is required, enter a value to continue · Ctrl+C to cancel", prompt.Hint("", true))
	})

	t.Run("shows the detected value above editable input", func(t *testing.T) {
		var prompt term.Prompt
		prefillResolver := parameter.NewStaticResolver(parameter.Changes{"DEVICE_ARCH": new("aarch64")})
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false), prefillResolver)
		parameters := []parameter.Parameter{{
			Name: "DEVICE_ARCH", Description: "Architecture of the deployment target.",
		}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"DEVICE_ARCH": new("aarch64")}, got)
		assert.Equal(t, []string{
			"   1/1 DEVICE_ARCH",
			"",
			"       Architecture of the deployment target.",
			"",
			"   i Detected on target: aarch64",
		}, prompt.Content)
		assert.Equal(t, "aarch64", prompt.Initial)
		assert.Equal(t, "   Press Enter to continue · Ctrl+C to cancel", prompt.Hint("aarch64", false))
	})

	t.Run("keeps an existing env value while showing the detected alternative", func(t *testing.T) {
		var prompt term.Prompt
		prefillResolver := parameter.NewStaticResolver(parameter.Changes{"DEVICE_ARCH": new("aarch64")})
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false), prefillResolver)
		parameters := []parameter.Parameter{{Name: "DEVICE_ARCH", ExistingValue: new("arm64")}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Equal(t, "arm64", prompt.Initial)
		assert.Contains(t, prompt.Content, "   i Detected on target: aarch64")
		assert.Equal(t, new("aarch64"), prompt.RestoreValue)
	})

	t.Run("offers Ctrl+R only when the input differs from the detected value", func(t *testing.T) {
		var prompt term.Prompt
		prefillResolver := parameter.NewStaticResolver(parameter.Changes{"DEVICE_ARCH": new("aarch64")})
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false), prefillResolver)
		parameters := []parameter.Parameter{{Name: "DEVICE_ARCH", ExistingValue: new("arm64")}}

		_, err := resolver.Resolve(parameters)
		require.NoError(t, err)

		hintForExistingValue := prompt.Hint("arm64", false)
		hintForDetectedValue := prompt.Hint("aarch64", false)

		assert.Equal(t, "   Press Enter to continue · Ctrl+R to use detected value · Ctrl+C to cancel", hintForExistingValue)
		assert.Equal(t, "   Press Enter to continue · Ctrl+C to cancel", hintForDetectedValue)
	})

	t.Run("uses an edit instead of the detected value", func(t *testing.T) {
		prefillResolver := parameter.NewStaticResolver(parameter.Changes{"DEVICE_ARCH": new("aarch64")})
		resolver := parameter.NewInteractiveResolver(promptValues("arm64"), term.NewPalette(false), prefillResolver)

		got, err := resolver.Resolve([]parameter.Parameter{{Name: "DEVICE_ARCH"}})

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"DEVICE_ARCH": new("arm64")}, got)
	})

	t.Run("accepts an empty detected value for a presence-only requirement", func(t *testing.T) {
		var prompt term.Prompt
		prefillResolver := parameter.NewStaticResolver(parameter.Changes{"DEVICE_LABEL": new("")})
		resolver := parameter.NewInteractiveResolver(recordPrompt(&prompt), term.NewPalette(false), prefillResolver)
		parameters := []parameter.Parameter{{
			Name: "DEVICE_LABEL", References: []parameter.Reference{{Expression: "${DEVICE_LABEL?required}"}},
		}}

		got, err := resolver.Resolve(parameters)

		require.NoError(t, err)
		assert.True(t, prompt.Validate(""))
		assert.Equal(t, parameter.Changes{"DEVICE_LABEL": new("")}, got)
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
