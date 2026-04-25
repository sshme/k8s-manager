package model

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/market"

	tea "github.com/charmbracelet/bubbletea"
)

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
	fields := strings.Fields(command)
	switch {
	case command == "":
		m.commandMode = false
		m.status = "Command mode"
		return m, nil
	case len(fields) > 0 && fields[0] == "plugin":
		m.commandMode = false
		return m.runPluginCommand(fields[1:])
	case len(fields) > 0 && fields[0] == "plugins":
		m.commandMode = false
		cmd := m.loadPlugins()
		return m, cmd
	case len(fields) > 0 && fields[0] == "install":
		m.commandMode = false
		return m.runPluginCommand(fields)
	case len(fields) > 0 && fields[0] == "uninstall":
		m.commandMode = false
		return m.runPluginCommand(fields)
	case len(fields) > 0 && fields[0] == "upload-artifact":
		m.commandMode = false
		args := append([]string{"upload"}, fields[1:]...)
		return m.runPluginCommand(args)
	}

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

func (m model) runPluginCommand(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.status = "Usage: :plugin install <id> | :plugin uninstall <id> | :plugin upload <release-id> <zip-path> [os] [arch]"
		return m, nil
	}

	switch args[0] {
	case "refresh":
		cmd := m.loadPlugins()
		return m, cmd
	case "install":
		if len(args) < 2 {
			m.status = "Usage: :plugin install <id>"
			return m, nil
		}
		pluginID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || pluginID <= 0 {
			m.status = "Plugin id must be a positive number"
			return m, nil
		}
		m.status = fmt.Sprintf("Installing plugin #%d...", pluginID)
		return m, installPluginCmd(m.marketService, pluginID)
	case "uninstall":
		if len(args) < 2 {
			m.status = "Usage: :plugin uninstall <id>"
			return m, nil
		}
		pluginID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || pluginID <= 0 {
			m.status = "Plugin id must be a positive number"
			return m, nil
		}
		m.status = fmt.Sprintf("Uninstalling plugin #%d...", pluginID)
		return m, uninstallPluginCmd(m.marketService, pluginID)
	case "upload":
		if len(args) < 3 {
			m.status = "Usage: :plugin upload <release-id> <zip-path> [os] [arch]"
			return m, nil
		}
		releaseID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || releaseID <= 0 {
			m.status = "Release id must be a positive number"
			return m, nil
		}

		osName := ""
		arch := ""
		if len(args) > 3 {
			osName = args[3]
		}
		if len(args) > 4 {
			arch = args[4]
		}

		m.status = fmt.Sprintf("Uploading artifact for release #%d...", releaseID)
		return m, uploadArtifactCmd(m.marketService, releaseID, args[2], osName, arch)
	default:
		m.status = fmt.Sprintf("Unknown plugin command: %s", args[0])
		return m, nil
	}
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

func loadPluginsCmd(marketService *market.Service, query string) tea.Cmd {
	return func() tea.Msg {
		if marketService == nil {
			return pluginsLoadedMsg{err: fmt.Errorf("market service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		list, err := marketService.ListPlugins(ctx, query)
		return pluginsLoadedMsg{
			list: list,
			err:  err,
		}
	}
}

func installPluginCmd(marketService *market.Service, pluginID int64) tea.Cmd {
	return func() tea.Msg {
		if marketService == nil {
			return pluginActionCompletedMsg{err: fmt.Errorf("market service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		plugin, err := marketService.InstallPlugin(ctx, pluginID)
		if err != nil {
			return pluginActionCompletedMsg{err: err}
		}

		return pluginActionCompletedMsg{
			message: fmt.Sprintf("Installed plugin #%d %s", plugin.ID, plugin.Name),
			reload:  true,
		}
	}
}

func uninstallPluginCmd(marketService *market.Service, pluginID int64) tea.Cmd {
	return func() tea.Msg {
		if marketService == nil {
			return pluginActionCompletedMsg{err: fmt.Errorf("market service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := marketService.UninstallPlugin(ctx, pluginID); err != nil {
			return pluginActionCompletedMsg{err: err}
		}

		return pluginActionCompletedMsg{
			message: fmt.Sprintf("Uninstalled plugin #%d", pluginID),
			reload:  true,
		}
	}
}

func uploadArtifactCmd(marketService *market.Service, releaseID int64, zipPath, osName, arch string) tea.Cmd {
	return func() tea.Msg {
		if marketService == nil {
			return artifactUploadedMsg{err: fmt.Errorf("market service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		artifact, err := marketService.UploadArtifact(ctx, releaseID, zipPath, osName, arch)
		return artifactUploadedMsg{
			artifact: artifact,
			err:      err,
		}
	}
}

func createDeveloperPluginCmd(marketService *market.Service, draft market.DeveloperPluginDraft) tea.Cmd {
	return func() tea.Msg {
		if marketService == nil {
			return developerPluginCreatedMsg{err: fmt.Errorf("market service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		plugin, err := marketService.CreateDeveloperPlugin(ctx, draft)
		return developerPluginCreatedMsg{
			plugin: plugin,
			err:    err,
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
