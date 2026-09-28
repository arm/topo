package term

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type Progress struct {
	output  io.Writer
	started bool
}

func NewProgress(output io.Writer) *Progress {
	return &Progress{output: output}
}

func (p *Progress) Header(description string) error {
	prefix := ""
	if p.started {
		prefix = "\n"
	}

	if err := printHeader(p.output, description, prefix); err != nil {
		return err
	}
	p.started = true
	return nil
}

func (p *Progress) Output() io.Writer {
	return p.output
}

func printHeader(w io.Writer, description string, prefix string) error {
	header := Header(description, NewPaletteFor(w))
	if header == "" {
		return nil
	}

	_, err := fmt.Fprintf(w, "%s%s\n", prefix, header)
	return err
}

func Header(description string, palette Palette) string {
	if description == "" {
		return ""
	}

	const totalWidth = 60
	leadingBar := "── "
	barSeparator := " "

	titleWidth := utf8.RuneCountInString(description)
	headerContentWidth := utf8.RuneCountInString(leadingBar) + titleWidth + utf8.RuneCountInString(barSeparator)
	trailingBarWidth := max(totalWidth-headerContentWidth, 0)
	trailingBar := ""
	if trailingBarWidth > 0 {
		trailingBar = barSeparator + strings.Repeat("─", trailingBarWidth)
	}
	return palette.Color(Dim, leadingBar) + description + palette.Color(Dim, trailingBar)
}
