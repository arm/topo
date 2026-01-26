package output

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"
)

// Output format for the commands
type Format int

const (
	// PlainFormat renders human-readable plain text
	PlainFormat Format = iota
	// JSONFormat renders machine-readable JSON
	JSONFormat
)

type printable interface {
	AsJSON() (string, error)
	AsPlain() (string, error)
}

type Printer struct {
	Target io.Writer
	format Format
}

func NewPrinter(target io.Writer, format Format) *Printer {
	return &Printer{Target: target, format: format}
}

func (p *Printer) Print(printable printable) error {
	if p.format == JSONFormat {
		jsonStr, err := printable.AsJSON()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(p.Target, jsonStr)
		return err
	}

	plainStr, err := printable.AsPlain()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(p.Target, plainStr)
	return err
}

const (
	reset  = "\033[0m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
)

var currentTemplate = template.Must(
	template.New("empty").Parse(""),
)

func getTemplate(printer *Printer, name string, tplText string) *template.Template {
	funcs := template.FuncMap{
		"join":   strings.Join,
		"wrap":   func(s string) string { return WrapText(s, 80, 2) },
		"cyan":   func(s string) string { return s },
		"blue":   func(s string) string { return s },
		"yellow": func(s string) string { return s },
	}

	if isTTY(printer.Target) {
		funcs["cyan"] = func(s string) string { return colour(cyan, s) }
		funcs["blue"] = func(s string) string { return colour(blue, s) }
		funcs["yellow"] = func(s string) string { return colour(yellow, s) }
	}

	return template.Must(
		template.New(name).
			Funcs(funcs).
			Parse(tplText),
	)
}

func isTTY(w io.Writer) bool {
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

func colour(col, str string) string {
	return col + str + reset
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
