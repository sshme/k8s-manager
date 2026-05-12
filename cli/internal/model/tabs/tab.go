package tabs

import (
	"k8s-manager/cli/internal/auth"

	tea "github.com/charmbracelet/bubbletea"
)

// Tab - одна рабочая панель в TUI. Главная модель хранит срез Tab
// и роутит Bubble Tea сообщения в активную вкладку.
type Tab interface {
	Title() string
	Init() tea.Cmd
	Update(msg tea.Msg) (Tab, tea.Cmd)
	View(width, height int) string
	Help() string
	// CapturingInput сообщает, захватывает ли таб ввод (например, строка поиска).
	// Если true, главная модель оставляет себе только ctrl+c и tab/shift+tab.
	CapturingInput() bool
}

// StatusMsg обновляет глобальную строку статуса. Табы возвращают SetStatus
// как Cmd. Главная модель перехватывает StatusMsg и пишет в свой статус бар.
type StatusMsg string

// SetStatus - вспомогательный Cmd, который эмитирует StatusMsg.
func SetStatus(text string) tea.Cmd {
	return func() tea.Msg { return StatusMsg(text) }
}

// SessionChangedMsg рассылается всем табам при изменении состояния авторизации.
// Session равен nil, если пользователь разлогинен.
type SessionChangedMsg struct {
	Session *auth.Session
}

// TabActivatedMsg приходит в таб, когда он становится активным, чтобы он
// мог лениво обновить данные.
type TabActivatedMsg struct{}
