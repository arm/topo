//go:build linux || darwin

package term_test

import (
	"context"
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

	t.Run("accepts a prefilled required value without editing", func(t *testing.T) {
		terminal := newTestPTY(t, "\r\x03")
		prompt := term.Prompt{Initial: "hello", Validate: func(input string) bool { return input != "" }}

		got, err := term.ReadPrompt(terminal, terminal, prompt)

		require.NoError(t, err)
		assert.Equal(t, "hello", got)
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

	t.Run("Ctrl+R replaces the input without submitting", func(t *testing.T) {
		initial := "sour"
		restoreValue := "sweet"
		ctrlR := "\x12"
		inputAfterCtrlR := "yellow"
		enter := "\r"
		terminal := newTestPTY(t, initial+ctrlR+inputAfterCtrlR+enter)
		prompt := term.Prompt{RestoreValue: new(restoreValue)}

		got, err := term.ReadPrompt(terminal, terminal, prompt)

		require.NoError(t, err)
		assert.Equal(t, restoreValue+inputAfterCtrlR, got)
	})

	t.Run("keeps editing available and cancels an action on manual submission", func(t *testing.T) {
		terminal := newTestPTY(t, "\x0fmanual\r")
		stopped := make(chan struct{})
		prompt := term.Prompt{Actions: []term.PromptAction{{Key: 15, Run: func(ctx context.Context, _ string) (string, error) {
			defer close(stopped)
			<-ctx.Done()
			return "", ctx.Err()
		}}}}

		got, err := term.ReadPrompt(terminal, terminal, prompt)

		require.NoError(t, err)
		assert.Equal(t, "manual", got)
		select {
		case <-stopped:
		default:
			t.Fatal("action was not stopped before returning")
		}
	})

	t.Run("fills an action result for confirmation", func(t *testing.T) {
		terminal, master := newTestPTYWithMaster(t, "\x0f")
		painted := make(chan struct{}, 1)
		prompt := term.Prompt{
			Actions: []term.PromptAction{{Key: 15, Run: func(context.Context, string) (string, error) {
				return "chosen", nil
			}}},
			Validate: func(value string) bool { return value == "chosen" },
			Hint: func(value string, _ bool) string {
				if value == "chosen" {
					select {
					case painted <- struct{}{}:
					default:
					}
				}
				return "Confirm"
			},
		}
		typed := make(chan error, 1)
		go func() {
			<-painted
			_, err := io.WriteString(master, "\r")
			typed <- err
		}()

		got, err := term.ReadPrompt(terminal, terminal, prompt)

		require.NoError(t, err)
		assert.Equal(t, "chosen", got)
		assert.NoError(t, <-typed)
	})

	t.Run("keeps manual entry available after an action fails", func(t *testing.T) {
		terminal, master := newTestPTYWithMaster(t, "\x0f")
		failed := make(chan struct{})
		prompt := term.Prompt{Actions: []term.PromptAction{{Key: 15, Run: func(context.Context, string) (string, error) {
			defer close(failed)
			return "", errors.New("browser unavailable")
		}}}}
		typed := make(chan error, 1)
		go func() {
			<-failed
			_, err := io.WriteString(master, "manual\r")
			typed <- err
		}()

		got, err := term.ReadPrompt(terminal, terminal, prompt)

		require.NoError(t, err)
		assert.Equal(t, "manual", got)
		assert.NoError(t, <-typed)
	})

	t.Run("cancels an action on interrupt", func(t *testing.T) {
		terminal := newTestPTY(t, "\x0f\x03")
		stopped := make(chan struct{})
		prompt := term.Prompt{Actions: []term.PromptAction{{Key: 15, Run: func(ctx context.Context, _ string) (string, error) {
			defer close(stopped)
			<-ctx.Done()
			return "", ctx.Err()
		}}}}

		_, err := term.ReadPrompt(terminal, terminal, prompt)

		assert.ErrorIs(t, err, readline.ErrInterrupt)
		select {
		case <-stopped:
		default:
			t.Fatal("action was not cancelled")
		}
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
	terminal, _ := newTestPTYWithMaster(t, keys)
	return terminal
}

func newTestPTYWithMaster(t *testing.T, keys string) (*os.File, *os.File) {
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
	return testTty, testPty
}
