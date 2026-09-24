package term_test

import (
	"testing"

	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
)

func TestPalette(t *testing.T) {
	t.Run("applies ANSI colors when enabled", func(t *testing.T) {
		palette := term.NewPalette(true)

		got := palette.Color(term.Green, "text")

		assert.Equal(t, "\x1b[32mtext\x1b[0m", got)
	})

	t.Run("returns text unchanged when disabled", func(t *testing.T) {
		palette := term.NewPalette(false)

		got := palette.Color(term.Green, "text")

		assert.Equal(t, "text", got)
	})

	t.Run("does not style empty text", func(t *testing.T) {
		palette := term.NewPalette(true)

		got := palette.Color(term.Green, "")

		assert.Empty(t, got)
	})
}
