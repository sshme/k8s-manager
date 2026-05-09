package profile

import (
	"errors"
	"fmt"

	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/model/tabs"

	tea "github.com/charmbracelet/bubbletea"
)

func (t *ProfileTab) handleSessionLoaded(msg SessionLoadedMsg) tea.Cmd {
	if msg.Err != nil && msg.Session == nil {
		// ErrSessionNotFound - ожидаемый случай первого запуска или ранее
		// разлогиненного пользователя. Остальные ошибки реальные, их
		// показываем в статус-баре
		if isSessionNotFound(msg.Err) {
			return tabs.SetStatus("Press :auth to sign in or :signup to create an account")
		}
		return tabs.SetStatus(fmt.Sprintf("Failed to restore session: %v", msg.Err))
	}

	if msg.Session == nil {
		return tabs.SetStatus("Press :auth to sign in or :signup to create an account")
	}

	t.session = msg.Session
	t.cursor = 0
	return tea.Batch(
		tabs.SetStatus(fmt.Sprintf("Authorized as %s", msg.Session.DisplayName())),
		broadcastSession(msg.Session),
	)
}

func (t *ProfileTab) handleAuthCompleted(msg AuthCompletedMsg) tea.Cmd {
	t.authInProgress = false

	if msg.Err != nil {
		return tabs.SetStatus(fmt.Sprintf("Authentication failed: %v", msg.Err))
	}
	if msg.Session == nil {
		return tabs.SetStatus("Authentication failed: empty session")
	}

	t.session = msg.Session
	t.cursor = 0
	return tea.Batch(
		tabs.SetStatus(fmt.Sprintf("Authenticated as %s", msg.Session.DisplayName())),
		broadcastSession(msg.Session),
	)
}

func (t *ProfileTab) handleLogoutCompleted(msg LogoutCompletedMsg) tea.Cmd {
	if msg.Err != nil {
		// Локальное состояние уже очищено, но отзыв токена на
		// сервере не прошёл. С точки зрения CLI пользователь разлогинен,
		// поэтому сессию всё равно сбрасываем и показываем предупреждение
		var revokeFailed *auth.RevocationFailedError
		if errors.As(msg.Err, &revokeFailed) {
			t.session = nil
			t.cursor = 0
			return tea.Batch(
				tabs.SetStatus(fmt.Sprintf(
					"Logged out locally - %v (refresh token still valid on Keycloak)",
					revokeFailed.Err)),
				broadcastSession(nil),
			)
		}

		// упала сама локальная очистка
		return tabs.SetStatus(fmt.Sprintf("Logout failed: %v", msg.Err))
	}

	t.session = nil
	t.cursor = 0
	return tea.Batch(
		tabs.SetStatus("Logged out"),
		broadcastSession(nil),
	)
}

// broadcastSession просит main разослать новое состояние сессии всем табам
// (чтобы Plugins перечитался, шапка обновилась и тд)
func broadcastSession(s *auth.Session) tea.Cmd {
	return func() tea.Msg { return tabs.SessionChangedMsg{Session: s} }
}

// isSessionNotFound сообщает, обёрнут ли auth.ErrSessionNotFound внутри err
func isSessionNotFound(err error) bool {
	return errors.Is(err, auth.ErrSessionNotFound)
}
