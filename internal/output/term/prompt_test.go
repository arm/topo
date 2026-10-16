//go:build linux || darwin

package term_test

import (
	"errors"
	"io"
	"os"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/arm/topo/internal/output/term"
	"github.com/chzyer/readline"
	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPrompt(t *testing.T) {
	t.Run("edits a prefilled value", func(t *testing.T) {
		terminal := newTestPTY(t, "!\r")
		prompt := term.Prompt{Initial: "hello"}

		got, err := term.ReadPrompt(terminal, terminal, prompt)

		require.NoError(t, err)
		assert.Equal(t, "hello!", got)
	})

	t.Run("rejects invalid input before accepting a valid value", func(t *testing.T) {
		terminal := newTestPTY(t, "\rhello\r")
		var validated []string
		prompt := term.Prompt{Content: []string{"Value:"}, Validate: func(input string) bool {
			validated = append(validated, input)
			return input != ""
		}}

		got, err := term.ReadPrompt(terminal, terminal, prompt)

		require.NoError(t, err)
		assert.Equal(t, "hello", got)
		assert.Equal(t, []string{"", "hello"}, validated)
	})

	t.Run("shows errors only after invalid Enter and clears them on cursor movement", func(t *testing.T) {
		terminal := newTestPTY(t, "\r\x1b[Dhello\r")
		var errorsShown []bool
		prompt := term.Prompt{
			Validate: func(input string) bool { return input != "" },
			Hint: func(input string, invalid bool) string {
				if input == "" {
					errorsShown = append(errorsShown, invalid)
				}
				return "   Hint"
			},
		}

		got, err := term.ReadPrompt(terminal, terminal, prompt)

		require.NoError(t, err)
		assert.Equal(t, "hello", got)
		assert.Equal(t, []bool{false, true, false}, slices.Compact(errorsShown))
	})

	t.Run("returns an interrupt on cancellation", func(t *testing.T) {
		terminal := newTestPTY(t, "\x03")
		prompt := term.Prompt{Content: []string{"Value:"}}

		_, err := term.ReadPrompt(terminal, terminal, prompt)

		assert.ErrorIs(t, err, readline.ErrInterrupt)
	})
}

func newTestPTY(t *testing.T, keys string) *os.File {
	t.Helper()
	testPty, testTty, err := pty.Open()
	require.NoError(t, err)
	done := make(chan error, 1)
	closeFiles := func() {
		for _, file := range []*os.File{testTty, testPty} {
			if err := file.Close(); !errors.Is(err, os.ErrClosed) {
				assert.NoError(t, err)
			}
		}
	}
	timeout := time.AfterFunc(5*time.Second, func() {
		t.Error("timed out waiting for prompt")
		closeFiles()
	})
	t.Cleanup(func() {
		timeout.Stop()
		closeFiles()
		assert.NoError(t, <-done)
	})
	// Queue keys without canonical buffering or Ctrl+C generating a signal.
	_, err = readline.MakeRaw(int(testTty.Fd()))
	require.NoError(t, err)
	go func() {
		if _, err := io.WriteString(testPty, keys); err != nil {
			done <- err
			return
		}
		_, err := io.Copy(io.Discard, testPty)
		if errors.Is(err, syscall.EIO) || errors.Is(err, os.ErrClosed) {
			err = nil
		}
		done <- err
	}()
	require.NoError(t, pty.Setsize(testTty, &pty.Winsize{Rows: 24, Cols: 80}))
	return testTty
}
