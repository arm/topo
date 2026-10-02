package term

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/clipperhouse/displaywidth"
	xterm "golang.org/x/term"
)

type Format int

const (
	// PlainFormat renders human-readable plain text
	Plain Format = iota
	// JSONFormat renders machine-readable JSON
	JSON
)

func IsTerminal(w io.Writer) bool {
	fd, ok := getFd(w)
	if !ok {
		return false
	}

	return xterm.IsTerminal(fd)
}

func Dimensions(w io.Writer) (width, height int, err error) {
	fd, ok := getFd(w)
	if !ok {
		return 0, 0, errors.New("not a terminal")
	}
	return xterm.GetSize(fd)
}

func WrapText(s string, maxWidth, indentSpaces int) string {
	if maxWidth <= 0 {
		return s
	}
	if indentSpaces < 0 {
		indentSpaces = 0
	}

	var out []string
	prefix := strings.Repeat(" ", indentSpaces)
	for para := range strings.SplitSeq(s, "\n\n") {
		for rawLine := range strings.SplitSeq(para, "\n") {
			line := prefix

			for word := range strings.FieldsSeq(rawLine) {
				space := 1
				if line == prefix {
					space = 0
				}

				if len(line)+space+len(word) > maxWidth {
					out = append(out, line)
					line = prefix + word
				} else {
					if line != prefix {
						line += " "
					}
					line += word
				}
			}

			if line != prefix {
				out = append(out, line)
			}
		}

		out = append(out, "")
	}
	if len(out) > 0 {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

func TextHeight(lines []string, width int) int {
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

func getFd(w io.Writer) (int, bool) {
	f, ok := w.(*os.File)
	if !ok {
		return 0, false
	}
	fd := int(f.Fd()) // #nosec G115 - posix fds and Windows handles fit into an int
	return fd, true
}
