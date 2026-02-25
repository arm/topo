package term_test

import (
	"bytes"
	"os"
	"strings"
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

		assert.False(t, term.IsTTY(w))
	})

	t.Run("stdout returns a boolean", func(t *testing.T) {
		got := term.IsTTY(os.Stdout)

		assert.IsType(t, true, got)
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

func TestPrintHeader(t *testing.T) {
	t.Run("renders header with padding", func(t *testing.T) {
		var buf bytes.Buffer

		require.NoError(t, term.PrintHeader(&buf, "Hello"))

		const totalWidth = 60
		prefix := "┌─ "
		suffix := " "
		barWidth := totalWidth - len(prefix) - len("Hello") - len(suffix)
		expected := "\n" + prefix + "Hello" + suffix + strings.Repeat("─", barWidth) + "\n"

		assert.Equal(t, expected, buf.String())
	})

	t.Run("renders without padding when description is too long", func(t *testing.T) {
		var buf bytes.Buffer
		description := strings.Repeat("x", 80)

		require.NoError(t, term.PrintHeader(&buf, description))

		expected := "\n┌─ " + description + " \n"
		assert.Equal(t, expected, buf.String())
	})
}
