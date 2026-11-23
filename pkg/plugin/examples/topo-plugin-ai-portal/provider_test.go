package main_test

import (
	"context"
	"encoding/json"
	"testing"

	portal "github.com/arm/topo/pkg/plugin/examples/topo-plugin-ai-portal"
	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider(t *testing.T) {
	t.Run("Suggestions", func(t *testing.T) {
		t.Run("offers model selection for model hints", func(t *testing.T) {
			for _, test := range []struct {
				name  string
				key   string
				value string
			}{
				{name: "task", key: "model.task", value: `"text-generation"`},
				{name: "format", key: "model.format", value: `"gguf"`},
				{name: "repository type", key: "meta.type", value: `"huggingface.repo-id"`},
				{name: "repository in type list", key: "meta.type", value: `["git.uri","huggingface.repo-id"]`},
			} {
				t.Run(test.name, func(t *testing.T) {
					provider := portal.Provider{}
					request := suggestions.Request{Hints: suggestions.Hints{test.key: json.RawMessage(test.value)}}

					got, err := provider.Suggestions(context.Background(), request)

					require.NoError(t, err)
					want := suggestions.Response{AvailableActions: []suggestions.Action{{
						ID: "choose_model", Label: "Choose a model in AI Portal",
						Description: "Open a browser filtered by project hints and observed target memory.",
					}}}
					assert.Equal(t, want, got)
				})
			}
		})

		t.Run("does not offer model selection for unrelated or unusable hints", func(t *testing.T) {
			for _, test := range []struct {
				name  string
				hints suggestions.Hints
			}{
				{name: "no hints", hints: suggestions.Hints{}},
				{name: "unrelated type", hints: suggestions.Hints{"meta.type": json.RawMessage(`"git.uri"`)}},
				{name: "empty task", hints: suggestions.Hints{"model.task": json.RawMessage(`" "`)}},
				{name: "invalid format", hints: suggestions.Hints{"model.format": json.RawMessage(`true`)}},
				{name: "invalid type", hints: suggestions.Hints{"meta.type": json.RawMessage(`123`)}},
				{name: "backend alone", hints: suggestions.Hints{"executorch.backend": json.RawMessage(`"xnnpack"`)}},
			} {
				t.Run(test.name, func(t *testing.T) {
					provider := portal.Provider{}

					got, err := provider.Suggestions(context.Background(), suggestions.Request{Hints: test.hints})

					require.NoError(t, err)
					assert.Equal(t, suggestions.Response{}, got)
				})
			}
		})

		t.Run("does not carry a model action into the next unrelated parameter", func(t *testing.T) {
			provider := portal.Provider{}
			model := suggestions.Request{Hints: suggestions.Hints{"model.task": json.RawMessage(`"text-generation"`)}}
			first, err := provider.Suggestions(context.Background(), model)
			require.NoError(t, err)
			require.NotEmpty(t, first.AvailableActions)

			got, err := provider.Suggestions(context.Background(), suggestions.Request{Hints: suggestions.Hints{}})

			require.NoError(t, err)
			assert.Equal(t, suggestions.Response{}, got)
		})

		t.Run("returns context cancellation", func(t *testing.T) {
			provider := portal.Provider{}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := provider.Suggestions(ctx, suggestions.Request{})

			assert.ErrorIs(t, err, context.Canceled)
		})
	})
}
