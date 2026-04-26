package model

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/market"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) updatePluginsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.pluginWizard.open {
		next, cmd := m.updatePluginWizardKey(msg)
		return next, cmd, true
	}

	if m.pluginFocused {
		switch msg.String() {
		case "esc":
			m.pluginFocused = false
			m.status = "Plugin search blurred"
			return m, nil, true
		case "enter":
			m.pluginFocused = false
			cmd := m.loadPlugins()
			return m, cmd, true
		case "ctrl+u":
			m.pluginSearch = ""
			cmd := m.loadPlugins()
			return m, cmd, true
		case "backspace", "delete":
			runes := []rune(m.pluginSearch)
			if len(runes) > 0 {
				m.pluginSearch = string(runes[:len(runes)-1])
			}
			cmd := m.loadPlugins()
			return m, cmd, true
		}

		if len(msg.Runes) > 0 {
			m.pluginSearch += string(msg.Runes)
			cmd := m.loadPlugins()
			return m, cmd, true
		}

		return m, nil, true
	}

	switch msg.String() {
	case "/":
		m.pluginFocused = true
		m.status = "Plugin search"
		return m, nil, true
	case "enter":
		next, cmd := m.activatePluginButton()
		return next, cmd, true
	case "ctrl+u":
		m.pluginSearch = ""
		m.pluginFocused = true
		cmd := m.loadPlugins()
		return m, cmd, true
	case "right", "l":
		m.pluginButton = (m.pluginButton + 1) % len(pluginButtons())
		return m, nil, true
	case "left", "h":
		m.pluginButton--
		if m.pluginButton < 0 {
			m.pluginButton = len(pluginButtons()) - 1
		}
		return m, nil, true
	case "up", "k":
		if m.pluginListCursor > 0 {
			m.pluginListCursor--
		}
		return m, nil, true
	case "down", "j":
		if m.pluginListCursor < len(m.pluginItems)-1 {
			m.pluginListCursor++
		}
		return m, nil, true
	}

	return m, nil, false
}

func (m model) activatePluginButton() (tea.Model, tea.Cmd) {
	switch pluginButtons()[m.pluginButton] {
	case "Refresh":
		cmd := m.loadPlugins()
		return m, cmd
	case "Create":
		m.pluginWizard = newPluginWizard()
		m.status = "Plugin creation wizard"
		return m, nil
	case "Install":
		if len(m.pluginItems) == 0 {
			m.status = "Select a plugin to install"
			return m, nil
		}
		pluginID := m.pluginItems[m.pluginListCursor].ID
		m.status = fmt.Sprintf("Installing plugin #%d...", pluginID)
		return m, installPluginCmd(m.marketService, pluginID)
	case "Uninstall":
		if len(m.pluginItems) == 0 {
			m.status = "Select a plugin to uninstall"
			return m, nil
		}
		pluginID := m.pluginItems[m.pluginListCursor].ID
		m.status = fmt.Sprintf("Uninstalling plugin #%d...", pluginID)
		return m, uninstallPluginCmd(m.marketService, pluginID)
	default:
		return m, nil
	}
}

func (m model) renderContent(selected string) []string {
	if selected == "Plugins" {
		return m.renderPluginsContent()
	}

	return []string{
		fmt.Sprintf("Selected tab: %s", selected),
		"",
		"Main workspace.",
	}
}

func (m model) renderPluginsContent() []string {
	focus := " "
	if m.pluginFocused {
		focus = ">"
	}

	mode := "installed"
	if strings.TrimSpace(m.pluginSearch) != "" {
		mode = "search"
	}

	lines := []string{
		titleStyle.Render("Plugins"),
		"",
		m.renderPluginButtons(),
		"",
		searchStyle.Render(fmt.Sprintf("%s Search: %s", focus, m.pluginSearch)),
		subtleStyle.Render(fmt.Sprintf("Mode: %s", mode)),
		"",
	}

	if m.pluginLoading {
		return append(lines, warningStyle.Render("Loading plugins..."))
	}

	if !m.pluginLoaded {
		return append(lines, subtleStyle.Render("Press Enter to load plugins."))
	}

	if len(m.pluginItems) == 0 {
		if m.pluginSource == "installed" {
			return append(lines, subtleStyle.Render("No installed plugins."))
		}
		return append(lines, subtleStyle.Render("No plugins found."))
	}

	lines = append(lines, subtleStyle.Render(fmt.Sprintf("Total: %d", m.pluginTotal)), "")
	for i, plugin := range m.pluginItems {
		prefix := "  "
		lineStyle := normalItemStyle
		if i == m.pluginListCursor {
			prefix = "> "
			lineStyle = selectedItemStyle
		}
		title := fmt.Sprintf("#%d  %s", plugin.ID, plugin.Name)
		if plugin.Identifier != "" {
			title += fmt.Sprintf(" (%s)", plugin.Identifier)
		}
		meta := strings.Trim(strings.Join(nonEmpty(plugin.Category, plugin.TrustStatus, plugin.Status), " | "), " | ")
		if plugin.Installed && m.pluginSource == "search" {
			meta = strings.Trim(strings.Join(nonEmpty(meta, "INSTALLED"), " | "), " | ")
		}
		lines = append(lines, lineStyle.Render(prefix+title))
		if meta != "" {
			lines = append(lines, subtleStyle.Render("  "+meta))
		}
		if plugin.Description != "" {
			lines = append(lines, subtleStyle.Render("  "+truncate(plugin.Description, 96)))
		}
		if plugin.InstalledAt != "" && m.pluginSource == "search" {
			lines = append(lines, installedStyle.Render("installed")+" "+subtleStyle.Render(plugin.InstalledAt))
		}
		lines = append(lines, "")
	}

	return lines
}

func pluginButtons() []string {
	return []string{"Refresh", "Create", "Install", "Uninstall"}
}

func (m model) renderPluginButtons() string {
	buttons := pluginButtons()
	rendered := make([]string, 0, len(buttons))
	for i, button := range buttons {
		style := buttonStyle
		if i == m.pluginButton {
			style = activeButtonStyle
		}
		rendered = append(rendered, style.Render(button))
	}
	return strings.Join(rendered, " ")
}

func newPluginWizard() pluginWizard {
	return pluginWizard{
		open:  true,
		focus: wizardFocusInput,
		fields: []wizardField{
			{label: "Publisher", placeholder: "acme", required: true},
			{label: "Identifier", placeholder: "acme.example", required: true},
			{label: "Plugin name", placeholder: "Example Plugin", required: true},
			{label: "Description", placeholder: "Short description"},
			{label: "Category", placeholder: "networking"},
			{label: "Version", placeholder: "1.0.0", defaultValue: "1.0.0", required: true},
			{label: "Zip artifact", placeholder: "/path/to/plugin.zip", required: true},
			{label: "OS", placeholder: "empty = current OS"},
			{label: "Arch", placeholder: "empty = current arch"},
		},
	}
}

func (m model) updatePluginWizardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pluginWizard.submitting {
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.pluginWizard = pluginWizard{}
		m.status = "Plugin creation canceled"
		return m, nil
	case "down":
		m.pluginWizard.focus = wizardFocusButtons
		return m, nil
	case "up":
		m.pluginWizard.focus = wizardFocusInput
		return m, nil
	}

	if m.pluginWizard.focus == wizardFocusButtons {
		switch msg.String() {
		case "left", "h":
			m.pluginWizard.button--
			if m.pluginWizard.button < 0 {
				m.pluginWizard.button = len(m.wizardButtons()) - 1
			}
			return m, nil
		case "right", "l":
			m.pluginWizard.button = (m.pluginWizard.button + 1) % len(m.wizardButtons())
			return m, nil
		case "enter":
			return m.activateWizardButton()
		}
		return m, nil
	}

	switch msg.String() {
	case "enter":
		return m.advanceWizard()
	case "backspace", "delete":
		field := &m.pluginWizard.fields[m.pluginWizard.step]
		runes := []rune(field.value)
		if len(runes) > 0 {
			field.value = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	if len(msg.Runes) > 0 {
		field := &m.pluginWizard.fields[m.pluginWizard.step]
		field.value += string(msg.Runes)
	}

	return m, nil
}

func (m model) wizardButtons() []string {
	if m.pluginWizard.step == len(m.pluginWizard.fields)-1 {
		return []string{"Back", "Create", "Cancel"}
	}
	if m.pluginWizard.step == 0 {
		return []string{"Next", "Cancel"}
	}
	return []string{"Back", "Next", "Cancel"}
}

func (m model) activateWizardButton() (tea.Model, tea.Cmd) {
	switch m.wizardButtons()[m.pluginWizard.button] {
	case "Back":
		if m.pluginWizard.step > 0 {
			m.pluginWizard.step--
			m.pluginWizard.button = 0
			m.pluginWizard.focus = wizardFocusInput
		}
		return m, nil
	case "Next":
		return m.advanceWizard()
	case "Create":
		return m.submitWizard()
	case "Cancel":
		m.pluginWizard = pluginWizard{}
		m.status = "Plugin creation canceled"
		return m, nil
	default:
		return m, nil
	}
}

func (m model) advanceWizard() (tea.Model, tea.Cmd) {
	m.applyWizardDefault()
	if err := m.validateWizardStep(); err != nil {
		m.status = err.Error()
		return m, nil
	}
	if m.pluginWizard.step == len(m.pluginWizard.fields)-1 {
		return m.submitWizard()
	}
	m.pluginWizard.step++
	m.pluginWizard.button = 0
	m.pluginWizard.focus = wizardFocusInput
	return m, nil
}

func (m model) validateWizardStep() error {
	field := m.pluginWizard.fields[m.pluginWizard.step]
	if field.required && strings.TrimSpace(field.value) == "" {
		return fmt.Errorf("%s is required", field.label)
	}
	return nil
}

func (m *model) applyWizardDefault() {
	field := &m.pluginWizard.fields[m.pluginWizard.step]
	if strings.TrimSpace(field.value) == "" && field.defaultValue != "" {
		field.value = field.defaultValue
	}
}

func (m model) validateWizard() error {
	for i := range m.pluginWizard.fields {
		m.pluginWizard.step = i
		if err := m.validateWizardStep(); err != nil {
			return err
		}
	}
	return nil
}

func (m model) submitWizard() (tea.Model, tea.Cmd) {
	if err := m.validateWizard(); err != nil {
		m.status = err.Error()
		return m, nil
	}

	m.pluginWizard.submitting = true
	m.status = "Creating plugin, release, and artifact..."
	return m, createDeveloperPluginCmd(m.marketService, m.pluginWizard.draft())
}

func (w pluginWizard) draft() market.DeveloperPluginDraft {
	value := func(index int) string {
		return strings.TrimSpace(w.fields[index].value)
	}

	return market.DeveloperPluginDraft{
		PublisherName: value(wizardPublisherName),
		Identifier:    value(wizardIdentifier),
		Name:          value(wizardPluginName),
		Description:   value(wizardDescription),
		Category:      value(wizardCategory),
		Version:       value(wizardVersion),
		ZipPath:       value(wizardZipPath),
		OS:            value(wizardOS),
		Arch:          value(wizardArch),
	}
}

func (m model) renderPluginWizardModal(width, height int) string {
	modalWidth := min(max(48, width-8), 76)
	field := m.pluginWizard.fields[m.pluginWizard.step]
	inputPrefix := "  "
	if m.pluginWizard.focus == wizardFocusInput {
		inputPrefix = "> "
	}

	value := field.value
	if value == "" {
		value = subtleStyle.Render(field.placeholder)
	}

	lines := []string{
		titleStyle.Render(fmt.Sprintf("Create Plugin  %d/%d", m.pluginWizard.step+1, len(m.pluginWizard.fields))),
		"",
		searchStyle.Render(field.label),
		inputPrefix + value,
		"",
		m.renderWizardButtons(),
	}
	if m.pluginWizard.submitting {
		lines = append(lines, "", warningStyle.Render("Submitting..."))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorTeal).
		Background(colorPanel).
		Padding(1, 2).
		Width(modalWidth).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) renderWizardButtons() string {
	buttons := m.wizardButtons()
	rendered := make([]string, 0, len(buttons))
	for i, button := range buttons {
		style := buttonStyle
		if m.pluginWizard.focus == wizardFocusButtons && i == m.pluginWizard.button {
			style = activeButtonStyle
		}
		rendered = append(rendered, style.Render(button))
	}
	return strings.Join(rendered, " ")

}
