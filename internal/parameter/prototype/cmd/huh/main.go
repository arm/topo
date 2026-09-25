// This is a throwaway huh comparison. It never reads or writes project configuration.
package main

import (
	"errors"
	"fmt"
	"os"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/arm/topo/internal/parameter/prototype/internal/demo"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	layout, err := demo.Options()
	if err != nil {
		return err
	}
	if err := demo.Banner(); err != nil {
		return err
	}
	parameters := demo.Fixtures()
	answers := make([]demo.Answer, len(parameters))
	fields := make([]huh.Field, len(parameters))
	for i, parameter := range parameters {
		fields[i] = newParameterInput(parameter, &answers[i], i+1, len(parameters))
	}
	if layout == "form" {
		if err := runForm(fields, false); err != nil {
			return err
		}
	} else {
		for i := range parameters {
			if err := runForm(fields[i:i+1], i > 0); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(os.Stderr, demo.Transcript(parameters[i], answers[i])); err != nil {
				return err
			}
		}
	}
	return demo.Report(parameters, answers)
}

func runForm(fields []huh.Field, afterAnswers bool) error {
	keys := huh.NewDefaultKeyMap()
	keys.Input.Next = key.NewBinding(key.WithKeys("enter", "tab"))
	keys.Input.Submit = key.NewBinding(key.WithKeys("enter", "tab"))
	options := []tea.ProgramOption{tea.WithInput(os.Stdin), tea.WithOutput(os.Stderr)}
	if os.Getenv("NO_COLOR") != "" {
		options = append(options, tea.WithColorProfile(colorprofile.NoTTY))
	}
	form := huh.NewForm(huh.NewGroup(fields...)).
		WithTheme(topoTheme(afterAnswers)).WithKeyMap(keys).WithAccessible(false).WithShowHelp(false).
		WithProgramOptions(options...)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return demo.ErrCanceled
		}
		return err
	}
	return nil
}

func topoTheme(afterAnswers bool) huh.Theme {
	color := func(code string) lipgloss.Style {
		style := lipgloss.NewStyle()
		if os.Getenv("NO_COLOR") == "" {
			style = style.Foreground(lipgloss.Color(code))
		}
		return style
	}
	styles := &huh.Styles{}
	if afterAnswers {
		styles.Form.Base = lipgloss.NewStyle().PaddingTop(1)
	}
	styles.Group.Base = lipgloss.NewStyle().PaddingLeft(1)
	styles.FieldSeparator = lipgloss.NewStyle().SetString("\n\n")
	styles.Focused.Base = lipgloss.NewStyle().PaddingLeft(1)
	styles.Focused.Title = color("6")
	styles.Focused.Description = lipgloss.NewStyle()
	styles.Focused.ErrorIndicator = color("1").SetString("!")
	styles.Focused.ErrorMessage = color("1")
	styles.Focused.TextInput.Prompt = color("6")
	styles.Focused.TextInput.Placeholder = color("8")
	styles.Focused.TextInput.Cursor = color("6")
	styles.Blurred = styles.Focused
	styles.Blurred.Title = lipgloss.NewStyle()
	styles.Blurred.TextInput.Placeholder = lipgloss.NewStyle()
	return huh.ThemeFunc(func(bool) *huh.Styles { return styles })
}
