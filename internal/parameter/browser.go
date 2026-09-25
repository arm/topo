package parameter

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/arm/topo/internal/env"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// BrowserResolver stages edits in a searchable overview before returning any updates.
type BrowserResolver struct {
	Input       io.Reader
	Output      io.Writer
	EnvFiles    []string
	Destination string
}

func (r *BrowserResolver) Resolve(definitions []Definition, current Values) (Values, error) {
	sources, err := env.Sources(r.EnvFiles, current)
	if err != nil {
		return nil, err
	}
	model := newBrowser(definitions, current, sources, r.Destination)
	result, err := tea.NewProgram(model, tea.WithInput(r.Input), tea.WithOutput(r.Output), tea.WithAltScreen()).Run()
	if err != nil {
		return nil, fmt.Errorf("configure browser: %w", err)
	}
	final := result.(browser)
	if !final.saved {
		return nil, fmt.Errorf("configuration cancelled; nothing written")
	}
	return final.updates, nil
}

type browser struct {
	definitions           []Definition
	current               Values
	sources               map[string]string
	updates               Values
	destination           string
	cursor, width, height int
	query                 string
	previousQuery         string
	input                 textinput.Model
	mode                  string
	message               string
	saved                 bool
	reveal                bool
}

func newBrowser(definitions []Definition, current Values, sources map[string]string, destination string) browser {
	definitions = slices.Clone(definitions)
	slices.SortFunc(definitions, func(a, b Definition) int { return strings.Compare(a.Name, b.Name) })
	input := textinput.New()
	input.CharLimit = 0
	return browser{definitions: definitions, current: current, sources: sources, updates: Values{}, destination: destination, width: 80, height: 24, input: input}
}

func (m browser) Init() tea.Cmd { return nil }

func (m browser) visible() []int {
	var indices []int
	for i, d := range m.definitions {
		if strings.Contains(strings.ToLower(d.Name+" "+d.Description), strings.ToLower(m.query)) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m browser) value(name string) string {
	if value, ok := m.updates[name]; ok {
		return value
	}
	return m.current[name]
}

func (m browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		m.input.Width = max(10, size.Width-10)
		return m, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if ok && key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.mode != "" {
		return m.updateInput(msg)
	}
	if !ok {
		return m, nil
	}
	indices := m.visible()
	switch key.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(max(0, len(indices)-1), m.cursor+1)
	case "home":
		m.cursor = 0
	case "end":
		m.cursor = max(0, len(indices)-1)
	case "pgup":
		m.cursor = max(0, m.cursor-max(1, m.height-17))
	case "pgdown":
		m.cursor = min(max(0, len(indices)-1), m.cursor+max(1, m.height-17))
	case "/":
		m.mode = "search"
		m.previousQuery = m.query
		m.input.SetValue(m.query)
		return m, m.input.Focus()
	case "v":
		m.reveal = !m.reveal
	case "u":
		if len(indices) > 0 {
			delete(m.updates, m.definitions[indices[m.cursor]].Name)
		}
		m.message = ""
	case "enter":
		if len(indices) > 0 {
			if strings.ContainsAny(m.value(m.definitions[indices[m.cursor]].Name), "\r\n") {
				m.message = "Multiline value: use topo configure NAME=VALUE to preserve line breaks."
				return m, nil
			}
			m.mode = "edit"
			m.input.SetValue(m.value(m.definitions[indices[m.cursor]].Name))
			m.input.EchoMode = textinput.EchoPassword
			if m.reveal {
				m.input.EchoMode = textinput.EchoNormal
			}
			return m, m.input.Focus()
		}
	case "ctrl+s":
		if err := validateRequiredValues(m.definitions, m.updates, m.current); err != nil {
			m.message = "Required values are missing. Fill the rows marked ! before saving."
			m.query = ""
			for i, d := range m.definitions {
				if d.Required && m.value(d.Name) == "" {
					m.cursor = i
					break
				}
			}
			return m, nil
		}
		m.saved = true
		return m, tea.Quit
	}
	return m, nil
}

func (m browser) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			if m.mode == "search" {
				m.query = m.previousQuery
				m.cursor = 0
			}
			m.mode = ""
			m.message = ""
			m.input.Blur()
			m.input.EchoMode = textinput.EchoNormal
			return m, nil
		case "enter":
			if m.mode == "search" {
				m.query = m.input.Value()
				m.cursor = 0
			} else {
				d := m.definitions[m.visible()[m.cursor]]
				value := m.input.Value()
				if d.Required && value == "" {
					m.message = "This parameter requires a value."
					return m, nil
				}
				if value == m.current[d.Name] {
					delete(m.updates, d.Name)
				} else {
					m.updates[d.Name] = value
				}
			}
			m.mode = ""
			m.message = ""
			m.input.Blur()
			m.input.EchoMode = textinput.EchoNormal
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.mode == "search" {
		m.query = m.input.Value()
		m.cursor = 0
	}
	return m, cmd
}

var browserAccent = lipgloss.NewStyle().Foreground(lipgloss.Color("115")).Bold(true)
var browserMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

// Keep file metadata and values from injecting terminal control sequences.
func terminalText(text string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || (r >= 128 && r <= 159) {
			return ' '
		}
		return r
	}, ansi.Strip(text))
}

func (m browser) parameterRow(d Definition, width int) string {
	marker := " "
	_, edited := m.updates[d.Name]
	if edited {
		marker = "*"
	} else if d.Required && m.value(d.Name) == "" {
		marker = "!"
	}
	value := "not set"
	_, assigned := m.current[d.Name]
	if assigned || edited {
		value = "(empty)"
	}
	if m.value(d.Name) != "" {
		value = "••••••"
		if m.reveal {
			value = terminalText(m.value(d.Name))
		}
	}
	source := m.sources[d.Name]
	if source == "" {
		source = "not set"
	}
	if edited {
		source = "staged edit"
	}
	nameWidth, valueWidth := min(24, width/3), min(16, width/4)
	name := lipgloss.NewStyle().Width(nameWidth).Render(ansi.Truncate(terminalText(d.Name), nameWidth, "…"))
	value = lipgloss.NewStyle().Width(valueWidth).Render(ansi.Truncate(value, valueWidth, "…"))
	return ansi.Truncate(fmt.Sprintf("%s %s  %s  %s", marker, name, value, terminalText(source)), width, "…")
}

func (m browser) parameterDetails(d Definition) []string {
	requirement := "optional"
	if d.Required {
		requirement = "required"
	}
	lines := []string{browserAccent.Render(terminalText(d.Name)) + " · " + requirement, terminalText(d.Description)}
	source := m.sources[d.Name]
	if source == "" {
		source = "not set"
	}
	precedence := "Precedence: shell > later env files > earlier env files"
	if source == "shell environment" {
		precedence = "Shell overrides saved edits. Unset " + d.Name + " to use the file."
	} else if source == "not set" {
		precedence = "No environment assignment. Compose fallbacks may still apply."
	}
	lines = append(lines, "Current source: "+terminalText(source), browserMuted.Render(terminalText(precedence)))
	if d.Example != "" {
		lines = append(lines, "Example: "+terminalText(d.Example))
	}
	return lines
}

func (m browser) View() string {
	if m.width < 45 || m.height < 18 {
		return "Resize terminal to at least 45 × 18.\nCtrl+C cancels without writing.\n"
	}
	width := max(10, m.width-4)
	var lines []string
	lines = append(lines, browserAccent.Render("CONFIGURE  /  Project parameters"), browserMuted.Render("Save to "+terminalText(m.destination)+" · only edited values are written"), "")
	lines = append(lines, fmt.Sprintf("%d parameters · %d staged edits   Filter: %s", len(m.definitions), len(m.updates), terminalText(m.query)))
	indices := m.visible()
	count := max(1, m.height-17)
	start := max(0, m.cursor-count+1)
	for _, i := range indices[start:min(len(indices), start+count)] {
		row := m.parameterRow(m.definitions[i], width-2)
		if len(indices) > 0 && i == indices[m.cursor] {
			row = browserAccent.Render("› " + row)
		} else {
			row = "  " + row
		}
		lines = append(lines, row)
	}
	if len(indices) == 0 {
		lines = append(lines, "No matching parameters. Press / to change the filter.")
	}
	lines = append(lines, "")
	if len(indices) > 0 {
		lines = append(lines, m.parameterDetails(m.definitions[indices[m.cursor]])...)
	}
	if m.mode != "" {
		lines = append(lines, "", m.mode+" › "+m.input.View(), "Enter apply · Esc back")
	} else {
		lines = append(lines, "", "↑↓ navigate · / search · Enter edit · u undo · v reveal values", "Ctrl+S save · Esc cancel    ! missing required · * edited")
	}
	if m.message != "" {
		lines = append(lines, m.message)
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "…")
	}
	return "\n  " + strings.Join(lines, "\n  ") + "\n"
}
