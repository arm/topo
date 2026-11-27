package plugins_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/arm/topo/internal/plugins"
	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/arm/topo/pkg/plugin/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("requires building and launching a plugin executable")
	}

	t.Run("suggestions plugin", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "topo-plugin-suggest-color.exe")
		build := exec.CommandContext(
			t.Context(), "go", "build", "-mod=readonly", "-o", path, "./testdata/topo-plugin-suggest-color",
		)
		output, err := build.CombinedOutput()
		require.NoError(t, err, "%s", output)
		ctx, stop := context.WithTimeout(t.Context(), 20*time.Second)
		defer stop()
		request := suggestions.Request{Project: suggestions.ProjectMetadata{Name: "Color picker"}}

		loaded, err := plugins.Load(ctx, path)

		require.NoError(t, err)
		t.Cleanup(loaded.Close)
		assert.Equal(t, transport.Descriptor{ID: "test.suggest-color", Capability: transport.Suggestions}, loaded.Descriptor())

		listed, err := loaded.Suggestions(ctx, request)

		require.NoError(t, err)
		assert.Equal(t, suggestions.Response{
			Suggestions: []suggestions.Suggestion{
				{Value: "blue", Label: "Blue"},
				{Value: "green", Label: "Green"},
			},
			AvailableActions: []suggestions.Action{{ID: "use-green", Label: "Use green"}},
		}, listed)

		selected, err := loaded.RunAction(ctx, "use-green", request)

		require.NoError(t, err)
		want := suggestions.ActionResponse{Suggestions: []suggestions.Suggestion{{Value: "green", Label: "Green"}}}
		assert.Equal(t, want, selected)

		loaded.Close()

		assert.True(t, loaded.Exited(), "plugin process is still running")
	})
}
