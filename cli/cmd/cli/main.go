package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	width  int
	height int
	items  []string
	cursor int
}

func initialModel() model {
	return model{
		items: []string{
			"Services",
			"Plugins",
			"Logs",
		},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading layout..."
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, true, false)

	sidebarStyle := lipgloss.NewStyle().
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		Height(0)

	contentStyle := lipgloss.NewStyle().
		Padding(1).
		Border(lipgloss.NormalBorder(), true, false, false, false)

	footerStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), true, false, false, false)

	selectedItemStyle := lipgloss.NewStyle().Bold(true)
	normalItemStyle := lipgloss.NewStyle()

	headerFrameW, _ := headerStyle.GetFrameSize()
	footerFrameW, _ := footerStyle.GetFrameSize()

	sidebarFrameW, sidebarFrameH := sidebarStyle.GetFrameSize()
	contentFrameW, contentFrameH := contentStyle.GetFrameSize()

	header := headerStyle.
		Width(max(0, m.width-headerFrameW)).
		Render("k8s-manager | context: dev-cluster | namespace: default")

	footer := footerStyle.
		Width(max(0, m.width-footerFrameW)).
		Render("try q,j,k")

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)

	bodyHeight := max(m.height-headerHeight-footerHeight, 3)

	sidebarWidth := 24
	contentWidth := max(m.width-sidebarWidth, 20)

	var sidebarLines []string
	sidebarLines = append(sidebarLines, "Tabs", "")

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
	contentText := []string{
		fmt.Sprintf("Selected tab: %s", selected),
		"",
		"Main workspace.",
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

func main() {
	fmt.Println("Hello")
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
