package term_test

import (
	"bytes"
	"fmt"
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

func TestCommandOutputFinish(t *testing.T) {
	for _, message := range []string{"message", "message\n", "message\n\n", "message\n\n\n"} {
		t.Run(fmt.Sprintf("appends newline after %q", message), func(t *testing.T) {
			buf := bytes.NewBufferString(message)
			err := term.NewCommandOutput(buf).Finish()
			require.NoError(t, err)
			assert.Equal(t, message+"\n", buf.String())
		})
	}
}

func TestSectionPrinterWrite(t *testing.T) {
	t.Run("does not print header until first nonempty write", func(t *testing.T) {
		var buf bytes.Buffer
		sections := term.NewSectionPrinter(term.NewCommandOutput(&buf), "Build images")

		output, outputError := sections.SubprocessOutput()
		_, emptyWriteError := sections.Write(nil)
		beforeLog := buf.String()
		_, writeError := sections.Write([]byte("Built image\n"))

		require.NoError(t, outputError)
		assert.Same(t, sections, output)
		require.NoError(t, emptyWriteError)
		require.NoError(t, writeError)
		assert.Empty(t, beforeLog)
		assert.Equal(t, term.Header("Build images", false)+"\nBuilt image\n", buf.String())
	})

	t.Run("does not print header for a section with no logs", func(t *testing.T) {
		var buf bytes.Buffer
		commandOutput := term.NewCommandOutput(&buf)

		term.NewSectionPrinter(commandOutput, "Empty section")
		sections := term.NewSectionPrinter(commandOutput, "Build images")
		_, writeError := sections.Write([]byte("Built image\n"))

		require.NoError(t, writeError)
		assert.NotContains(t, buf.String(), "Empty section")
	})

	t.Run("omits leading newline before first header and keeps blank line before subsequent headers", func(t *testing.T) {
		var buf bytes.Buffer
		commandOutput := term.NewCommandOutput(&buf)
		buildOutput := term.NewSectionPrinter(commandOutput, "Build images")
		pullOutput := term.NewSectionPrinter(commandOutput, "Pull images")

		_, bodyError := buildOutput.Write([]byte("Built image\n"))
		_, secondBodyError := pullOutput.Write([]byte("Pulled image\n"))

		require.NoError(t, bodyError)
		require.NoError(t, secondBodyError)
		assert.Equal(t, term.Header("Build images", false)+"\nBuilt image\n\n"+
			term.Header("Pull images", false)+"\nPulled image\n", buf.String())
	})
}

func TestHeader(t *testing.T) {
	t.Run("renders header with padding", func(t *testing.T) {
		header := term.Header("Hello", false)

		const totalWidth = 60
		prefix := "┌─ "
		suffix := " "
		barWidth := totalWidth - len(prefix) - len("Hello") - len(suffix)
		expected := prefix + "Hello" + suffix + strings.Repeat("─", barWidth)

		assert.Equal(t, expected, header)
	})

	t.Run("renders without padding when description is too long", func(t *testing.T) {
		description := strings.Repeat("x", 80)

		header := term.Header(description, false)

		expected := "┌─ " + description + " "
		assert.Equal(t, expected, header)
	})

	t.Run("dims borders for terminal output", func(t *testing.T) {
		header := term.Header("Hello", true)

		assert.Contains(t, header, term.Color(term.Dim, "┌─ "))
		barWidth := 60 - len("┌─ ") - len("Hello") - len(" ")
		assert.Contains(t, header, term.Color(term.Dim, " "+strings.Repeat("─", barWidth)))
	})
}
