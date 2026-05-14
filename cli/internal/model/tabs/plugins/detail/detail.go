// Package detail - детальная страница одного плагина
package detail

import (
	"runtime"

	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/tabs"
	"k8s-manager/cli/internal/model/tabs/plugins/shared"
	pluginsmgr "k8s-manager/cli/internal/plugins"

	tea "github.com/charmbracelet/bubbletea"
)

// adminRole - имя роли в JWT, которая разрешает менять trust_status плагина.
// Должно совпадать с auth.RoleMarketAdmin на стороне маркета
const adminRole = "market_admin"

// focusZone - какая часть экрана сейчас принимает клавиши
type focusZone int

const (
	focusButtons focusZone = iota
	focusVersion
	focusArch
)

// Индексы и метки кнопок. Лейблы зависят от текущего состояния
// (установлен/нет, verified/нет), но количество и порядок фиксированы
const (
	btnLibrary  = 0
	btnArtifact = 1
	btnVerify   = 2
)

// Model - состояние детальной страницы
type Model struct {
	market  *market.Service
	plugins *pluginsmgr.Manager
	session *auth.Session

	summary market.PluginSummary

	plugin    *market.PluginDetails
	releases  []market.Release
	artifacts []market.Artifact

	releaseIdx  int
	artifactIdx int

	focus        focusZone
	buttonCursor int

	loadingPlugin    bool
	loadingReleases  bool
	loadingArtifacts bool
	actionInProgress bool

	width, height int
}

// New создаёт модель и возвращает Cmd для параллельной загрузки plugin/releases
func New(svc *market.Service, mgr *pluginsmgr.Manager, summary market.PluginSummary, session *auth.Session) (Model, tea.Cmd) {
	m := Model{
		market:          svc,
		plugins:         mgr,
		session:         session,
		summary:         summary,
		focus:           focusButtons,
		loadingPlugin:   true,
		loadingReleases: true,
	}
	return m, tea.Batch(
		loadPluginCmd(svc, summary.ID),
		loadReleasesCmd(svc, summary.ID),
	)
}

// SetSize запоминает текущие размеры body для рендеринга
func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	return m
}

// SetSession обновляет сессию
func (m Model) SetSession(s *auth.Session) Model {
	m.session = s
	if m.buttonCursor >= m.buttonCount() {
		m.buttonCursor = m.buttonCount() - 1
	}
	return m
}

// isAdmin проверяет, есть ли в текущей сессии роль market_admin
func (m Model) isAdmin() bool {
	if m.session == nil {
		return false
	}
	for _, r := range m.session.Roles {
		if r == adminRole {
			return true
		}
	}
	return false
}

func (m Model) verifyVisible() bool {
	if !m.isAdmin() {
		return false
	}
	return m.summary.TrustStatus == "VERIFIED" || m.summary.TrustStatus == "COMMUNITY"
}

// buttonCount - количество видимых кнопок
func (m Model) buttonCount() int {
	if m.verifyVisible() {
		return 3
	}
	return 2
}

// CapturingInput всгеда false, нет текстовых полей
func (m Model) CapturingInput() bool { return false }

// Update реагирует на входящие события для обновления модели
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case pluginLoadedMsg:
		return m.handlePluginLoaded(msg)

	case releasesLoadedMsg:
		return m.handleReleasesLoaded(msg)

	case artifactsLoadedMsg:
		return m.handleArtifactsLoaded(msg)

	case libraryActionMsg:
		return m.handleLibraryAction(msg)

	case artifactDownloadedMsg:
		return m.handleArtifactDownloaded(msg)

	case pluginVerifiedMsg:
		return m.handlePluginVerified(msg)
	}
	return m, nil
}

// handlePluginVerified обновляет локальный trust статус и статусную строку
func (m Model) handlePluginVerified(msg pluginVerifiedMsg) (Model, tea.Cmd) {
	m.actionInProgress = false
	if msg.err != nil {
		return m, tabs.SetStatus("Verify action failed: " + msg.err.Error())
	}
	m.summary.TrustStatus = msg.target
	label := "community"
	if msg.target == "VERIFIED" {
		label = "verified"
	}
	return m, tabs.SetStatus("Plugin marked as " + label)
}

func (m Model) handlePluginLoaded(msg pluginLoadedMsg) (Model, tea.Cmd) {
	m.loadingPlugin = false
	if msg.err != nil {
		return m, tabs.SetStatus("Failed to load plugin: " + msg.err.Error())
	}
	m.plugin = msg.plugin
	return m, nil
}

func (m Model) handleReleasesLoaded(msg releasesLoadedMsg) (Model, tea.Cmd) {
	m.loadingReleases = false
	if msg.err != nil {
		return m, tabs.SetStatus("Failed to load releases: " + msg.err.Error())
	}
	m.releases = msg.releases
	if len(m.releases) == 0 {
		return m, tabs.SetStatus("Plugin has no releases yet")
	}

	// По умолчанию выбираем релиз с is_latest=true, иначе первый
	m.releaseIdx = pickLatestRelease(m.releases)
	m.loadingArtifacts = true
	return m, loadArtifactsCmd(m.market, m.releases[m.releaseIdx].ID)
}

func (m Model) handleArtifactsLoaded(msg artifactsLoadedMsg) (Model, tea.Cmd) {
	m.loadingArtifacts = false
	if msg.err != nil {
		return m, tabs.SetStatus("Failed to load artifacts: " + msg.err.Error())
	}
	m.artifacts = msg.artifacts
	m.artifactIdx = pickHostArtifact(m.artifacts)
	return m, nil
}

func (m Model) handleLibraryAction(msg libraryActionMsg) (Model, tea.Cmd) {
	m.actionInProgress = false
	if msg.err != nil {
		return m, tabs.SetStatus("Library action failed: " + msg.err.Error())
	}
	m.summary.Installed = msg.installed

	verb := "Removed from"
	if msg.installed {
		verb = "Added to"
	}
	// Уведомляем роутер чтобы список синхронизировал бейдж
	notify := func() tea.Msg {
		return shared.LibraryChangedMsg{PluginID: m.summary.ID, Installed: msg.installed}
	}
	return m, tea.Batch(
		tabs.SetStatus(verb+" library: "+m.summary.Name),
		notify,
	)
}

// handleArtifactDownloaded реагирует на завершение Install в plugins.Manager.
// При успехе сообщает статус и где лежит, при ошибке выводит причину
func (m Model) handleArtifactDownloaded(msg artifactDownloadedMsg) (Model, tea.Cmd) {
	m.actionInProgress = false
	if msg.err != nil {
		return m, tabs.SetStatus("Artifact download failed: " + msg.err.Error())
	}
	if msg.installed == nil {
		return m, tabs.SetStatus("Artifact downloaded")
	}
	return m, tabs.SetStatus("Artifact installed locally: " + msg.installed.InstallDir)
}

// pickLatestRelease возвращает индекс релиза с IsLatest=true, иначе 0
func pickLatestRelease(releases []market.Release) int {
	for i, r := range releases {
		if r.IsLatest {
			return i
		}
	}
	return 0
}

// pickHostArtifact возвращает индекс артефакта под текущую платформу,
// иначе 0
func pickHostArtifact(arts []market.Artifact) int {
	for i, a := range arts {
		if a.OS == runtime.GOOS && a.Arch == runtime.GOARCH {
			return i
		}
	}
	return 0
}
