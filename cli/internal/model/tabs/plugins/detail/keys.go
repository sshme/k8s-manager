package detail

import (
	"k8s-manager/cli/internal/model/tabs/plugins/shared"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKey - роутер клавиш
func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return shared.BackToListMsg{} }
	case "up", "k":
		return m.cycleFocus(-1), nil
	case "down", "j":
		return m.cycleFocus(+1), nil
	}

	switch m.focus {
	case focusVersion:
		return m.handleVersionKey(msg)
	case focusArch:
		return m.handleArchKey(msg)
	default:
		return m.handleButtonsKey(msg)
	}
}

// cycleFocus переключает зону циклически
func (m Model) cycleFocus(dir int) Model {
	order := []focusZone{focusButtons, focusVersion, focusArch}
	idx := 0
	for i, z := range order {
		if z == m.focus {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(order)) % len(order)
	m.focus = order[idx]
	return m
}

func (m Model) handleVersionKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if len(m.releases) == 0 {
		return m, nil
	}
	switch msg.String() {
	case "left", "h":
		m.releaseIdx--
		if m.releaseIdx < 0 {
			m.releaseIdx = len(m.releases) - 1
		}
		return m.reloadArtifacts()
	case "right", "l":
		m.releaseIdx = (m.releaseIdx + 1) % len(m.releases)
		return m.reloadArtifacts()
	}
	return m, nil
}

func (m Model) handleArchKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if len(m.artifacts) == 0 {
		return m, nil
	}
	switch msg.String() {
	case "left", "h":
		m.artifactIdx--
		if m.artifactIdx < 0 {
			m.artifactIdx = len(m.artifacts) - 1
		}
		return m, nil
	case "right", "l":
		m.artifactIdx = (m.artifactIdx + 1) % len(m.artifacts)
		return m, nil
	}
	return m, nil
}

func (m Model) handleButtonsKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		m.buttonCursor--
		if m.buttonCursor < 0 {
			m.buttonCursor = 2
		}
		return m, nil
	case "right", "l":
		m.buttonCursor = (m.buttonCursor + 1) % 3
		return m, nil
	case "enter":
		return m.activateButton()
	}
	return m, nil
}

// reloadArtifacts вызывается при смене версии: артефакты у каждого релиза свои
func (m Model) reloadArtifacts() (Model, tea.Cmd) {
	m.artifacts = nil
	m.artifactIdx = 0
	m.loadingArtifacts = true
	return m, loadArtifactsCmd(m.market, m.releases[m.releaseIdx].ID)
}
