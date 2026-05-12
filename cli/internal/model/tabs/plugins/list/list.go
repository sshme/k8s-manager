// Package list - главный экран вкладки Plugins
package list

import (
	"fmt"

	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/tabs"
	"k8s-manager/cli/internal/model/tabs/plugins/shared"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// focusZone - какая часть экрана сейчас принимает клавиши (поиск или список плагинов).
type focusZone int

const (
	focusList focusZone = iota
	focusSearch
)

// Model - состояние главного экрана
type Model struct {
	market  *market.Service
	session *auth.Session

	search       textinput.Model
	viewport     viewport.Model
	viewportInit bool

	focus      focusZone
	listCursor int

	items   []market.PluginSummary
	total   int64
	source  string // "installed" или "search"
	loading bool
	loaded  bool
}

// New создаёт модель и настраивает textinput.
func New(svc *market.Service) Model {
	ti := textinput.New()
	ti.Placeholder = "Search plugins..."
	ti.CharLimit = 128
	ti.Prompt = ""

	return Model{
		market:   svc,
		search:   ti,
		viewport: viewport.New(0, 0),
		focus:    focusList,
	}
}

// SetSession обновляет сессию и (если нужно) триггерит первую загрузку
func (m Model) SetSession(s *auth.Session) (Model, tea.Cmd) {
	m.session = s
	if s != nil && !m.loaded && !m.loading {
		return m.startLoad()
	}
	return m, nil
}

// OnTabActivated вызывается роутером при переключении на эту вкладку.
// Если список ещё не был загружен и пользователь залогинен, то подгружаем список
func (m Model) OnTabActivated() (Model, tea.Cmd) {
	if m.session != nil && !m.loaded && !m.loading {
		return m.startLoad()
	}
	return m, nil
}

// Reload вызывает перезагрузку списка с текущим поисковым запросом
func (m Model) Reload() (Model, tea.Cmd) {
	if m.session == nil {
		return m, tabs.SetStatus("Sign in to load plugins")
	}
	return m.startLoad()
}

// startLoad помечает state как loading и возвращает Cmd для подгрузки списка плагинов
func (m Model) startLoad() (Model, tea.Cmd) {
	m.loading = true
	return m, loadCmd(m.market, m.search.Value())
}

// CapturingInput сообщает, что поисковая строка принимает символы
func (m Model) CapturingInput() bool {
	return m.focus == focusSearch
}

// MarkPluginInstalled помечает плагин как установленный (добавленный библиотеку)
func (m Model) MarkPluginInstalled(pluginID int64, installed bool) Model {
	for i := range m.items {
		if m.items[i].ID == pluginID {
			m.items[i].Installed = installed
			break
		}
	}
	return m.refresh()
}

// Update реагирует на входящие события для обновления модели
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case loadedMsg:
		return m.handleLoaded(msg)
	}
	return m, nil
}

// handleLoaded обрабатывает ответ сервера на поисковый запрос
func (m Model) handleLoaded(msg loadedMsg) (Model, tea.Cmd) {
	m.loading = false
	m.loaded = true

	if msg.err != nil {
		return m, tabs.SetStatus(fmt.Sprintf("Failed to load plugins: %v", msg.err))
	}
	if msg.list == nil {
		return m, tabs.SetStatus("Failed to load plugins: empty response")
	}

	m.items = msg.list.Items
	m.total = msg.list.Total
	if m.listCursor >= len(m.items) {
		if len(m.items) == 0 {
			m.listCursor = 0
		} else {
			m.listCursor = len(m.items) - 1
		}
	}

	if msg.list.Installed {
		m.source = "installed"
		m = m.refresh()
		return m, tabs.SetStatus(fmt.Sprintf("Loaded %d installed plugins", len(m.items)))
	}
	m.source = "search"
	m = m.refresh()
	return m, tabs.SetStatus(fmt.Sprintf("Found %d plugins", len(m.items)))
}

// refreshAction - шорткат, перезагрузка списка
func (m Model) refreshAction() (Model, tea.Cmd) {
	return m.startLoad()
}

// createAction - открыть wizard
func (m Model) createAction() (Model, tea.Cmd) {
	return m, func() tea.Msg { return shared.OpenWizardMsg{} }
}

// openDetail формирует Cmd с сигналом роутеру для открытия детальной страницы
// для текущего выбранного плагина
func (m Model) openDetail() tea.Cmd {
	if len(m.items) == 0 {
		return tabs.SetStatus("Select a plugin first")
	}
	plugin := m.items[m.listCursor]
	return func() tea.Msg {
		return shared.OpenDetailMsg{Summary: plugin}
	}
}
