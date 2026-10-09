package term

import (
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
		Stdout:       outputFile,
		Stderr:       outputFile,
		HistoryLimit: -1,
		FuncMakeRaw: func() error {
			var err error
			inputOriginalState, err = readline.MakeRaw(inputFd)
			return err
		},
		FuncExitRaw: func() error {
			return readline.Restore(inputFd, inputOriginalState)
		},
	})
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, editor.Close()) }()

	var currentContent []string
	var currentInput []rune
	var drawErr error

	editor.Config.SetListener(func(newInput []rune, _ int, key rune) ([]rune, int, bool) {
		if key == readline.CharCtrlJ || key == readline.CharInterrupt || key == readline.CharEnter || (key == readline.CharDelete && len(currentInput) == 0) {
			return nil, 0, false
		}

		if key == 0 {
			// Readline's startup callback omits the prefilled input.
			newInput = []rune(prompt.Initial)
		}

		if key == readline.CharCtrlL {
			currentContent = nil
		}

		currentInput = newInput

		currentContent, drawErr = render(editor, outputFile, prompt, newInput, currentContent)
		return nil, 0, false
	})

	editor.Config.FuncFilterInputRune = func(key rune) (rune, bool) {
		if (key == readline.CharEnter || key == readline.CharCtrlJ) && prompt.Validate != nil {
			return key, prompt.Validate(string(currentInput))
		}

		return key, true
	}

	value, err := editor.ReadlineWithDefault(prompt.Initial)
	return value, errors.Join(err, drawErr)
}

const (
	carriageReturn = "\r"
	lineFeed       = "\n"
	clearToEnd     = "\x1b[J"
)

func moveUp(rows int) string {
	return fmt.Sprintf("\x1b[%dA", rows)
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

func render(editor *readline.Instance, outputFile *os.File, prompt Prompt, currentInput []rune, previousLines []string) ([]string, error) {
	editor.Clean()
	defer editor.Refresh()

	outputFd := int(outputFile.Fd()) // #nosec G115 - posix fds and Windows handles fit into an int
	width, height, sizeErr := readline.GetSize(outputFd)
	if sizeErr != nil || width < 1 || height < 1 {
		editor.Operation.Close()
		return nil, fmt.Errorf("terminal width unavailable: %d (%v)", width, sizeErr)
	}

	var update strings.Builder
	if previousRows := textHeight(previousLines, width); previousRows > 0 {
		fmt.Fprint(&update, moveUp(previousRows))
	}
	update.WriteString(clearToEnd)

	rows := prompt.Content(string(currentInput))

	for _, row := range rows {
		update.WriteString(row)
		update.WriteString(carriageReturn)
		update.WriteString(lineFeed)
	}

	_, err := io.WriteString(outputFile, update.String())
	if err != nil {
		editor.Operation.Close()
	}
	return rows, err
}
