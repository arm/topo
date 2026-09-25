package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/arm/topo/internal/parameter/prototype/internal/demo"
)

// Keep the current value separate from the draft so blank input does not create an update.
type parameterInput struct {
	*huh.Input
	parameter demo.Parameter
	answer    *demo.Answer
	focused   bool
	inputErr  error
}

func newParameterInput(parameter demo.Parameter, answer *demo.Answer, number, total int) *parameterInput {
	field := &parameterInput{parameter: parameter, answer: answer}
	field.Input = huh.NewInput().Key(parameter.Name).Title(parameter.Heading(number, total)).
		Prompt("> ").Value(&answer.Text).CharLimit(4096).
		Validate(func(value string) error {
			_, _, err := demo.Resolve(parameter, demo.Answer{Text: value})
			return err
		})
	return field
}

func (f *parameterInput) Update(message tea.Msg) (huh.Model, tea.Cmd) {
	if pasted, ok := message.(tea.PasteMsg); ok {
		f.inputErr = demo.ValidateText(pasted.Content)
		if f.inputErr != nil {
			return f, nil
		}
	}
	if pressed, ok := message.(tea.KeyPressMsg); ok {
		f.inputErr = nil
		switch pressed.String() {
		case "ctrl+d":
			return f, tea.Interrupt
		case "right":
			if f.answer.Text == "" && f.parameter.Current != nil {
				*f.answer = demo.Answer{Text: f.parameter.Current.Value}
				f.Input.Value(&f.answer.Text)
				return f, nil
			}
		case "ctrl+u":
			*f.answer = demo.Answer{}
			f.Input.Value(&f.answer.Text)
		}
	}
	_, command := f.Input.Update(message)
	// Returning the embedded Input here would silently discard this wrapper after one event.
	return f, command
}

func (f *parameterInput) Focus() tea.Cmd {
	f.focused = true
	return f.Input.Focus()
}

func (f *parameterInput) Blur() tea.Cmd {
	f.focused = false
	return f.Input.Blur()
}

func (f *parameterInput) Error() error {
	if f.inputErr != nil {
		return f.inputErr
	}
	return f.Input.Error()
}

func (f *parameterInput) View() string {
	f.Input.Placeholder(f.parameter.Placeholder())
	f.Input.Description("").Prompt("  ")
	if f.focused {
		f.Input.Description(f.parameter.Details()).Prompt("> ")
	}
	return f.Input.View()
}
