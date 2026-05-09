package model

import (
	"fmt"
	"strings"

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
	m.commandMode = false

	fields := strings.Fields(command)
	if len(fields) == 0 {
		m.status = "Command mode"
		return m, nil
	}

	switch fields[0] {
	case "auth":
		return m, m.profile.RunLogin()
	case "signup", "register":
		return m, m.profile.RunSignup()
	case "logout":
		return m, m.profile.RunLogout()
	case "plugin":
		return m, m.plugins.RunCommand(fields[1:])
	case "plugins":
		return m, m.plugins.Reload()
	case "install", "uninstall":
		return m, m.plugins.RunCommand(fields)
	case "upload-artifact":
		return m, m.plugins.RunCommand(append([]string{"upload"}, fields[1:]...))
	}

	m.status = fmt.Sprintf("Unknown command: %s", fields[0])
	return m, nil
}
