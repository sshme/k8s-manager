package profile

import (
	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/model/tabs"

	tea "github.com/charmbracelet/bubbletea"
)

// ProfileTab - панель аутентификации в TUI. Хранит auth.Service, текущую
// сессию, флаг идущего OIDC-флоу и позицию курсора по кнопкам
type ProfileTab struct {
	authService    *auth.Service
	session        *auth.Session
	authInProgress bool
	cursor         int
}

func New(authService *auth.Service) *ProfileTab {
	return &ProfileTab{authService: authService}
}

func (*ProfileTab) Title() string { return "Profile" }

// Init восстановление сессии с диска. Результат прилетит
// обратно как SessionLoadedMsg, который main роутит сюда же.
func (t *ProfileTab) Init() tea.Cmd {
	return loadSessionCmd(t.authService)
}

func (t *ProfileTab) Update(msg tea.Msg) (tabs.Tab, tea.Cmd) {
	switch msg := msg.(type) {

	case SessionLoadedMsg:
		return t, t.handleSessionLoaded(msg)

	case AuthCompletedMsg:
		return t, t.handleAuthCompleted(msg)

	case LogoutCompletedMsg:
		return t, t.handleLogoutCompleted(msg)

	case tabs.SessionChangedMsg:
		// SessionChangedMsg рождается тут же, и когда main его броадкастит,
		// t.session уже совпадает. На всякий случай всё же
		// присваиваем (вдруг сессию поменяют снаружи) и сбрасываем курсор,
		// потому что набор кнопок мог измениться
		if t.session != msg.Session {
			t.session = msg.Session
			t.cursor = 0
		}
		return t, nil

	case tea.KeyMsg:
		return t.handleKey(msg)
	}

	return t, nil
}

func (t *ProfileTab) Help() string {
	if t.authInProgress {
		return "authorizing with Keycloak..."
	}
	if t.session == nil {
		return "q quit | tab tabs | ←/→ select | enter activate | :auth :signup"
	}
	return "q quit | tab tabs | ←/→ select | enter activate | :logout"
}

// CapturingInput всегда false: на профиле нет текстовых полей, так что
// глобальные шорткаты вроде 'q' и ':' должны продолжать работать
func (*ProfileTab) CapturingInput() bool { return false }

// Роутинг command-mode и шапка

// Session возвращает текущую сессию (nil, если разлогинен). Используется
// шапкой главного view для бейджа
func (t *ProfileTab) Session() *auth.Session { return t.session }

// AuthInProgress говорит, идёт ли сейчас OIDC-флоу
func (t *ProfileTab) AuthInProgress() bool { return t.authInProgress }

// RunLogin - то, во что разворачивается :auth
func (t *ProfileTab) RunLogin() tea.Cmd { return t.login() }

// RunSignup - то, во что разворачиваются :signup / :register
func (t *ProfileTab) RunSignup() tea.Cmd { return t.signup() }

// RunLogout - то, во что разворачивается :logout
func (t *ProfileTab) RunLogout() tea.Cmd { return t.logout() }
