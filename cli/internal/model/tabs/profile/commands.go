package profile

import (
	"context"
	"fmt"
	"time"

	"k8s-manager/cli/internal/auth"

	tea "github.com/charmbracelet/bubbletea"
)

// Msg - интерфейс, который говорит о том, что сообщение принадлежит ProfileTab.
// Главная модель по нему понимает, что результат auth-команды надо вернуть в ProfileTab,
// какой бы таб ни был сейчас активен
type Msg interface {
	isProfileMsg()
}

// SessionLoadedMsg прилетает один раз на старте и содержит сессию, найденную
// на диске.
type SessionLoadedMsg struct {
	Session *auth.Session
	Err     error
}

func (SessionLoadedMsg) isProfileMsg() {}

// AuthCompletedMsg - результат интерактивного Login или Signup.
type AuthCompletedMsg struct {
	Session *auth.Session
	Err     error
}

func (AuthCompletedMsg) isProfileMsg() {}

// LogoutCompletedMsg - результат действия Logout.
type LogoutCompletedMsg struct {
	Err error
}

func (LogoutCompletedMsg) isProfileMsg() {}

// authKind выбирает, какой именно OIDC-флоу запустить.
type authKind int

const (
	authLogin authKind = iota
	authSignup
)

func loadSessionCmd(svc *auth.Service) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return SessionLoadedMsg{}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		session, err := svc.LoadSession(ctx)
		return SessionLoadedMsg{Session: session, Err: err}
	}
}

func authenticateCmd(svc *auth.Service, kind authKind) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return AuthCompletedMsg{Err: fmt.Errorf("auth service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		var (
			session *auth.Session
			err     error
		)

		switch kind {
		case authSignup:
			session, err = svc.Register(ctx)
		default:
			session, err = svc.Authenticate(ctx)
		}

		return AuthCompletedMsg{Session: session, Err: err}
	}
}

func logoutCmd(svc *auth.Service) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return LogoutCompletedMsg{}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return LogoutCompletedMsg{Err: svc.Logout(ctx)}
	}
}
