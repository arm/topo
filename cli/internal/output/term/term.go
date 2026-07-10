package term

import (
	"io"
	"os"
	"strings"
)

type Format int

const (
	// PlainFormat renders human-readable plain text
	Plain Format = iota
	// JSONFormat renders machine-readable JSON
	JSON
)

const (
	Reset  = "\033[0m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Red    = "\033[31m"
)

func Color(col, str string) string {
	return col + str + Reset
}

func IsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
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
