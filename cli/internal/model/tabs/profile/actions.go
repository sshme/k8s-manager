package profile

import (
	"k8s-manager/cli/internal/model/tabs"

	tea "github.com/charmbracelet/bubbletea"
)

// action - одна кнопка на табе. do возвращает Cmd, который запустится при
// её нажатии
type action struct {
	label string
	do    func(t *ProfileTab) tea.Cmd
}

// availableActions возвращает кнопки, актуальные для текущего состояния сессии
func (t *ProfileTab) availableActions() []action {
	if t.session == nil {
		return []action{
			{label: "Log in", do: (*ProfileTab).login},
			{label: "Sign up", do: (*ProfileTab).signup},
		}
	}
	return []action{
		{label: "Log out", do: (*ProfileTab).logout},
	}
}

func (t *ProfileTab) login() tea.Cmd {
	if t.authInProgress {
		return tabs.SetStatus("Authentication is already running")
	}
	if t.session != nil {
		return tabs.SetStatus("Already signed in - log out first")
	}
	t.authInProgress = true
	return tea.Batch(
		tabs.SetStatus("Opening Keycloak in browser..."),
		authenticateCmd(t.authService, authLogin),
	)
}

func (t *ProfileTab) signup() tea.Cmd {
	if t.authInProgress {
		return tabs.SetStatus("Authentication is already running")
	}
	if t.session != nil {
		return tabs.SetStatus("Already signed in - log out first")
	}
	t.authInProgress = true
	return tea.Batch(
		tabs.SetStatus("Opening Keycloak registration in browser..."),
		authenticateCmd(t.authService, authSignup),
	)
}

func (t *ProfileTab) logout() tea.Cmd {
	if t.authInProgress {
		return tabs.SetStatus("Authentication is in progress, try again later")
	}
	if t.session == nil {
		return tabs.SetStatus("Not signed in")
	}
	return tea.Batch(
		tabs.SetStatus("Logging out..."),
		logoutCmd(t.authService),
	)
}

func (t *ProfileTab) handleKey(msg tea.KeyMsg) (tabs.Tab, tea.Cmd) {
	if t.authInProgress {
		// Пока идёт OIDC-флоу, ввод игнорируем
		return t, nil
	}

	actions := t.availableActions()
	if len(actions) == 0 {
		return t, nil
	}

	switch msg.String() {
	case "left", "h":
		t.cursor--
		if t.cursor < 0 {
			t.cursor = len(actions) - 1
		}
	case "right", "l":
		t.cursor = (t.cursor + 1) % len(actions)
	case "enter":
		// набор кнопок мог только что стать меньше
		if t.cursor >= len(actions) {
			t.cursor = 0
		}
		return t, actions[t.cursor].do(t)
	}
	return t, nil
}
