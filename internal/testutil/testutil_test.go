package testutil_test

import (
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMergeJSONValues(t *testing.T) {
	tests := []struct {
		name     string
		base     any
		override any
		want     any
	}{
		{
			name: "merges nested maps",
			base: map[string]any{
				"name": "service",
				"metadata": map[string]any{
					"version": "1.0",
					"status":  "old",
				},
			},
			override: map[string]any{
				"metadata": map[string]any{
					"status": "new",
				},
			},
			want: map[string]any{
				"name": "service",
				"metadata": map[string]any{
					"version": "1.0",
					"status":  "new",
				},
			},
		},
		{
			name: "merges nested arrays by index",
			base: []any{
				map[string]any{
					"name": "first",
					"checks": []any{
						map[string]any{"status": "old", "name": "nested"},
					},
				},
				map[string]any{"name": "second"},
			},
			override: []any{
				map[string]any{
					"checks": []any{
						map[string]any{"status": "new"},
					},
				},
			},
			want: []any{
				map[string]any{
					"name": "first",
					"checks": []any{
						map[string]any{"status": "new", "name": "nested"},
					},
				},
				map[string]any{"name": "second"},
			},
		},
		{
			name: "appends extra array overrides",
			base: []any{"first"},
			override: []any{
				nil,
				"second",
			},
			want: []any{nil, "second"},
		},
		{
			name:     "replaces mismatched values",
			base:     map[string]any{"name": "old"},
			override: "new",
			want:     "new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.MergeJSONValues(tt.base, tt.override)

			assert.Equal(t, tt.want, got)
		})
	}
}
