package main_test

import (
	"encoding/json"
	"fmt"
	"testing"

	portal "github.com/arm/topo/pkg/plugin/examples/topo-plugin-ai-portal"
	"github.com/arm/topo/pkg/plugin/observations"
	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildModelQuery(t *testing.T) {
	t.Run("trims project names while preserving their contents", func(t *testing.T) {
		request := suggestions.Request{Project: suggestions.ProjectMetadata{
			Name: " \t模型 + demo:  50% & /?#\u2003",
		}}

		got := portal.BuildModelQuery(request)

		assert.Equal(t, portal.ModelQuery{ProjectName: "模型 + demo:  50% & /?#"}, got)
	})

	t.Run("omits whitespace-only project names", func(t *testing.T) {
		request := suggestions.Request{Project: suggestions.ProjectMetadata{Name: " \t\n\u2003"}}

		got := portal.BuildModelQuery(request)

		assert.Equal(t, portal.ModelQuery{}, got)
	})

	t.Run("uses ONNX text generator hints without assuming a backend", func(t *testing.T) {
		request := suggestions.Request{Hints: suggestions.Hints{
			"meta.type":    json.RawMessage(`"huggingface.repo-id"`),
			"model.task":   json.RawMessage(`"text-generation"`),
			"model.format": json.RawMessage(`"onnx"`),
		}}

		got := portal.BuildModelQuery(request)

		want := portal.ModelQuery{Task: "text-generation", Format: "onnx", Runtime: "onnx"}
		assert.Equal(t, want, got)
	})

	t.Run("uses ExecuTorch segmentation hints", func(t *testing.T) {
		request := suggestions.Request{Hints: suggestions.Hints{
			"model.task":         json.RawMessage(`"image-segmentation"`),
			"model.format":       json.RawMessage(`"pte"`),
			"executorch.backend": json.RawMessage(`"xnnpack"`),
		}}

		got := portal.BuildModelQuery(request)

		want := portal.ModelQuery{
			Task: "image-segmentation", Format: "pte", Runtime: "executorch", Backend: "cpu",
		}
		assert.Equal(t, want, got)
	})

	t.Run("converts namespaced target memory bytes to whole decimal MB", func(t *testing.T) {
		var tree observations.Tree
		err := json.Unmarshal([]byte(`{
			"example": {"memory": {"target": {"memory": {"total_bytes": 4294967296}}}}
		}`), &tree)
		require.NoError(t, err)
		request := suggestions.Request{TargetObservations: tree}

		got := portal.BuildModelQuery(request)

		assert.Equal(t, portal.ModelQuery{PeakMemoryMB: 4294}, got)
	})

	t.Run("leaves missing metadata unfiltered", func(t *testing.T) {
		request := suggestions.Request{}

		got := portal.BuildModelQuery(request)

		assert.Equal(t, portal.ModelQuery{}, got)
	})

	t.Run("ignores unsupported hint types and unknown hints", func(t *testing.T) {
		request := suggestions.Request{Hints: suggestions.Hints{
			"model.task":   json.RawMessage(`["text-generation"]`),
			"model.format": json.RawMessage(`true`),
			"other.hint":   json.RawMessage(`"ignored"`),
		}}

		got := portal.BuildModelQuery(request)

		assert.Equal(t, portal.ModelQuery{}, got)
	})

	t.Run("does not infer runtime for unknown formats or use unrelated backend hints", func(t *testing.T) {
		request := suggestions.Request{Hints: suggestions.Hints{
			"model.format":       json.RawMessage(`"safetensors"`),
			"executorch.backend": json.RawMessage(`"xnnpack"`),
		}}

		got := portal.BuildModelQuery(request)

		assert.Equal(t, portal.ModelQuery{Format: "safetensors"}, got)
	})

	t.Run("maps ExecuTorch delegates to Portal execution backends", func(t *testing.T) {
		for _, test := range []struct {
			name    string
			hint    string
			backend string
		}{
			{name: "preserves Ethos-U", hint: `"ethos-u"`, backend: "ethos-u"},
			{name: "normalizes backend", hint: `" Ethos-U "`, backend: "ethos-u"},
			{name: "maps other delegates to CPU", hint: `"other-delegate"`, backend: "cpu"},
			{name: "ignores empty backend", hint: `" "`, backend: ""},
		} {
			t.Run(test.name, func(t *testing.T) {
				request := suggestions.Request{Hints: suggestions.Hints{
					"model.format":       json.RawMessage(`"pte"`),
					"executorch.backend": json.RawMessage(test.hint),
				}}

				got := portal.BuildModelQuery(request)

				want := portal.ModelQuery{Format: "pte", Runtime: "executorch", Backend: test.backend}
				assert.Equal(t, want, got)
			})
		}
	})

	t.Run("ignores malformed ExecuTorch backend", func(t *testing.T) {
		request := suggestions.Request{Hints: suggestions.Hints{
			"model.format":       json.RawMessage(`"pte"`),
			"executorch.backend": json.RawMessage(`123`),
		}}

		got := portal.BuildModelQuery(request)

		assert.Equal(t, portal.ModelQuery{Format: "pte", Runtime: "executorch"}, got)
	})

	t.Run("omits unusable memory cutoffs", func(t *testing.T) {
		for _, value := range []any{nil, "4294967296", int64(-1), int64(0), int64(999999)} {
			t.Run(fmt.Sprintf("%T %v", value, value), func(t *testing.T) {
				request := suggestions.Request{TargetObservations: observations.Tree{
					"example": observations.Tree{"memory": observations.Tree{
						"target": observations.Tree{"memory": observations.Tree{"total_bytes": value}},
					}},
				}}

				got := portal.BuildModelQuery(request)

				assert.Equal(t, portal.ModelQuery{}, got)
			})
		}
	})
}
