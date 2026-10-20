package term

import (
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

type Format int

const (
	// PlainFormat renders human-readable plain text
	Plain Format = iota
	// JSONFormat renders machine-readable JSON
	JSON
)

func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := int(f.Fd()) // #nosec G115 - posix fds and Windows handles fit into an int
	return readline.IsTerminal(fd)
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
