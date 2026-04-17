package target_test

import (
	"testing"

	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindKeyValueInString(t *testing.T) {
	t.Run("finds key and parses value", func(t *testing.T) {
		text := `MemTotal:       16384000 kB
MemFree:        8192000 kB`

		got, err := target.FindKeyValueInString("MemTotal", text)

		require.NoError(t, err)
		assert.Equal(t, int64(16384000), got)
	})

	t.Run("returns error when key not found", func(t *testing.T) {
		text := `MemTotal:       16384000 kB`

		got, err := target.FindKeyValueInString("MissingKey", text)

		assert.Error(t, err)
		assert.Equal(t, int64(0), got)
	})

	t.Run("returns error when value is invalid", func(t *testing.T) {
		text := `MemTotal:       notanumber`

		_, err := target.FindKeyValueInString("MemTotal", text)

		assert.Error(t, err)
	})
}
