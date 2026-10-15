package term

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/clipperhouse/displaywidth"
)

type Prompt struct {
	Prefix   string
	Initial  string
	Validate func(input string) bool
	Content  func(input string) []string
}

func ReadPrompt(
	inputFile, outputFile *os.File,
	prompt Prompt,
) (_ string, err error) {
	if !IsTerminal(inputFile) || !IsTerminal(outputFile) {
		panic("internal error: ReadPrompt not running in an interactive terminal")
	}

	inputFd := int(inputFile.Fd()) // #nosec G115 - posix fds and Windows handles fit into an int
	var inputOriginalState *readline.State

	editor, err := readline.NewEx(&readline.Config{
		Prompt:       prompt.Prefix,
		Stdin:        inputFile,
		Stdout:       promptOutput{outputFile},
		Stderr:       promptOutput{outputFile},
		HistoryLimit: -1,
		FuncMakeRaw: func() error {
			var err error
			inputOriginalState, err = readline.MakeRaw(inputFd)
			return err
		},
		FuncExitRaw: func() error {
			if inputOriginalState == nil {
				return nil
			}
			return readline.Restore(inputFd, inputOriginalState)
		},
	})
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, editor.Close()) }()

	var currentContent []string
	var currentInput string
	var drawErr error

	editor.Config.SetListener(func(newInput []rune, _ int, key rune) ([]rune, int, bool) {
		if key == readline.CharCtrlJ || key == readline.CharInterrupt || key == readline.CharEnter || (key == readline.CharDelete && len(currentInput) == 0) {
			return nil, 0, false
		}

		currentInput = string(newInput)
		if key == 0 {
			// Readline's startup callback omits the prefilled input.
			currentInput = prompt.Initial
		}

		if key == readline.CharCtrlL {
			currentContent = nil
		}

		content := prompt.Content(currentInput)
		drawErr = errors.Join(drawErr, render(editor, outputFile, content, currentContent))
		currentContent = content

		_, writeErr := io.WriteString(outputFile, endSynchronizedUpdate)
		drawErr = errors.Join(drawErr, writeErr)
		return nil, 0, false
	})

	editor.Config.FuncFilterInputRune = func(key rune) (rune, bool) {
		if (key == readline.CharEnter || key == readline.CharCtrlJ) && prompt.Validate != nil {
			return key, prompt.Validate(currentInput)
		}

		// Readline clears and repaints before calling the listener. Group both
		// renderers into one frame, excluding keys that can bypass the listener.
		if editor.Operation.IsNormalMode() && canSynchronizeKey(key) {
			_, writeErr := io.WriteString(outputFile, beginSynchronizedUpdate)
			if writeErr != nil {
				drawErr = errors.Join(drawErr, writeErr)
				editor.Operation.Close()
				return key, false
			}
		}
		return key, true
	}

	value, err := editor.ReadlineWithDefault(prompt.Initial)
	return value, errors.Join(err, drawErr)
}

const (
	lineBreak      = "\r\n"
	clearToEnd     = "\x1b[J"
	cursorUpOneRow = "\x1b[A"
	cursorUpRows   = "\x1b[%dA"
	backspace      = "\b"
	// DEC synchronized output presents a complete edit as one frame.
	beginSynchronizedUpdate = "\x1b[?2026h"
	endSynchronizedUpdate   = "\x1b[?2026l"
)

func canSynchronizeKey(key rune) bool {
	return key != 0 && key != readline.CharInterrupt &&
		key != readline.CharCtrlZ && key != readline.CharTab && key != readline.CharEnter &&
		key != readline.CharCtrlJ && key != readline.CharDelete
}

func textHeight(lines []string, width int) int {
	measure := displaywidth.Options{ControlSequences: true}

	rows := 0
	for _, line := range lines {
		lineRows, lineColumn := 1, 0
		graphemes := measure.StringGraphemes(line)

		for graphemes.Next() {
			cells := graphemes.Width()
			if cells == 0 {
				continue
			}

			if lineColumn+cells > width {
				lineRows++
				lineColumn = 0
			}
			lineColumn += cells
		}
		rows += lineRows
	}
	return rows
}

func render(editor *readline.Instance, outputFile *os.File, rows, previousLines []string) error {
	outputFd := int(outputFile.Fd()) // #nosec G115 - posix fds and Windows handles fit into an int
	width, height, sizeErr := readline.GetSize(outputFd)
	if sizeErr != nil || width < 1 || height < 1 {
		editor.Operation.Close()
		return fmt.Errorf("terminal width unavailable: %d (%v)", width, sizeErr)
	}

	editor.Clean()
	defer editor.Refresh()

	var update strings.Builder
	if previousRows := textHeight(previousLines, width); previousRows > 0 {
		fmt.Fprintf(&update, cursorUpRows, previousRows)
	}
	update.WriteString(clearToEnd)

	for _, row := range rows {
		update.WriteString(row)
		update.WriteString(lineBreak)
	}

	_, err := io.WriteString(outputFile, update.String())
	if err != nil {
		editor.Operation.Close()
	}
	return err
}

// Readline backspaces before moving up across a wrapped row. With reverse
// wrapping, that backspace already moves up, so remove the redundant step.
type promptOutput struct{ file *os.File }

func (w promptOutput) Write(data []byte) (int, error) {
	_, err := w.file.Write(bytes.ReplaceAll(data, []byte(backspace+cursorUpOneRow), []byte(cursorUpOneRow)))
	if err != nil {
		return 0, err
	}
	return len(data), nil
}
