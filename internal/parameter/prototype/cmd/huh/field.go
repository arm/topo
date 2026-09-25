package main

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/arm/topo/internal/parameter/prototype/internal/demo"
)

// The stock input only returns a string. This wrapper retains keep versus explicit-empty intent.
// It delegates editing, validation presentation, focus, and layout to huh.
type parameterInput struct {
	*huh.Input
	parameter demo.Parameter
	answer    *demo.Answer
	focused   bool
	inputErr  error
}

func newParameterInput(parameter demo.Parameter, answer *demo.Answer) *parameterInput {
	field := &parameterInput{parameter: parameter, answer: answer}
	field.Input = huh.NewInput().Key(parameter.Name).Title(parameter.Heading()).
		Prompt("> ").Value(&answer.Text).CharLimit(4096).
		Validate(func(value string) error {
			_, _, err := demo.Resolve(parameter, demo.Answer{Text: value, Empty: answer.Empty})
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
		f.answer.Empty = false
	}
	if pressed, ok := message.(tea.KeyPressMsg); ok {
		f.inputErr = nil
		switch pressed.String() {
		case "ctrl+x":
			*f.answer = demo.Answer{Empty: true}
			f.Input.Value(&f.answer.Text)
			message = tea.KeyPressMsg{Code: tea.KeyEnter}
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
		case "backspace", "delete":
			f.answer.Empty = false
		default:
			if pressed.Text != "" {
				f.answer.Empty = false
			}
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
	f.Input.Placeholder(f.parameter.Placeholder(*f.answer))
	description := demo.Preview(f.parameter, *f.answer)
	if f.focused {
		description = f.parameter.Description + "\nOn Enter: " + description
	}
	f.Input.Description(description)
	return f.Input.View()
}

func (f *parameterInput) KeyBinds() []key.Binding {
	bindings := f.Input.KeyBinds()
	return append(bindings,
		key.NewBinding(key.WithKeys("right"), key.WithHelp("→ (blank)", "edit current")),
		key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "set empty")),
		key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "clear draft")),
		key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "cancel")))
}
