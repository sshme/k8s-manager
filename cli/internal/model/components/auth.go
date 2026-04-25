package components

import "github.com/charmbracelet/lipgloss"

type Auth struct {
	Username string
}

func (a Auth) View() string {
	label := "[auth]"
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 1)

	if a.Username == "" {
		return style.Render(label)
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("24")).
		Padding(0, 1).
		Render(a.Username)
}
