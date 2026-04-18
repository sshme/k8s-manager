package components

import tea "github.com/charmbracelet/bubbletea"

type Component interface {
	Update(msg tea.Msg) (Component, tea.Cmd)
	View(focused bool) string
}
