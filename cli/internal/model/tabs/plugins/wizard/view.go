package wizard

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/model/styles"

	"github.com/charmbracelet/lipgloss"
)

// View рисует мастер в виде окна по центру экрана
func (m Model) View(width, height int) string {
	if !m.open {
		return ""
	}

	modalWidth := min(max(48, width-8), 76)
	current := m.fields[m.step]
	inputPrefix := "  "
	if m.focus == focusInput {
		inputPrefix = "> "
	}

	value := current.value
	if value == "" {
		value = styles.Subtle.Render(current.placeholder)
	}

	lines := []string{
		styles.Title.Render(fmt.Sprintf("Create Plugin  %d/%d", m.step+1, len(m.fields))),
		"",
		styles.Search.Render(current.label),
		inputPrefix + value,
		"",
		m.renderButtons(),
	}
	if m.submitting {
		lines = append(lines, "", styles.Warning.Render("Submitting..."))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorTeal).
		Background(styles.ColorPanel).
		Padding(1, 2).
		Width(modalWidth).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) renderButtons() string {
	buttons := m.buttons()
	rendered := make([]string, 0, len(buttons))
	for i, btn := range buttons {
		style := styles.Button
		if m.focus == focusButtons && i == m.button {
			style = styles.ActiveButton
		}
		rendered = append(rendered, style.Render(btn))
	}
	return strings.Join(rendered, lipgloss.NewStyle().Background(styles.ColorPanel).Render(" "))
}
