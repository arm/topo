package main

import (
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/arm/topo/internal/parameter/prototype/internal/demo"
)

type editor struct {
	parameters []demo.Parameter
	answers    []demo.Answer
	cursors    []int
	active     int
	layout     string
	message    string
}

func newEditor(layout string) *editor {
	parameters := demo.Fixtures()
	return &editor{
		parameters: parameters,
		answers:    make([]demo.Answer, len(parameters)),
		cursors:    make([]int, len(parameters)),
		layout:     layout,
	}
}

func (e *editor) update(event inputEvent) (bool, error) {
	e.message = ""
	answer := &e.answers[e.active]
	switch event.key {
	case "cancel":
		return false, demo.ErrCanceled
	case "previous":
		if e.layout == "form" {
			e.active = max(0, e.active-1)
		}
		return false, nil
	case "next":
		return e.advance(), nil
	case "right":
		current := e.parameters[e.active].Current
		if answer.Text == "" && current != nil {
			*answer = demo.Answer{Text: current.Value}
			e.cursors[e.active] = utf8.RuneCountInString(answer.Text)
			return false, nil
		}
	case "invalid":
		e.message = "Invalid UTF-8 input rejected"
		return false, nil
	}
	if err := e.edit(event); err != nil {
		e.message = err.Error()
	}
	return false, nil
}

func (e *editor) advance() bool {
	if _, _, err := demo.Resolve(e.parameters[e.active], e.answers[e.active]); err != nil {
		e.message = err.Error()
		return false
	}
	e.active++
	return e.active == len(e.parameters)
}

func (e *editor) edit(event inputEvent) error {
	answer := &e.answers[e.active]
	text := []rune(answer.Text)
	cursor := e.cursors[e.active]
	switch event.key {
	case "text":
		if err := demo.ValidateText(event.text); err != nil {
			return err
		}
		text = slices.Insert(text, cursor, []rune(event.text)...)
		cursor += utf8.RuneCountInString(event.text)
	case "backspace":
		if cursor > 0 {
			text = slices.Delete(text, cursor-1, cursor)
			cursor--
		}
	case "delete":
		if cursor < len(text) {
			text = slices.Delete(text, cursor, cursor+1)
		}
	case "clear":
		text, cursor = nil, 0
	case "left":
		cursor = max(0, cursor-1)
	case "right":
		cursor = min(len(text), cursor+1)
	case "home":
		cursor = 0
	case "end":
		cursor = len(text)
	}
	if len(text) > 4096 {
		return fmt.Errorf("prototype input limit is 4096 code points")
	}
	answer.Text = string(text)
	e.cursors[e.active] = cursor
	return nil
}
