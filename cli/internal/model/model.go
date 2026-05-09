package model

import (
	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/tabs"
	"k8s-manager/cli/internal/model/tabs/plugins"
	"k8s-manager/cli/internal/model/tabs/profile"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	width  int
	height int

	tabs    []tabs.Tab
	active  int
	plugins *plugins.PluginsTab // для роутинга в command-mode
	profile *profile.ProfileTab // роутинга в command-mode и хедера

	status       string
	commandMode  bool
	commandInput string
}

func New(authService *auth.Service, marketService *market.Service) model {
	pluginsTab := plugins.New(marketService)
	profileTab := profile.New(authService)

	return model{
		tabs: []tabs.Tab{
			tabs.NewServices(),
			pluginsTab,
			profileTab,
		},
		plugins: pluginsTab,
		profile: profileTab,
		status:  "Press :auth to sign in or :signup to create an account",
	}
}

func (m model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.tabs))
	for _, t := range m.tabs {
		if c := t.Init(); c != nil {
			cmds = append(cmds, c)
		}
	}
	return tea.Batch(cmds...)
}
