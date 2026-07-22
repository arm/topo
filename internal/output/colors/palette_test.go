package colors_test

import (
	"testing"

	"github.com/arm/topo/internal/output/colors"
	"github.com/stretchr/testify/assert"
)

func TestPalette(t *testing.T) {
	t.Run("applies normal ANSI styles for semantic roles", func(t *testing.T) {
		palette := colors.NewPalette(true)
		tests := []struct {
			name string
			role colors.Role
			want string
		}{
			{name: "accent", role: colors.Accent, want: "\x1b[36mtext\x1b[0m"},
			{name: "success", role: colors.Success, want: "\x1b[32mtext\x1b[0m"},
			{name: "warning", role: colors.Warning, want: "\x1b[33mtext\x1b[0m"},
			{name: "failure", role: colors.Failure, want: "\x1b[31mtext\x1b[0m"},
			{name: "information", role: colors.Information, want: "\x1b[34mtext\x1b[0m"},
			{name: "muted", role: colors.Muted, want: "\x1b[2mtext\x1b[0m"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				got := palette.Apply(test.role, "text")

				assert.Equal(t, test.want, got)
			})
		}
	})

	t.Run("returns text unchanged when disabled", func(t *testing.T) {
		palette := colors.NewPalette(false)

		got := palette.Apply(colors.Success, "text")

		assert.Equal(t, "text", got)
	})

	t.Run("is disabled by default", func(t *testing.T) {
		var palette colors.Palette

		got := palette.Apply(colors.Success, "text")

		assert.Equal(t, "text", got)
	})

	t.Run("does not style empty text", func(t *testing.T) {
		palette := colors.NewPalette(true)

		got := palette.Apply(colors.Success, "")

		assert.Empty(t, got)
	})
}
