package plugins

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
