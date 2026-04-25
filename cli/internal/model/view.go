package model

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/model/components"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return warningStyle.Render("Loading layout...")
	}

	headerFrameW, _ := headerStyle.GetFrameSize()
	footerFrameW, _ := footerStyle.GetFrameSize()

	sidebarFrameW, sidebarFrameH := sidebarStyle.GetFrameSize()
	contentFrameW, contentFrameH := contentStyle.GetFrameSize()

	header := headerStyle.
		Width(max(0, m.width-headerFrameW)).
		Render(m.renderHeader(max(0, m.width-headerFrameW)))

	footer := footerStyle.
		Width(max(0, m.width-footerFrameW)).
		Render(m.renderFooter())

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)

	bodyHeight := max(m.height-headerHeight-footerHeight, 3)

	sidebarWidth := 24
	contentWidth := max(m.width-sidebarWidth, 20)

	var sidebarLines []string
	sidebarLines = append(sidebarLines, titleStyle.Render("Tabs"), "")

	for i, item := range m.items {
		prefix := "  "
		style := normalItemStyle

		if i == m.cursor {
			prefix = "> "
			style = selectedItemStyle
		}

		sidebarLines = append(sidebarLines, style.Render(prefix+item))
	}

	sidebar := sidebarStyle.
		Width(max(0, sidebarWidth-sidebarFrameW)).
		Height(max(0, bodyHeight-sidebarFrameH)).
		Render(strings.Join(sidebarLines, "\n"))

	selected := m.items[m.cursor]
	contentText := m.renderContent(selected)
	if selected == "Plugins" && m.pluginWizard.open {
		contentText = []string{m.renderPluginWizardModal(max(20, contentWidth-contentFrameW), max(8, bodyHeight-contentFrameH))}
	}

	content := contentStyle.
		Width(max(0, contentWidth-contentFrameW)).
		Height(max(0, bodyHeight-contentFrameH)).
		Render(strings.Join(contentText, "\n"))

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		content,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		footer,
	)
}

func (m model) renderHeader(width int) string {
	left := titleStyle.Render("k8s-manager") + subtleStyle.Render(" | context: dev-cluster | namespace: default")
	right := components.Auth{Username: m.profileLabel()}.View()

	padding := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", padding) + right
}

func (m model) renderFooter() string {
	if m.commandMode {
		return searchStyle.Render(":") + m.commandInput
	}

	help := "q quit | tab tabs | j/k move | :auth login"
	if m.session == nil {
		help = "q quit | tab tabs | j/k move | :auth login | :signup register"
	}
	if m.selectedTab() == "Plugins" {
		help = "q quit | tab tabs | arrows buttons/list | / search | enter action"
		if m.session == nil {
			help = "q quit | tab tabs | :auth login | :signup register"
		}
		if m.pluginWizard.open {
			help = "tab tabs | esc cancel | up/down input/buttons | left/right buttons | enter next/create"
		}
	}
	if m.authInProgress {
		help = "authorizing with Keycloak..."
	}

	if m.status == "" {
		return help
	}

	return fmt.Sprintf("%s %s %s", statusStyle(m.status).Render(m.status), subtleStyle.Render("|"), subtleStyle.Render(help))
}

func statusStyle(status string) lipgloss.Style {
	switch {
	case strings.Contains(status, "failed"), strings.Contains(status, "Failed"), strings.Contains(status, "error"):
		return dangerStyle
	case strings.Contains(status, "Creating"), strings.Contains(status, "Loading"), strings.Contains(status, "Uploading"):
		return warningStyle
	default:
		return searchStyle
	}
}

func (m model) profileLabel() string {
	if m.session == nil {
		return ""
	}

	return m.session.DisplayName()
}
