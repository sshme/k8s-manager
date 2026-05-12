package model

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/model/components"
	"k8s-manager/cli/internal/model/styles"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return styles.Warning.Render("Loading layout...")
	}

	headerFrameW, _ := styles.Header.GetFrameSize()
	footerFrameW, _ := styles.Footer.GetFrameSize()

	header := styles.Header.
		Width(max(0, m.width-headerFrameW)).
		Render(m.renderHeader(max(0, m.width-headerFrameW)))

	footer := styles.Footer.
		Width(max(0, m.width-footerFrameW)).
		Render(m.renderFooter())

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	bodyHeight := max(m.height-headerHeight-footerHeight, 3)

	sidebarWidth := 18
	contentWidth := max(m.width-sidebarWidth, 20)

	sidebar := sizedBlock(styles.Sidebar, sidebarWidth, bodyHeight, m.renderSidebar())
	body := sizedBlock(styles.Content, contentWidth, bodyHeight, m.tabs[m.active].View(
		max(0, contentWidth-styles.Content.GetHorizontalFrameSize()),
		max(0, bodyHeight-styles.Content.GetVerticalFrameSize()),
	))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, body),
		footer,
	)
}

func (m model) renderSidebar() string {
	lines := []string{styles.Title.Render("Tabs"), ""}
	for i, t := range m.tabs {
		prefix := "  "
		style := styles.NormalItem
		if i == m.active {
			prefix = "> "
			style = styles.SelectedItem
		}
		lines = append(lines, style.Render(prefix+t.Title()))
	}
	return strings.Join(lines, "\n")
}

func (m model) renderHeader(width int) string {
	left := styles.Title.Render("k8s-manager") + styles.Subtle.Render(" | context: dev-cluster | namespace: default")
	right := components.Auth{Username: m.profileLabel()}.View()
	padding := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", padding) + right
}

func (m model) renderFooter() string {
	if m.commandMode {
		return styles.Search.Render(":") + m.commandInput
	}

	help := m.tabs[m.active].Help()
	if m.profile.AuthInProgress() {
		help = "authorizing with Keycloak..."
	}

	if m.status == "" {
		return help
	}

	return fmt.Sprintf("%s %s %s",
		statusStyle(m.status).Render(m.status),
		styles.Subtle.Render("|"),
		styles.Subtle.Render(help),
	)
}

func statusStyle(status string) lipgloss.Style {
	switch {
	case strings.Contains(status, "failed"), strings.Contains(status, "Failed"), strings.Contains(status, "error"):
		return styles.Danger
	case strings.Contains(status, "Creating"), strings.Contains(status, "Loading"), strings.Contains(status, "Uploading"):
		return styles.Warning
	default:
		return styles.Search
	}
}

func (m model) profileLabel() string {
	session := m.profile.Session()
	if session == nil {
		return ""
	}
	return session.DisplayName()
}

// sizedBlock рендерит content с заданной outer-шириной и высотой
func sizedBlock(s lipgloss.Style, outerW, outerH int, content string) string {
	return s.
		Width(max(0, outerW-s.GetHorizontalBorderSize())).
		Height(max(0, outerH-s.GetVerticalBorderSize())).
		Render(content)
}
