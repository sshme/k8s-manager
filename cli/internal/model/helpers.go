package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) selectedTab() string {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return ""
	}

	return m.items[m.cursor]
}

func (m *model) loadPlugins() tea.Cmd {
	m.pluginLoading = true
	return loadPluginsCmd(m.marketService, m.pluginSearch)
}

func nonEmpty(values ...string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "UNKNOWN" {
			items = append(items, value)
		}
	}
	return items
}

func truncate(value string, maxWidth int) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if lipgloss.Width(value) <= maxWidth {
		return value
	}

	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxWidth-3 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

func shortChecksum(checksum string) string {
	if len(checksum) <= 12 {
		return checksum
	}
	return checksum[:12]
}
