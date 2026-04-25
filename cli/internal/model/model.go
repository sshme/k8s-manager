package model

import (
	"context"
	"fmt"
	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/model/components"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	width          int
	height         int
	items          []string
	cursor         int
	status         string
	commandMode    bool
	commandInput   string
	authInProgress bool
	authService    *auth.Service
	session        *auth.Session
}

type authLoadedMsg struct {
	session *auth.Session
	err     error
}

type authCompletedMsg struct {
	session *auth.Session
	err     error
}

type authAction string

const (
	authActionLogin  authAction = "auth"
	authActionSignup authAction = "signup"
)

func New(authService *auth.Service) model {
	return model{
		items: []string{
			"Services",
			"Plugins",
			"Logs",
		},
		status:      "Press :auth to sign in or :signup to create an account",
		authService: authService,
	}
}

func (m model) Init() tea.Cmd {
	return loadSessionCmd(m.authService)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case authLoadedMsg:
		if msg.err == nil && msg.session != nil {
			m.session = msg.session
			m.status = fmt.Sprintf("Authorized as %s", msg.session.DisplayName())
			return m, nil
		}

		if msg.err == auth.ErrSessionNotFound || msg.err == nil {
			m.status = "Press :auth to sign in or :signup to create an account"
			return m, nil
		}

		m.status = fmt.Sprintf("Failed to restore session: %v", msg.err)
	case authCompletedMsg:
		m.authInProgress = false
		m.commandMode = false
		m.commandInput = ""

		if msg.err != nil {
			m.status = fmt.Sprintf("Authentication failed: %v", msg.err)
			return m, nil
		}

		m.session = msg.session
		m.status = fmt.Sprintf("Authenticated as %s", msg.session.DisplayName())
	case tea.KeyMsg:
		if m.commandMode {
			return m.updateCommandMode(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case ":":
			if m.authInProgress {
				m.status = "Authentication is already in progress"
				return m, nil
			}
			m.commandMode = true
			m.commandInput = ""
			m.status = "Command mode"
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
		Border(lipgloss.NormalBorder(), false, false, true, false)

	sidebarStyle := lipgloss.NewStyle().
		Padding(0, 0, 0, 1).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		Height(0)

	contentStyle := lipgloss.NewStyle().
		Padding(1)

	footerStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false)

	selectedItemStyle := lipgloss.NewStyle().Bold(true)
	normalItemStyle := lipgloss.NewStyle()

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

func (m model) updateCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.commandMode = false
		m.commandInput = ""
		m.status = "Command canceled"
		return m, nil
	case tea.KeyEnter:
		command := strings.TrimSpace(m.commandInput)
		m.commandInput = ""
		return m.runCommand(command)
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.commandInput) > 0 {
			m.commandInput = m.commandInput[:len(m.commandInput)-1]
		}
		return m, nil
	}

	if len(msg.Runes) > 0 {
		m.commandInput += string(msg.Runes)
	}

	return m, nil
}

func (m model) runCommand(command string) (tea.Model, tea.Cmd) {
	switch command {
	case "":
		m.commandMode = false
		m.status = "Command mode"
		return m, nil
	case "auth":
		if m.authInProgress {
			m.commandMode = false
			m.status = "Authentication is already running"
			return m, nil
		}

		m.authInProgress = true
		m.status = "Opening Keycloak in browser..."
		return m, authenticateCmd(m.authService, authActionLogin)
	case "signup", "register":
		if m.authInProgress {
			m.commandMode = false
			m.status = "Authentication is already running"
			return m, nil
		}

		m.authInProgress = true
		m.status = "Opening Keycloak registration in browser..."
		return m, authenticateCmd(m.authService, authActionSignup)
	default:
		m.commandMode = false
		m.status = fmt.Sprintf("Unknown command: %s", command)
		return m, nil
	}
}

func (m model) renderHeader(width int) string {
	left := "k8s-manager | context: dev-cluster | namespace: default"
	right := components.Auth{Username: m.profileLabel()}.View()

	padding := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", padding) + right
}

func (m model) renderFooter() string {
	if m.commandMode {
		return ":" + m.commandInput
	}

	help := "q quit | j/k move | :auth login"
	if m.session == nil {
		help = "q quit | j/k move | :auth login | :signup register"
	}
	if m.authInProgress {
		help = "authorizing with Keycloak..."
	}

	if m.status == "" {
		return help
	}

	return fmt.Sprintf("%s | %s", m.status, help)
}

func (m model) profileLabel() string {
	if m.session == nil {
		return ""
	}

	return m.session.DisplayName()
}

func loadSessionCmd(authService *auth.Service) tea.Cmd {
	return func() tea.Msg {
		if authService == nil {
			return authLoadedMsg{}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		session, err := authService.LoadSession(ctx)
		return authLoadedMsg{
			session: session,
			err:     err,
		}
	}
}

func authenticateCmd(authService *auth.Service, action authAction) tea.Cmd {
	return func() tea.Msg {
		if authService == nil {
			return authCompletedMsg{err: fmt.Errorf("auth service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		var (
			session *auth.Session
			err     error
		)

		switch action {
		case authActionSignup:
			session, err = authService.Register(ctx)
		default:
			session, err = authService.Authenticate(ctx)
		}

		return authCompletedMsg{
			session: session,
			err:     err,
		}
	}
}
