package arguments_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInteractiveProvider(t *testing.T) {
	t.Run("prompts for arguments and reads input", func(t *testing.T) {
		input := strings.NewReader("Hello, World\n8080\n")
		output := &bytes.Buffer{}
		provider := arguments.NewInteractiveProvider(input, output)

		args := []arguments.Arg{
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

		got, err := provider.Provide(args)

		require.NoError(t, err)
		assert.Equal(t, "Hello, World", got["GREETING"])
		assert.Equal(t, "8080", got["PORT"])
		assert.Contains(t, output.String(), "The greeting message")
		assert.Contains(t, output.String(), "Example: Hello")
		assert.Contains(t, output.String(), "GREETING (required)>")
	})

	t.Run("skips empty inputs", func(t *testing.T) {
		input := strings.NewReader("\n")
		output := &bytes.Buffer{}
		provider := arguments.NewInteractiveProvider(input, output)

		args := []arguments.Arg{
			{Name: "OPTIONAL", Required: false},
		}

		got, err := provider.Provide(args)

		require.NoError(t, err)
		assert.Empty(t, got["OPTIONAL"])
	})

	t.Run("returns correct name", func(t *testing.T) {
		provider := arguments.NewInteractiveProvider(nil, nil)
		assert.Equal(t, "interactive", provider.Name())
	})
}
