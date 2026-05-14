package plugins

import (
	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/tabs"
	"k8s-manager/cli/internal/model/tabs/plugins/detail"
	"k8s-manager/cli/internal/model/tabs/plugins/list"
	"k8s-manager/cli/internal/model/tabs/plugins/shared"
	"k8s-manager/cli/internal/model/tabs/plugins/wizard"
	pluginsmgr "k8s-manager/cli/internal/plugins"

	tea "github.com/charmbracelet/bubbletea"
)

// Msg - интерфейс для сообщений вкладки Plugins
type Msg = shared.Msg

// screenKind - какой экран сейчас видит пользователь. Wizard отрисовывается
// поверх list или detail и имеет приоритет, поэтому отдельным screen не считается
type screenKind int

const (
	screenList screenKind = iota
	screenDetail
)

// Tab - модель вкладки с плагинами
type Tab struct {
	market  *market.Service
	plugins *pluginsmgr.Manager
	session *auth.Session

	list   list.Model
	detail detail.Model
	wizard wizard.Model

	screen        screenKind
	width, height int
}

// New создаёт вкладку с инициализированным list-экраном
func New(svc *market.Service, mgr *pluginsmgr.Manager) *Tab {
	return &Tab{
		market:  svc,
		plugins: mgr,
		list:    list.New(svc),
		wizard:  wizard.New(svc),
		screen:  screenList,
	}
}

func (t *Tab) Title() string { return "Plugins" }

func (t *Tab) Init() tea.Cmd { return nil }

// CapturingInput - true когда поиск или wizard ловят ввод. Тогда главная
// модель не пускает шорткаты вроде 'q'
func (t *Tab) CapturingInput() bool {
	if t.wizard.Active() {
		return true
	}
	if t.screen == screenList {
		return t.list.CapturingInput()
	}
	return t.detail.CapturingInput()
}

// Help - подсказка по клавишам для текущего экрана
func (t *Tab) Help() string {
	if t.wizard.Active() {
		return "esc cancel | up/down input/buttons | left/right buttons | enter next/create"
	}
	if t.screen == screenDetail {
		return "q quit | esc back | ↑/↓ focus zone | ←/→ change/buttons | enter activate"
	}
	return "q quit | / search | ↑/↓ list | enter open | r refresh | c create"
}

// Reload - точка входа для command-mode :plugins
func (t *Tab) Reload() tea.Cmd {
	if t.screen != screenList {
		return tabs.SetStatus("Reload works only on the list screen")
	}
	next, cmd := t.list.Reload()
	t.list = next
	return cmd
}

// Update раскидывает по экранам и обрабатывает tea.Msg
func (t *Tab) Update(msg tea.Msg) (tabs.Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		t.list = t.list.SetSize(t.width, t.height)
		t.detail = t.detail.SetSize(t.width, t.height)
		return t, nil

	case tabs.SessionChangedMsg:
		t.session = msg.Session
		next, cmd := t.list.SetSession(msg.Session)
		t.list = next
		if t.screen == screenDetail {
			t.detail = t.detail.SetSession(msg.Session)
		}
		return t, cmd

	case tabs.TabActivatedMsg:
		next, cmd := t.list.OnTabActivated()
		t.list = next
		return t, cmd

	case shared.OpenDetailMsg:
		t.screen = screenDetail
		d, cmd := detail.New(t.market, t.plugins, msg.Summary, t.session)
		t.detail = d.SetSize(t.width, t.height)
		return t, cmd

	case shared.BackToListMsg:
		t.screen = screenList
		return t, tabs.SetStatus("Back to list")

	case shared.LibraryChangedMsg:
		// Синхронизируем флаг Installed в списке без RPC
		t.list = t.list.MarkPluginInstalled(msg.PluginID, msg.Installed)
		return t, nil

	case shared.OpenWizardMsg:
		t.wizard = t.wizard.Open()
		return t, tabs.SetStatus("Plugin creation wizard")

	case wizard.CreatedMsg:
		next, cmd := t.wizard.Update(msg)
		t.wizard = next
		// После успешного создания обновляем список, чтобы юзер увидел новый плагин
		if msg.Err == nil && msg.Plugin != nil {
			reloaded, reloadCmd := t.list.Reload()
			t.list = reloaded
			return t, tea.Batch(cmd, reloadCmd)
		}
		return t, cmd
	}

	// Клавиши и остальные сообщения отдаём активному экрану
	return t.delegate(msg)
}

// delegate перенаправляет msg в активный экран. wizard поверх всего,
// иначе detail или list
func (t *Tab) delegate(msg tea.Msg) (tabs.Tab, tea.Cmd) {
	if t.wizard.Active() {
		next, cmd := t.wizard.Update(msg)
		t.wizard = next
		return t, cmd
	}

	if t.screen == screenDetail {
		next, cmd := t.detail.Update(msg)
		t.detail = next
		return t, cmd
	}

	next, cmd := t.list.Update(msg)
	t.list = next
	return t, cmd
}

// View рисует активный экран. Wizard, если открыт, поверх всего
func (t *Tab) View(width, height int) string {
	if t.wizard.Active() {
		return t.wizard.View(width, height)
	}
	if t.screen == screenDetail {
		return t.detail.View(width, height)
	}
	return t.list.View(width, height)
}
