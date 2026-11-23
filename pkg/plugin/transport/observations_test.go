package transport

import (
	"encoding/json"
	"testing"

	"github.com/arm/topo/pkg/plugin/observations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeObservations(t *testing.T) {
	t.Run("preserves producer values through host collection", func(t *testing.T) {
		produced := observations.Tree{
			"cpu":      observations.Tree{"count": 8},
			"memory":   observations.Tree{"bytes": int64(9007199254740993)},
			"features": []string{"avx2", "aes"},
			"os": struct {
				Name string `json:"name"`
			}{Name: "linux"},
			"mixed": []any{true, "text", 3.5, nil},
			"":      "empty key",
		}
		wire, err := json.Marshal(produced)
		require.NoError(t, err)

		decoded, err := decodeObservations(wire)

		require.NoError(t, err)
		assert.Equal(t, observations.Tree{
			"cpu":      map[string]any{"count": json.Number("8")},
			"memory":   map[string]any{"bytes": json.Number("9007199254740993")},
			"features": []any{"avx2", "aes"},
			"os":       map[string]any{"name": "linux"},
			"mixed":    []any{true, "text", json.Number("3.5"), nil},
			"":         "empty key",
		}, decoded)
		collected, err := json.Marshal(decoded)
		require.NoError(t, err)
		assert.JSONEq(t, string(wire), string(collected))
	})
	t.Run("rejects invalid JSON", func(t *testing.T) {
		_, err := decodeObservations([]byte(`{`))

		assert.ErrorContains(t, err, "decoding observations")
	})
}
