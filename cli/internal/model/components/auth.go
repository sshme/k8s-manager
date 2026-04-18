package components

import tea "github.com/charmbracelet/bubbletea"

type Auth struct {
	onEnter func() tea.Cmd
}
