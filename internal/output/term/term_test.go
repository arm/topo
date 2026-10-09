package term_test

import (
	"os"
	"testing"

	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTTY(t *testing.T) {
	t.Run("returns false for a pipe", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, r.Close())
		}()
		defer func() {
			require.NoError(t, w.Close())
		}()

		assert.False(t, term.IsTerminal(w))
	})

	t.Run("stdout returns a boolean", func(t *testing.T) {
		got := term.IsTerminal(os.Stdout)

		assert.IsType(t, true, got)
	})

	t.Run("null device returns false", func(t *testing.T) {
		file, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, file.Close()) })

		got := term.IsTerminal(file)

		assert.False(t, got)
	})
}

func TestWrapText(t *testing.T) {
	t.Run("returns input when maxWidth is zero", func(t *testing.T) {
		out := term.WrapText("hello world", 0, 0)

		assert.Equal(t, "hello world", out)
	})

	t.Run("wraps text to max width", func(t *testing.T) {
		out := term.WrapText("hello world here", 11, 0)

		assert.Equal(t, "hello world\nhere", out)
	})

	t.Run("applies indentation", func(t *testing.T) {
		out := term.WrapText("hello world", 20, 2)

		assert.Equal(t, "  hello world", out)
	})

	t.Run("wraps with indentation", func(t *testing.T) {
		out := term.WrapText("hello world here", 12, 2)

		assert.Equal(t, "  hello\n  world here", out)
	})

	t.Run("handles multiple paragraphs", func(t *testing.T) {
		in := "one two three\n\nfour five"
		out := term.WrapText(in, 10, 0)

		assert.Equal(t, "one two\nthree\n\nfour five", out)
	})

	t.Run("negative indent treated as zero", func(t *testing.T) {
		out := term.WrapText("hello world", 20, -5)

		assert.Equal(t, "hello world", out)
	})

	t.Run("preserves explicit newlines", func(t *testing.T) {
		in := "hello\nworld here"
		out := term.WrapText(in, 10, 0)

		assert.Equal(t, "hello\nworld here", out)
	})
}
