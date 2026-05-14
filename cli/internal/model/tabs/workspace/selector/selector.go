// Package selector реализует выпадающий список
package selector

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	label    string
	items    []string
	cursor   int
	selected int
	open     bool
	loading  bool
}

func New(label string) Model {
	return Model{label: label, selected: -1}
}

func (m Model) Label() string { return m.label }

// SetItems заменяет список и подсвечивает defaultIdx как выбранный
func (m Model) SetItems(items []string, defaultIdx int) Model {
	m.items = items
	m.loading = false
	if len(items) == 0 {
		m.selected = -1
		m.cursor = 0
		return m
	}
	if defaultIdx < 0 || defaultIdx >= len(items) {
		defaultIdx = 0
	}
	m.selected = defaultIdx
	m.cursor = defaultIdx
	return m
}

func (m Model) SetLoading(v bool) Model {
	m.loading = v
	if v {
		m.items = nil
		m.selected = -1
	}
	return m
}

func (m Model) Open() Model {
	if len(m.items) == 0 {
		return m
	}
	m.open = true
	if m.selected >= 0 {
		m.cursor = m.selected
	}
	return m
}

func (m Model) Close() Model {
	m.open = false
	return m
}

func (m Model) IsOpen() bool   { return m.open }
func (m Model) Loading() bool  { return m.loading }
func (m Model) HasItems() bool { return len(m.items) > 0 }

func (m Model) Selected() string {
	if m.selected < 0 || m.selected >= len(m.items) {
		return ""
	}
	return m.items[m.selected]
}

func (m Model) SelectedIdx() int { return m.selected }

func (m Model) Items() []string { return m.items }

// Update обрабатывает ввод когда селектор открыт. Второе возвращаемое значение
// true, если пользователь подтвердил выбор (Enter).
func (m Model) Update(msg tea.KeyMsg) (Model, bool) {
	if !m.open {
		return m, false
	}
	switch msg.String() {
	case "esc":
		m.open = false
		return m, false
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, false
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
		return m, false
	case "enter":
		m.selected = m.cursor
		m.open = false
		return m, true
	}
	return m, false
}

// Cursor возвращает индекс подсвеченного в списке элемента
func (m Model) Cursor() int { return m.cursor }
