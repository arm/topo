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
	if e.layout == "sequential" && e.active > 0 {
		view.lines = append(view.lines, "")
	}
	details := strings.Split(e.parameters[e.active].Details(), "\n")
	activeHeight := len(details) + 2
	if e.message != "" {
		activeHeight++
	}
	start, end := e.active, e.active+1
	if e.layout == "form" {
		visible := 1 + max(0, (height-activeHeight)/3)
		start = max(0, e.active-visible+1)
		end = min(len(e.parameters), start+visible)
	}
	for i := start; i < end; i++ {
		if i > start {
			view.lines = append(view.lines, "")
		}
		parameter, answer := e.parameters[i], e.answers[i]
		heading := clip(" "+parameter.Heading(i+1, len(e.parameters)), width-1)
		if i != e.active {
			value := answer.Text
			if value == "" {
				value = parameter.Placeholder()
			}
			view.lines = append(view.lines, heading, clip("   "+value, width-1))
			continue
		}
		view.lines = append(view.lines, palette.Color(term.Cyan, heading))
		for _, line := range details {
			view.lines = append(view.lines, clip(" "+line, width-1))
		}
		view.row = len(view.lines)
		input, column := inputLine(parameter, answer, e.cursors[i], width, palette)
		view.column = column
		view.lines = append(view.lines, input)
		if e.message != "" {
			view.lines = append(view.lines, palette.Color(term.Red, clip(" ! "+e.message, width-1)))
		}
	}
	return view
}

func inputLine(parameter demo.Parameter, answer demo.Answer, cursor, width int, palette term.Palette) (string, int) {
	const prompt = " > "
	if answer.Text == "" {
		return palette.Color(term.Cyan, prompt) + palette.Color(term.Dim, clip(parameter.Placeholder(), width-len(prompt)-1)), len(prompt)
	}
	text := []rune(answer.Text)
	start := max(0, cursor-max(1, width-len(prompt)-2))
	return palette.Color(term.Cyan, prompt) + clip(string(text[start:]), width-len(prompt)-1), len(prompt) + cursor - start
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
