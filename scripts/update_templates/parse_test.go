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
`
		compose := strings.NewReader(composeContent)

		tpl, err := BuildTemplate("git@github.com:Arm-Debug/example.git", compose)
		require.NoError(t, err)

		assert.Equal(t, Template{
			ID:          "Example-Template",
			Description: "Example description",
			Features:    []string{"SME", "NEON"},
			URL:         "git@github.com:Arm-Debug/example.git",
		}, tpl)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		composeContent := `x-topo:
  name: ""
  description: Example description
`
		compose := strings.NewReader(composeContent)

		_, err := BuildTemplate("git@github.com:Arm-Debug/example.git", compose)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "no valid x-topo")
	})
}
