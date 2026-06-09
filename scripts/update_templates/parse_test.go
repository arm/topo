package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTemplate(t *testing.T) {
	t.Run("builds template", func(t *testing.T) {
		composeContent := `x-topo:
  name: Example Template
  description: Example description
  features:
    - SME
    - NEON
  args:
    MODEL:
      description: Model ID
      required: true
      default: qwen3:7B
      example: qwen3:0.6B
      hints:
        org.example.kind: model
        org.example.tags: [llm, cpu]
`
		compose := strings.NewReader(composeContent)
		source := Source{Repo: "Spaghetti/bolognese", SHA: "fresh"}

		tpl, err := NewTemplate(source, compose)
		require.NoError(t, err)

		assert.Equal(t, Template{
			Name:        "Example Template",
			Description: "Example description",
			Features:    []string{"SME", "NEON"},
			Args: map[string]Arg{
				"MODEL": {
					Description: "Model ID",
					Required:    true,
					Default:     "qwen3:7B",
					Example:     "qwen3:0.6B",
					Hints: map[string]any{
						"org.example.kind": "model",
						"org.example.tags": []any{"llm", "cpu"},
					},
				},
			},
			URL: source.URL(),
			Ref: source.SHA,
		}, tpl)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		composeContent := `x-topo:
  name: ""
  description: Example description
`
		compose := strings.NewReader(composeContent)

		_, err := NewTemplate(Source{}, compose)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "no valid x-topo")
	})
}
