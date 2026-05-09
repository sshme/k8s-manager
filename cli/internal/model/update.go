package model

import (
	"k8s-manager/cli/internal/model/tabs"
	"k8s-manager/cli/internal/model/tabs/profile"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tabs.StatusMsg:
		m.status = string(msg)
		return m, nil

	case tabs.SessionChangedMsg:
		// Оповещаем все вкладки об изменении сессии
		return m, m.broadcast(msg)

	case profile.Msg:
		// SessionLoaded, AuthCompleted, LogoutCompleted) всегда
		// идут в ProfileTab, независимо от того, какая вкладка сейчас активна.
		return m.delegateToProfile(msg)

	case tea.KeyMsg:
		if m.commandMode {
			return m.updateCommandMode(msg)
		}
		return m.handleKey(msg)
	}

	// Остальные сообщения направляются в активную вкладку
	return m.delegateToActive(msg)
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	active := m.tabs[m.active]

	// ctrl+c выходит всегда, в любом режиме.
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if msg.String() == "tab" || msg.String() == "shift+tab" {
		return m.updateTabNavigation(msg)
	}

	// Остальные клавиши работают только если активная вкладка не
	// захватывает ввод, иначе строка поиска не сможет принимать 'q' как символ
	if !active.CapturingInput() {
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case ":":
			if m.profile.AuthInProgress() {
				m.status = "Authentication is already in progress"
				return m, nil
			}
			m.commandMode = true
			m.commandInput = ""
			m.status = "Command mode"
			return m, nil
		}
	}

	return m.delegateToActive(msg)
}

func (m model) delegateToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	newTab, cmd := m.tabs[m.active].Update(msg)
	m.tabs[m.active] = newTab
	return m, cmd
}

// delegateToProfile отправляет сообщение конкретно в ProfileTab, вне
// зависимости от текущей активной вкладки.
func (m model) delegateToProfile(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.profile.Update(msg)
	return m, cmd
}

func (m model) updateTabNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.tabs) == 0 {
		return m, nil
	}

	if msg.String() == "shift+tab" {
		m.active--
		if m.active < 0 {
			m.active = len(m.tabs) - 1
		}
	} else {
		m.active = (m.active + 1) % len(m.tabs)
	}

	return m, sendTabActivated()
}

// broadcast доставляет msg каждому табу и собирает результирующие Cmd в батч
func (m *model) broadcast(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.tabs))
	for i := range m.tabs {
		newTab, cmd := m.tabs[i].Update(msg)
		m.tabs[i] = newTab
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func sendTabActivated() tea.Cmd {
	return func() tea.Msg { return tabs.TabActivatedMsg{} }
}
