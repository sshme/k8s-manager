package dialog

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/model/styles"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View(width, height int) string {
	if !m.open {
		return ""
	}

	modalWidth := min(max(48, width-8), 80)
	a := m.artifact

	title := fmt.Sprintf("%s  %s", a.PluginName, a.Version)
	meta := fmt.Sprintf("%s · %s/%s", a.PublisherName, a.OS, a.Arch)
	path := truncate(a.InstallDir, modalWidth-2)

	lines := []string{
		styles.Title.Render(title),
		styles.Subtle.Render(meta),
		"",
		styles.Subtle.Render(path),
		"",
		m.renderButtons(),
	}
	if m.working {
		lines = append(lines, "", styles.Warning.Render("Deleting plugin..."))
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
	rendered := make([]string, 0, len(buttonLabels))
	for i, label := range buttonLabels {
		style := styles.Button
		if i == m.cursor && !m.working {
			style = styles.ActiveButton
		}
		rendered = append(rendered, style.Render(label))
	}
	gap := lipgloss.NewStyle().Background(styles.ColorPanel).Render("  ")
	return strings.Join(rendered, gap)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 1 {
		return "-"
	}
	runes := []rune(s)
	return string(runes[:max-3]) + "..."
}
