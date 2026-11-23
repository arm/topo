package term

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/chzyer/readline"
)

// PromptAction runs without blocking editing. Its result replaces the editable value.
type PromptAction struct {
	Key rune
	Run func(context.Context, string) (string, error)
}

type Prompt struct {
	Prefix       string
	Initial      string
	Validate     func(input string) bool
	Content      []string
	Hint         func(input string, showValidationError bool) string
	RestoreValue *string
	Actions      []PromptAction
}

func ReadPrompt(
	inputFile, outputFile *os.File,
	prompt Prompt,
) (_ string, err error) {
	if !IsTerminal(inputFile) || !IsTerminal(outputFile) {
		panic("internal error: ReadPrompt not running in an interactive terminal")
	}

	inputFd := int(inputFile.Fd())   // #nosec G115 - posix fds and Windows handles fit into an int
	outputFd := int(outputFile.Fd()) // #nosec G115
	var inputOriginalState *readline.State
	output := &promptOutput{writer: outputFile}

	editor, err := readline.NewEx(&readline.Config{
		Prompt:              prompt.Prefix,
		Stdin:               inputFile,
		Stdout:              output,
		Stderr:              output,
		HistoryLimit:        -1,
		ForceUseInteractive: true,
		FuncGetWidth: func() int {
			width, _, err := readline.GetSize(outputFd)
			if err != nil {
				return 0
			}
			return width
		},
		FuncMakeRaw: func() error {
			var err error
			inputOriginalState, err = readline.MakeRaw(inputFd)
			return err
		},
		FuncExitRaw: func() error {
			if inputOriginalState == nil {
				return nil
			}
			err := readline.Restore(inputFd, inputOriginalState)
			if syscallErr, ok := errors.AsType[syscall.Errno](err); ok && syscallErr == 0 {
				err = nil
			}
			return err
		},
	})
	if err != nil {
		return "", err
	}
	defer func() {
		err = errors.Join(err, output.Close(), editor.Close())
	}()

	var showValidationError atomic.Bool
	var writeErr error
	var inputMu sync.Mutex
	var currentInput atomic.Value
	currentInput.Store(prompt.Initial)
	actions := newPromptActions(editor, prompt.Actions, &inputMu)
	defer actions.close()

	editor.Config.SetPainter(promptPainter{
		prompt, editor.Config.FuncGetWidth, &showValidationError, actions.status, &currentInput,
	})
	editor.Config.FuncFilterInputRune = func(key rune) (rune, bool) {
		showValidationError.Store(false)
		inputMu.Lock()
		defer inputMu.Unlock()
		if key == readline.CharBckSearch && prompt.RestoreValue != nil {
			currentInput.Store(*prompt.RestoreValue)
			editor.Operation.SetBuffer(*prompt.RestoreValue)
			return readline.CharBell, true // Let readline advance without entering reverse search.
		}
		if actions.invoke(key, currentInput.Load().(string)) {
			editor.Refresh()
			return readline.CharBell, true
		}
		if (key == readline.CharEnter || key == readline.CharCtrlJ) && prompt.Validate != nil {
			showValidationError.Store(!prompt.Validate(currentInput.Load().(string)))
		}
		if !showValidationError.Load() && (key == readline.CharEnter || key == readline.CharCtrlJ || key == readline.CharInterrupt) {
			actions.cancel()
		}
		return key, !showValidationError.Load()
	}

	for row := range prompt.Content {
		if _, writeErr = fmt.Fprint(outputFile, prompt.Content[row]+lineBreak); writeErr != nil {
			return "", writeErr
		}
	}
	value, err := editor.ReadlineWithDefault(prompt.Initial)
	return value, errors.Join(err, writeErr)
}

const (
	clearToEnd              = "\x1b[J"
	beginSynchronizedUpdate = "\x1b[?2026h"
	endSynchronizedUpdate   = "\x1b[?2026l"
	carriageReturn          = "\r"
	lineFeed                = "\n"
	lineBreak               = carriageReturn + lineFeed
	cursorUpOneRow          = "\x1b[A"
	cursorUpRows            = "\x1b[%dA"
	backspace               = "\b"
	disableAutowrap         = "\x1b[?7l"
	enableAutowrap          = "\x1b[?7h"
	cursorForwardColumns    = "\x1b[%dC"
)

type promptOutput struct {
	writer io.Writer
	mu     sync.Mutex
	closed bool
}

func (w *promptOutput) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return len(data), nil
	}
	// Readline backspaces before moving up across a wrapped row. With reverse wrapping, that backspace already moves up, so remove the redundant step.
	frame := bytes.ReplaceAll(data, []byte(backspace+cursorUpOneRow), []byte(cursorUpOneRow))
	// Readline writes its clear and repaint separately; display them together.
	if bytes.HasPrefix(frame, []byte(clearToEnd)) {
		frame = append([]byte(beginSynchronizedUpdate), frame...)
	} else {
		frame = append(frame, endSynchronizedUpdate...)
	}
	_, err := w.writer.Write(frame)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (w *promptOutput) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	_, err := fmt.Fprint(w.writer, endSynchronizedUpdate)
	return err
}

type promptPainter struct {
	prompt              Prompt
	width               func() int
	showValidationError *atomic.Bool
	status              func() string
	input               *atomic.Value
}

func (p promptPainter) Paint(line []rune, _ int) []rune {
	p.input.Store(string(line))
	width := p.width()
	if p.prompt.Hint == nil || width < 1 || strings.HasSuffix(string(line), lineFeed) {
		return line
	}
	hint := p.prompt.Hint(string(line), p.showValidationError.Load())
	if status := p.status(); status != "" {
		hint += " · " + status
	}

	runes := readline.Runes{}
	cells := runes.WidthAll(runes.ColorFilter([]rune(p.prompt.Prefix))) + runes.WidthAll(line)
	column := cells % width

	var output strings.Builder
	output.WriteString(string(line))
	if cells > 0 && column == 0 {
		output.WriteString(lineBreak)
	}

	output.WriteString(lineBreak + lineBreak)
	output.WriteString(disableAutowrap)
	output.WriteString(hint)
	output.WriteString(" ")
	output.WriteString(carriageReturn)
	output.WriteString(enableAutowrap)

	fmt.Fprintf(&output, cursorUpRows, 2)
	if column > 0 {
		fmt.Fprintf(&output, cursorForwardColumns, column)
	}

	return []rune(output.String())
}
