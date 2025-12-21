package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	pb "k8s-manager/proto/market"
	"k8s-manager/cli/internal/k8s"
)

type model struct {
	marketClient pb.MarketServiceClient
	k8sClient    *k8s.Client
	
	plugins      []*pb.Plugin
	selectedIdx  int
	view         string // "list", "details", "install"
	loading      bool
	error        string
	message      string
	
	width        int
	height       int
}

func NewModel(marketClient pb.MarketServiceClient, k8sClient *k8s.Client) *model {
	return &model{
		marketClient: marketClient,
		k8sClient:    k8sClient,
		view:         "list",
		selectedIdx:  0,
		loading:      true,
	}
}

func (m *model) Init() tea.Cmd {
	return m.loadPlugins
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.view == "list" && m.selectedIdx > 0 {
				m.selectedIdx--
			}
			return m, nil
		case "down", "j":
			if m.view == "list" && m.selectedIdx < len(m.plugins)-1 {
				m.selectedIdx++
			}
			return m, nil
		case "enter":
			if m.view == "list" {
				m.view = "details"
				return m, nil
			}
		case "i":
			if m.view == "details" {
				return m, m.installPlugin
			}
		case "esc", "b":
			if m.view == "details" {
				m.view = "list"
				return m, nil
			}
		case "/":
			// Search mode (simplified - in production add search input)
			return m, nil
		}

	case pluginsLoadedMsg:
		m.plugins = msg.plugins
		m.loading = false
		return m, nil

	case pluginInstalledMsg:
		m.message = fmt.Sprintf("✓ Plugin '%s' installed successfully!", msg.pluginName)
		m.view = "list"
		return m, nil

	case errorMsg:
		m.error = msg.err
		m.loading = false
		return m, nil
	}

	return m, nil
}

func (m *model) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.error != "" {
		return m.renderError()
	}

	if m.message != "" {
		// Clear message after a moment
		defer func() { m.message = "" }()
	}

	switch m.view {
	case "list":
		return m.renderList()
	case "details":
		return m.renderDetails()
	default:
		return m.renderList()
	}
}

func (m *model) renderLoading() string {
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Loading plugins..."),
	)
}

func (m *model) renderError() string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("1"))

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		errorStyle.Render(fmt.Sprintf("Error: %s\n\nPress 'q' to quit", m.error)),
	)
}

func (m *model) renderList() string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1).
		MarginBottom(1)

	b.WriteString(headerStyle.Render("K8s Manager - Plugin Marketplace"))
	b.WriteString("\n\n")

	// Message
	if m.message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			MarginBottom(1)
		b.WriteString(msgStyle.Render(m.message))
		b.WriteString("\n")
	}

	// Plugins list
	if len(m.plugins) == 0 {
		b.WriteString("No plugins available.\n")
	} else {
		for i, plugin := range m.plugins {
			style := lipgloss.NewStyle().Padding(0, 1)
			if i == m.selectedIdx {
				style = style.
					Background(lipgloss.Color("62")).
					Foreground(lipgloss.Color("230"))
			}

			category := plugin.Category.String()
			rating := fmt.Sprintf("⭐ %.1f", plugin.Rating)
			downloads := fmt.Sprintf("📥 %d", plugin.Downloads)

			line := fmt.Sprintf("%s | %s | %s | %s",
				plugin.Name,
				category,
				rating,
				downloads,
			)
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(2)

	help := "↑/↓: Navigate | Enter: View Details | i: Install | q: Quit"
	b.WriteString(footerStyle.Render(help))

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Left, lipgloss.Top,
		b.String(),
	)
}

func (m *model) renderDetails() string {
	if m.selectedIdx >= len(m.plugins) {
		return m.renderList()
	}

	plugin := m.plugins[m.selectedIdx]

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render(plugin.Name))
	b.WriteString("\n\n")

	// Details
	detailStyle := lipgloss.NewStyle().MarginBottom(1)
	b.WriteString(detailStyle.Render(fmt.Sprintf("Version: %s", plugin.Version)))
	b.WriteString(detailStyle.Render(fmt.Sprintf("Author: %s", plugin.Author)))
	b.WriteString(detailStyle.Render(fmt.Sprintf("Category: %s", plugin.Category.String())))
	b.WriteString(detailStyle.Render(fmt.Sprintf("Rating: ⭐ %.1f", plugin.Rating)))
	b.WriteString(detailStyle.Render(fmt.Sprintf("Downloads: %d", plugin.Downloads)))
	b.WriteString("\n")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginBottom(1)
	b.WriteString(descStyle.Render(plugin.Description))
	b.WriteString("\n")

	// Tags
	if len(plugin.Tags) > 0 {
		tagStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
			MarginRight(1)

		b.WriteString("\nTags: ")
		for _, tag := range plugin.Tags {
			b.WriteString(tagStyle.Render(tag))
		}
		b.WriteString("\n")
	}

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(2)

	help := "i: Install Plugin | b/esc: Back | q: Quit"
	b.WriteString(footerStyle.Render(help))

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Left, lipgloss.Top,
		b.String(),
	)
}

// Commands

type pluginsLoadedMsg struct {
	plugins []*pb.Plugin
}

type pluginInstalledMsg struct {
	pluginName string
}

type errorMsg struct {
	err string
}

func (m *model) loadPlugins() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.marketClient.ListPlugins(ctx, &pb.ListPluginsRequest{
			Page:     1,
			PageSize: 50,
		})
		if err != nil {
			return errorMsg{err: fmt.Sprintf("Failed to load plugins: %v", err)}
		}
		return pluginsLoadedMsg{plugins: resp.Plugins}
	}
}

func (m *model) installPlugin() tea.Cmd {
	return func() tea.Msg {
		if m.selectedIdx >= len(m.plugins) {
			return errorMsg{err: "Invalid plugin selection"}
		}

		plugin := m.plugins[m.selectedIdx]

		ctx := context.Background()
		resp, err := m.marketClient.InstallPlugin(ctx, &pb.InstallPluginRequest{
			Id:      plugin.Id,
			Version: plugin.Version,
		})
		if err != nil {
			return errorMsg{err: fmt.Sprintf("Failed to install plugin: %v", err)}
		}

		// Apply manifest to Kubernetes
		if resp.Manifest != nil {
			for _, resource := range resp.Manifest.Resources {
				if err := m.k8sClient.ApplyManifest(ctx, resource.Yaml); err != nil {
					return errorMsg{err: fmt.Sprintf("Failed to apply manifest: %v", err)}
				}
			}
		}

		return pluginInstalledMsg{pluginName: plugin.Name}
	}
}

