package parameter

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func browserKey(m browser, key tea.KeyMsg) browser {
	next, _ := m.Update(key)
	return next.(browser)
}

func browserRune(m browser, text string) browser {
	return browserKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
}

func TestBrowser(t *testing.T) {
	definitions := []Definition{{Name: "FIRST", Description: "First setting"}, {Name: "SECOND", Description: "Listen port", Required: true}}
	newModel := func() browser {
		return newBrowser(definitions, Values{"FIRST": "old", "SECOND": "8080"}, map[string]string{"FIRST": ".env", "SECOND": "shell environment"}, ".env.topo")
	}
	t.Run("search matches descriptions as you type", func(t *testing.T) {
		m := browserRune(newModel(), "/")
		m = browserRune(m, "PORT")
		assert.Equal(t, []int{1}, m.visible())
	})
	t.Run("edit stages only the selected value", func(t *testing.T) {
		m := browserKey(newModel(), tea.KeyMsg{Type: tea.KeyDown})
		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		m.input.SetValue("9090")
		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		assert.Equal(t, Values{"SECOND": "9090"}, m.updates)
		assert.Equal(t, "8080", m.current["SECOND"])
		assert.False(t, m.saved)
	})
	t.Run("escape discards the active edit", func(t *testing.T) {
		m := browserKey(newModel(), tea.KeyMsg{Type: tea.KeyEnter})
		m.input.SetValue("changed")
		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		assert.Empty(t, m.updates)
	})
	t.Run("undo restores the inherited value", func(t *testing.T) {
		m := newModel()
		m.updates["FIRST"] = "changed"
		m = browserRune(m, "u")
		assert.Empty(t, m.updates)
		assert.Equal(t, "old", m.value("FIRST"))
	})
	t.Run("save blocks missing required values and selects the missing row", func(t *testing.T) {
		m := newModel()
		delete(m.current, "SECOND")
		m.query = "first"
		m = browserKey(m, tea.KeyMsg{Type: tea.KeyCtrlS})
		assert.False(t, m.saved)
		assert.NotEmpty(t, m.message)
		assert.Equal(t, 1, m.cursor)
		assert.Empty(t, m.query)
	})
	t.Run("save accepts inherited required values without copying them", func(t *testing.T) {
		m := browserKey(newModel(), tea.KeyMsg{Type: tea.KeyCtrlS})
		assert.True(t, m.saved)
		assert.Empty(t, m.updates)
	})
	t.Run("empty optional value is an explicit update", func(t *testing.T) {
		m := browserKey(newModel(), tea.KeyMsg{Type: tea.KeyEnter})
		m.input.SetValue("")
		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		assert.Equal(t, Values{"FIRST": ""}, m.updates)
	})
	t.Run("values are hidden until explicitly revealed", func(t *testing.T) {
		m := newModel()
		assert.NotContains(t, m.View(), "8080")
		m = browserRune(m, "v")
		assert.Contains(t, m.View(), "8080")
	})
	t.Run("no matches is safe to navigate and edit", func(t *testing.T) {
		m := newModel()
		m.query = "no matches"
		m = browserKey(m, tea.KeyMsg{Type: tea.KeyDown})
		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		assert.Contains(t, m.View(), "No matching")
		assert.Empty(t, m.mode)
	})
	t.Run("escape restores the previous search", func(t *testing.T) {
		m := newModel()
		m.query = "first"
		m = browserRune(m, "/")
		m.input.SetValue("second")
		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEsc})
		assert.Equal(t, "first", m.query)
	})
	t.Run("render fits terminal height", func(t *testing.T) {
		m := newModel()
		for range 100 {
			m.definitions = append(m.definitions, Definition{Name: "MORE", Example: "example"})
		}
		m.cursor = 80
		require.LessOrEqual(t, strings.Count(m.View(), "\n"), m.height)
	})
	t.Run("multiline values cannot be silently flattened by the editor", func(t *testing.T) {
		m := newModel()
		m.current["FIRST"] = "first\nsecond"

		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEnter})

		assert.Empty(t, m.mode)
		assert.Contains(t, m.message, "Multiline")
		assert.Empty(t, m.updates)
	})
	t.Run("restoring the original value removes the staged edit", func(t *testing.T) {
		m := newModel()
		m.updates["FIRST"] = "changed"
		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEnter})
		m.input.SetValue("old")

		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEnter})

		assert.Empty(t, m.updates)
	})
	t.Run("cancel does not mark staged edits for saving", func(t *testing.T) {
		m := newModel()
		m.updates["FIRST"] = "changed"

		m = browserKey(m, tea.KeyMsg{Type: tea.KeyEsc})

		assert.False(t, m.saved)
	})
	t.Run("definitions have stable alphabetical order without mutating caller input", func(t *testing.T) {
		definitions := []Definition{{Name: "Z"}, {Name: "A"}}

		m := newBrowser(definitions, nil, nil, ".env.topo")

		assert.Equal(t, "A", m.definitions[0].Name)
		assert.Equal(t, "Z", definitions[0].Name)
	})
	t.Run("terminal controls are sanitized", func(t *testing.T) {
		assert.Equal(t, "red next", terminalText("\x1b[31mred\x1b[0m\nnext"))
	})
}
