package model

import (
	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/market"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	width            int
	height           int
	items            []string
	cursor           int
	status           string
	commandMode      bool
	commandInput     string
	authInProgress   bool
	authService      *auth.Service
	marketService    *market.Service
	session          *auth.Session
	pluginSearch     string
	pluginFocused    bool
	pluginLoading    bool
	pluginLoaded     bool
	pluginItems      []market.PluginSummary
	pluginTotal      int64
	pluginSource     string
	pluginListCursor int
	pluginButton     int
	pluginWizard     pluginWizard
}

type authLoadedMsg struct {
	session *auth.Session
	err     error
}

type authCompletedMsg struct {
	session *auth.Session
	err     error
}

type pluginsLoadedMsg struct {
	list *market.PluginList
	err  error
}

type pluginActionCompletedMsg struct {
	message string
	err     error
	reload  bool
}

type artifactUploadedMsg struct {
	artifact *market.UploadedArtifact
	err      error
}

type developerPluginCreatedMsg struct {
	plugin *market.DeveloperPlugin
	err    error
}

type authAction string

type wizardFocus int

const (
	wizardFocusInput wizardFocus = iota
	wizardFocusButtons
)

type wizardField struct {
	label        string
	value        string
	placeholder  string
	defaultValue string
	required     bool
}

type pluginWizard struct {
	open       bool
	step       int
	button     int
	focus      wizardFocus
	submitting bool
	fields     []wizardField
}

const (
	authActionLogin  authAction = "auth"
	authActionSignup authAction = "signup"
)

const (
	wizardPublisherName = iota
	wizardIdentifier
	wizardPluginName
	wizardDescription
	wizardCategory
	wizardVersion
	wizardZipPath
	wizardOS
	wizardArch
)

func New(authService *auth.Service, marketService *market.Service) model {
	return model{
		items: []string{
			"Services",
			"Plugins",
			"Logs",
		},
		status:        "Press :auth to sign in or :signup to create an account",
		authService:   authService,
		marketService: marketService,
	}
}

func (m model) Init() tea.Cmd {
	return loadSessionCmd(m.authService)
}
