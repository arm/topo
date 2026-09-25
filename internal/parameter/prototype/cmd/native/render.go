package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter/prototype/internal/demo"
)

type frame struct {
	lines  []string
	row    int
	column int
}

func (e *editor) view(width, height int, palette term.Palette) frame {
	var view frame
	start, end := e.active, e.active+1
	if e.layout == "form" {
		visible := max(1, (height-6)/2)
		start = max(0, e.active-visible+1)
		end = min(len(e.parameters), start+visible)
	}
	for i := start; i < end; i++ {
		parameter, answer := e.parameters[i], e.answers[i]
		heading := clip(parameter.Heading(), width-1)
		if i != e.active {
			view.lines = append(view.lines, heading,
				palette.Color(term.Dim, clip("  "+demo.Preview(parameter, answer), width-1)))
			continue
		}
		view.lines = append(view.lines, palette.Color(term.Cyan, heading))
		view.row = len(view.lines)
		input, column := inputLine(parameter, answer, e.cursors[i], width, palette)
		view.column = column
		view.lines = append(view.lines, input)
	}
	view.lines = append(view.lines,
		clip(e.parameters[e.active].Description, width-1),
		clip("On Enter: "+demo.Preview(e.parameters[e.active], e.answers[e.active]), width-1),
		palette.Color(term.Dim, clip(demo.Help, width-1)),
		palette.Color(term.Dim, clip(demo.EditingHelp, width-1)))
	message := fmt.Sprintf("Field %d/%d | in memory only", e.active+1, len(e.parameters))
	color := term.Gray
	if e.message != "" {
		message, color = "! "+e.message, term.Red
	}
	view.lines = append(view.lines, palette.Color(color, clip(message, width-1)))
	return view
}

func inputLine(parameter demo.Parameter, answer demo.Answer, cursor, width int, palette term.Palette) (string, int) {
	if answer.Text == "" {
		return "> " + palette.Color(term.Dim, clip(parameter.Placeholder(answer), width-3)), 2
	}
	text := []rune(answer.Text)
	start := max(0, cursor-max(1, width-4))
	return "> " + clip(string(text[start:]), width-3), 2 + cursor - start
}

// This deliberate code-point approximation exposes the cost of omitting a Unicode width library.
func clip(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= max(0, width) {
		return text
	}
	if width < 2 {
		return ""
	}
	return string(runes[:width-1]) + "…"
}

type renderer struct {
	output io.Writer
	rows   int
	row    int
}

func (r *renderer) draw(view frame) error {
	var output strings.Builder
	output.WriteString("\x1b[?25l\r")
	if r.row > 0 {
		fmt.Fprintf(&output, "\x1b[%dA", r.row)
	}
	rows := max(r.rows, len(view.lines))
	for i := 0; i < rows; i++ {
		if i > 0 {
			output.WriteString("\r\n")
		}
		output.WriteString("\x1b[2K")
		if i < len(view.lines) {
			output.WriteString(view.lines[i])
		}
	}
	if up := rows - 1 - view.row; up > 0 {
		fmt.Fprintf(&output, "\x1b[%dA", up)
	}
	fmt.Fprintf(&output, "\r\x1b[%dG\x1b[?25h", view.column+1)
	_, err := io.WriteString(r.output, output.String())
	r.rows, r.row = rows, view.row
	return err
}

func (r *renderer) clear() error {
	if r.rows == 0 {
		return nil
	}
	err := r.draw(frame{lines: []string{""}})
	r.rows, r.row = 0, 0
	return err
}
