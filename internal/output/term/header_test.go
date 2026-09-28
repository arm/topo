package term_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgress(t *testing.T) {
	t.Run("Header", func(t *testing.T) {
		t.Run("does not prefix the first header", func(t *testing.T) {
			var buffer bytes.Buffer
			progress := term.NewProgress(&buffer)

			err := progress.Header("Hello")

			require.NoError(t, err)
			assert.False(t, strings.HasPrefix(buffer.String(), "\n"))
		})

		t.Run("prefixes subsequent headers with a newline", func(t *testing.T) {
			var buffer bytes.Buffer
			progress := term.NewProgress(&buffer)

			require.NoError(t, progress.Header("First"))
			err := progress.Header("Second")

			require.NoError(t, err)
			assert.Contains(t, buffer.String(), "\n\n── Second")
		})

	})
}

func TestHeader(t *testing.T) {
	t.Run("renders header with padding", func(t *testing.T) {
		got := term.Header("Hello", term.NewPalette(false))

		want := "── Hello " + strings.Repeat("─", 51)
		//               ^
		//       123456789 => 60 - 9 = 51
		assert.Equal(t, want, got)
	})

	t.Run("correctly pads around unicode symbols", func(t *testing.T) {
		got := term.Header("✓✓✓", term.NewPalette(false))

		want := "── ✓✓✓ " + strings.Repeat("─", 53)
		//             ^
		//       1234567 => 60 - 7 = 53
		assert.Equal(t, want, got)
	})

	t.Run("renders without padding when description is too long", func(t *testing.T) {
		description := strings.Repeat("x", 80)

		got := term.Header(description, term.NewPalette(false))

		want := "── " + description
		assert.Equal(t, want, got)
	})

	t.Run("dims borders for terminal output", func(t *testing.T) {
		palette := term.NewPalette(true)

		got := term.Header("Hello", palette)

		want := palette.Color(term.Dim, "── ") +
			"Hello" +
			palette.Color(term.Dim, " ───────────────────────────────────────────────────")
		assert.Equal(t, want, got)
	})
}
