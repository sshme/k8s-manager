package model

import (
	"fmt"

	"k8s-manager/cli/internal/auth"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case authLoadedMsg:
		if msg.err == nil && msg.session != nil {
			m.session = msg.session
			m.status = fmt.Sprintf("Authorized as %s", msg.session.DisplayName())
			if m.selectedTab() == "Plugins" {
				cmd := m.loadPlugins()
				return m, cmd
			}
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
		if m.selectedTab() == "Plugins" {
			cmd := m.loadPlugins()
			return m, cmd
		}
	case tea.KeyMsg:
		if m.commandMode {
			return m.updateCommandMode(msg)
		}

		if msg.String() == "tab" || msg.String() == "shift+tab" {
			next, cmd := m.updateTabNavigation(msg)
			return next, cmd
		}

		if m.selectedTab() == "Plugins" {
			if next, cmd, handled := m.updatePluginsKey(msg); handled {
				return next, cmd
			}
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
			if m.selectedTab() == "Plugins" && !m.pluginLoaded && !m.pluginLoading {
				cmd := m.loadPlugins()
				return m, cmd
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
			if m.selectedTab() == "Plugins" && !m.pluginLoaded && !m.pluginLoading {
				cmd := m.loadPlugins()
				return m, cmd
			}
		}
	case pluginsLoadedMsg:
		m.pluginLoading = false
		m.pluginLoaded = true
		if msg.err != nil {
			m.status = fmt.Sprintf("Failed to load plugins: %v", msg.err)
			return m, nil
		}
		if msg.list == nil {
			m.status = "Failed to load plugins: empty response"
			return m, nil
		}

		m.pluginItems = msg.list.Items
		m.pluginTotal = msg.list.Total
		if m.pluginListCursor >= len(m.pluginItems) {
			m.pluginListCursor = max(0, len(m.pluginItems)-1)
		}
		if msg.list.Installed {
			m.pluginSource = "installed"
			m.status = fmt.Sprintf("Loaded %d installed plugins", len(msg.list.Items))
		} else {
			m.pluginSource = "search"
			m.status = fmt.Sprintf("Found %d plugins", len(msg.list.Items))
		}
	case pluginActionCompletedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Plugin action failed: %v", msg.err)
			return m, nil
		}
		m.status = msg.message
		if msg.reload {
			cmd := m.loadPlugins()
			return m, cmd
		}
	case artifactUploadedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Artifact upload failed: %v", msg.err)
			return m, nil
		}
		if msg.artifact == nil {
			m.status = "Artifact upload failed: empty response"
			return m, nil
		}
		m.status = fmt.Sprintf("Artifact #%d uploaded (%d bytes, sha256 %s)", msg.artifact.ID, msg.artifact.Size, shortChecksum(msg.artifact.Checksum))
	case developerPluginCreatedMsg:
		m.pluginWizard.submitting = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Plugin creation failed: %v", msg.err)
			return m, nil
		}
		if msg.plugin == nil {
			m.status = "Plugin creation failed: empty response"
			return m, nil
		}
		m.pluginWizard = pluginWizard{}
		m.pluginSearch = msg.plugin.Name
		m.status = fmt.Sprintf("Created plugin #%d release #%d artifact #%d", msg.plugin.PluginID, msg.plugin.ReleaseID, msg.plugin.ArtifactID)
		cmd := m.loadPlugins()
		return m, cmd
	}
	return m, nil
}

func (m model) updateTabNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.items) == 0 {
		return m, nil
	}

	switch msg.String() {
	case "shift+tab":
		m.cursor--
		if m.cursor < 0 {
			m.cursor = len(m.items) - 1
		}
	default:
		m.cursor = (m.cursor + 1) % len(m.items)
	}

	if m.selectedTab() == "Plugins" && !m.pluginLoaded && !m.pluginLoading {
		cmd := m.loadPlugins()
		return m, cmd
	}

	return m, nil
}
