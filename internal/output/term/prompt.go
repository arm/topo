package term

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/clipperhouse/displaywidth"
	goterm "golang.org/x/term"
)

type Prompt struct {
	Initial  string
	Validate func(input string) error
	Content  func(input string, validationErr error) []string
}

func ReadPrompt(
	input, output *os.File,
	prompt Prompt,
) (string, error) {
	if !IsTerminal(input) || !IsTerminal(output) {
		panic("internal error: ReadPrompt not running in an interactive terminal")
	}

	restore, err := makeRaw(input)
	if err != nil {
		return "", err
	}
	defer func() {
		err = errors.Join(err, restore())
	}()

	reader := bufio.NewReader(input)

	var lastContent []string
	currentInput := prompt.Initial
	var validationErr error

	// foreach keypress loop
	for {
		// re-measure terminal dimensions before nuking the previous content
		width, _, err := dimensions(output)
		if err != nil {
			return "", err
		}

		// clear last content
		if lastContentHeight := textHeight(lastContent, width); lastContentHeight > 1 {
			fmt.Fprint(output, moveUp(lastContentHeight-1))
		}
		fmt.Fprint(output, moveToRowStart+clearToEnd)

		// write new content
		content := prompt.Content(currentInput, validationErr)
		fmt.Fprint(output, strings.Join(content, newLine))
		lastContent = content

		// wait for keypress
		key, _, err := reader.ReadRune()
		if err != nil {
			return "", err
		}
		// TODO handle enter, backspace, and other control keys
		currentInput += string(key)
	}
}

const (
	ctrlA          = '\x01'
	ctrlC          = '\x03'
	ctrlE          = '\x05'
	backspace      = '\x08'
	lineFeed       = '\n'
	carriageReturn = '\r'
	ctrlU          = '\x15'
	escape         = '\x1b'
	deleteByte     = '\x7f'
)

const (
	moveToRowStart = string(carriageReturn)
	newLine        = string(carriageReturn) + string(lineFeed)
)

func moveUp(rows int) string {
	return fmt.Sprintf("%c[%dA", escape, rows)
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

func dimensions(f *os.File) (width, height int, err error) {
	fd := int(f.Fd()) // #nosec G115 - posix fds and Windows handles fit into an int
	return goterm.GetSize(fd)
}

func makeRaw(f *os.File) (restore func() error, err error) {
	fd := int(f.Fd()) // #nosec G115 - posix fds and Windows handles fit into an int
	oldState, err := goterm.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() error {
		return goterm.Restore(fd, oldState)
	}, nil
}

type keyKind uint8

const (
	keyIgnore keyKind = iota
	keyText
	keyEnter
	keyCancel
	keyBackspace
	keyDelete
	keyLeft
	keyRight
	keyHome
	keyEnd
	keyDeleteToStart
)

type key struct {
	kind    keyKind
	content rune
}

func readKey(reader *bufio.Reader) (key, error) {
	r, _, err := reader.ReadRune()
	if err != nil {
		return key{}, err
	}

	switch r {
	case carriageReturn, lineFeed:
		return key{kind: keyEnter}, nil
	case ctrlC:
		return key{kind: keyCancel}, nil
	case deleteByte, backspace:
		return key{kind: keyBackspace}, nil
	case ctrlA:
		return key{kind: keyHome}, nil
	case ctrlE:
		return key{kind: keyEnd}, nil
	case ctrlU:
		return key{kind: keyDeleteToStart}, nil
	case escape:
		return readEscapeSequence(reader)
	}

	if unicode.IsPrint(r) {
		return key{kind: keyText, content: r}, nil
	}
	return key{kind: keyIgnore}, nil
}

// Escape sequences
const (
	csi = string(escape) + "[" // Control Sequence Introducer
	ss3 = string(escape) + "O" // Single Shift 3

	clearToEnd   = csi + "J"
	moveUpFormat = csi + "%dA"

	leftSequence   = csi + "D"
	rightSequence  = csi + "C"
	homeSequence   = csi + "H"
	endSequence    = csi + "F"
	deleteSequence = csi + "3~"

	leftApplicationSequence  = ss3 + "D"
	rightApplicationSequence = ss3 + "C"
	homeApplicationSequence  = ss3 + "H"
	endApplicationSequence   = ss3 + "F"
)

func readEscapeSequence(reader *bufio.Reader) (key, error) {
	prefix, _, err := reader.ReadRune()
	if err != nil {
		return key{}, err
	}

	start := string(escape) + string(prefix)
	if start != csi && start != ss3 {
		return key{kind: keyIgnore}, nil
	}

	sequence := []byte{byte(escape), byte(prefix)}

	for len(sequence) < 32 {
		b, err := reader.ReadByte()
		if err != nil {
			return key{}, err
		}
		sequence = append(sequence, b)

		isFinalByte := b >= '@' && b <= '~'
		if !isFinalByte {
			continue
		}

		switch string(sequence) {
		case leftSequence, leftApplicationSequence:
			return key{kind: keyLeft}, nil
		case rightSequence, rightApplicationSequence:
			return key{kind: keyRight}, nil
		case homeSequence, homeApplicationSequence:
			return key{kind: keyHome}, nil
		case endSequence, endApplicationSequence:
			return key{kind: keyEnd}, nil
		case deleteSequence:
			return key{kind: keyDelete}, nil
		default:
			return key{kind: keyIgnore}, nil
		}
	}

	return key{}, fmt.Errorf("terminal key sequence too long")
}
