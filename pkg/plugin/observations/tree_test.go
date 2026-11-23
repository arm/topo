package observations_test

import (
	"encoding/json"
	"testing"

	"github.com/arm/topo/pkg/plugin/observations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTree(t *testing.T) {
	t.Run("Decode", func(t *testing.T) {
		t.Run("decodes selected fields into Go types", func(t *testing.T) {
			tree := observations.Tree{"arm": observations.Tree{"fancy": observations.Tree{
				"target": observations.Tree{
					"cpu_count": 8, "memory_bytes": int64(9007199254740993),
					"os":       observations.Tree{"name": "linux"},
					"features": []string{"avx2", "aes"}, "extra": true,
				},
			}}}
			type system struct {
				CPUCount    int   `json:"cpu_count"`
				MemoryBytes int64 `json:"memory_bytes"`
				OS          struct {
					Name string `json:"name"`
				} `json:"os"`
				Features []string `json:"features"`
				Missing  *int     `json:"missing"`
			}
			var got system

			err := tree.Decode("arm.fancy.target", &got)

			require.NoError(t, err)
			want := system{CPUCount: 8, MemoryBytes: 9007199254740993,
				OS: struct {
					Name string `json:"name"`
				}{Name: "linux"}, Features: []string{"avx2", "aes"}}
			assert.Equal(t, want, got)
		})
		t.Run("reports missing path", func(t *testing.T) {
			tree := observations.Tree{}
			var got struct{}

			err := tree.Decode("arm.fancy.target", &got)

			assert.ErrorContains(t, err, `observation "arm.fancy.target" unavailable`)
		})
		t.Run("reports malformed JSON", func(t *testing.T) {
			tree := observations.Tree{"arm": json.RawMessage(`{`)}
			var got struct{}

			err := tree.Decode("arm.fancy.target", &got)

			assert.ErrorContains(t, err, "encoding observations")
		})
		t.Run("decodes a nested path without losing precision", func(t *testing.T) {
			tree := observations.Tree{"arm": observations.Tree{"fancy": observations.Tree{
				"target": observations.Tree{"cpu": observations.Tree{"count": int64(9007199254740993)}},
			}}}
			var got int64

			err := tree.Decode("arm.fancy.target.cpu.count", &got)

			require.NoError(t, err)
			assert.Equal(t, int64(9007199254740993), got)
		})
		for _, scenario := range []struct {
			name string
			data string
		}{
			{name: "missing key", data: `{}`},
			{name: "incompatible type", data: `{"cpu_count":"eight"}`},
			{name: "integer overflow", data: `{"cpu_count":9223372036854775808}`},
			{name: "fractional integer", data: `{"cpu_count":1.5}`},
		} {
			t.Run(scenario.name, func(t *testing.T) {
				tree := observations.Tree{"arm": observations.Tree{"fancy": observations.Tree{
					"target": json.RawMessage(scenario.data),
				}}}
				var got int64

				err := tree.Decode("arm.fancy.target.cpu_count", &got)

				assert.Error(t, err)
			})
		}
		t.Run("distinguishes zero from null", func(t *testing.T) {
			tree := observations.Tree{"arm": observations.Tree{"fancy": observations.Tree{
				"target": observations.Tree{"cpu_count": 0, "missing": nil},
			}}}
			var count, missing *int64

			countErr := tree.Decode("arm.fancy.target.cpu_count", &count)
			missingErr := tree.Decode("arm.fancy.target.missing", &missing)

			require.NoError(t, countErr)
			require.NoError(t, missingErr)
			require.NotNil(t, count)
			assert.Zero(t, *count)
			assert.Nil(t, missing)
		})
	})
}
