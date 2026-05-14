package selector

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/model/styles"

	"github.com/charmbracelet/lipgloss"
)

const arrow = " ▾"

// ViewChip рендерит кнопку с выбранным значением
func (m Model) ViewChip(focused bool, width int) string {
	value := m.Selected()
	if m.loading {
		value = "loading..."
	} else if value == "" {
		value = "(none)"
	}
	text := fmt.Sprintf("%s: %s%s", m.label, truncate(value, max(8, width-len(m.label)-6)), arrow)

	style := styles.Card
	if focused {
		style = styles.SelectedCard
	}
	return style.Render(text)
}

// ViewPopup рендерит вертикальный список под кнопкой.
// maxHeight ограничивает высоту с учётом рамки
func (m Model) ViewPopup(maxHeight int) string {
	if !m.open || len(m.items) == 0 {
		return ""
	}

	visible := len(m.items)
	if maxHeight > 2 && visible > maxHeight-2 {
		visible = maxHeight - 2
	}

	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(m.items) {
		end = len(m.items)
	}

	lines := make([]string, 0, end-start)
	maxItemWidth := 0
	for i := start; i < end; i++ {
		if w := lipgloss.Width(m.items[i]); w > maxItemWidth {
			maxItemWidth = w
		}
	}

	for i := start; i < end; i++ {
		marker := "  "
		if i == m.cursor {
			marker = "> "
		}
		line := marker + m.items[i]
		if i == m.cursor {
			line = styles.SelectedItem.Render(line)
		}
		lines = append(lines, line)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorTeal).
		Background(styles.ColorPanelAlt).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
	return box
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
