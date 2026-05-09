package plugins

import (
	"fmt"

	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/tabs"

	tea "github.com/charmbracelet/bubbletea"
)

var buttonLabels = []string{"Refresh", "Create", "Install", "Uninstall"}

// PluginsTab реализует tabs.Tab для панели маркетплейса плагинов
type PluginsTab struct {
	market  *market.Service
	session *auth.Session

	search     string
	focused    bool
	loading    bool
	loaded     bool
	items      []market.PluginSummary
	total      int64
	source     string // "installed" или "search"
	listCursor int
	button     int
	wizard     wizard
}

func New(market *market.Service) *PluginsTab {
	return &PluginsTab{market: market}
}

func (t *PluginsTab) Title() string { return "Plugins" }

func (t *PluginsTab) Init() tea.Cmd { return nil }

func (t *PluginsTab) Help() string {
	if t.wizard.open {
		return "tab tabs | esc cancel | up/down input/buttons | left/right buttons | enter next/create"
	}
	if t.session == nil {
		return "q quit | tab tabs | :auth login | :signup register"
	}
	return "q quit | tab tabs | arrows buttons/list | / search | enter action"
}

func (t *PluginsTab) CapturingInput() bool {
	return t.focused || t.wizard.open
}

func (t *PluginsTab) Update(msg tea.Msg) (tabs.Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tabs.SessionChangedMsg:
		t.session = msg.Session
		if t.session != nil && !t.loaded && !t.loading {
			return t, t.startLoad()
		}
		return t, nil

	case tabs.TabActivatedMsg:
		if t.session != nil && !t.loaded && !t.loading {
			return t, t.startLoad()
		}
		return t, nil

	case tea.KeyMsg:
		next, cmd := t.updateKey(msg)
		return &next, cmd

	case loadedMsg:
		return t.handleLoaded(msg)

	case actionCompletedMsg:
		return t.handleActionCompleted(msg)

	case artifactUploadedMsg:
		return t.handleArtifactUploaded(msg)

	case developerPluginCreatedMsg:
		return t.handleDeveloperPluginCreated(msg)
	}
	return t, nil
}

// startLoad помечает веладку как загружающуюся и возвращает Cmd для загрузки списка
func (t *PluginsTab) startLoad() tea.Cmd {
	t.loading = true
	return loadCmd(t.market, t.search)
}

func (t *PluginsTab) updateKey(msg tea.KeyMsg) (PluginsTab, tea.Cmd) {
	if t.wizard.open {
		return t.updateWizardKey(msg)
	}

	if t.focused {
		switch msg.String() {
		case "esc":
			t.focused = false
			return *t, tabs.SetStatus("Plugin search blurred")
		case "enter":
			t.focused = false
			return *t, t.startLoad()
		case "ctrl+u":
			t.search = ""
			return *t, t.startLoad()
		case "backspace", "delete":
			runes := []rune(t.search)
			if len(runes) > 0 {
				t.search = string(runes[:len(runes)-1])
			}
			return *t, t.startLoad()
		}

		if len(msg.Runes) > 0 {
			t.search += string(msg.Runes)
			return *t, t.startLoad()
		}

		return *t, nil
	}

	switch msg.String() {
	case "/":
		t.focused = true
		return *t, tabs.SetStatus("Plugin search")
	case "enter":
		return t.activateButton()
	case "ctrl+u":
		t.search = ""
		t.focused = true
		return *t, t.startLoad()
	case "right", "l":
		t.button = (t.button + 1) % len(buttonLabels)
		return *t, nil
	case "left", "h":
		t.button--
		if t.button < 0 {
			t.button = len(buttonLabels) - 1
		}
		return *t, nil
	case "up", "k":
		if t.listCursor > 0 {
			t.listCursor--
		}
		return *t, nil
	case "down", "j":
		if t.listCursor < len(t.items)-1 {
			t.listCursor++
		}
		return *t, nil
	}

	return *t, nil
}

func (t *PluginsTab) activateButton() (PluginsTab, tea.Cmd) {
	switch buttonLabels[t.button] {
	case "Refresh":
		return *t, t.startLoad()
	case "Create":
		t.wizard = newWizard()
		return *t, tabs.SetStatus("Plugin creation wizard")
	case "Install":
		if len(t.items) == 0 {
			return *t, tabs.SetStatus("Select a plugin to install")
		}
		pluginID := t.items[t.listCursor].ID
		return *t, tea.Batch(
			tabs.SetStatus(fmt.Sprintf("Installing plugin #%d...", pluginID)),
			installCmd(t.market, pluginID),
		)
	case "Uninstall":
		if len(t.items) == 0 {
			return *t, tabs.SetStatus("Select a plugin to uninstall")
		}
		pluginID := t.items[t.listCursor].ID
		return *t, tea.Batch(
			tabs.SetStatus(fmt.Sprintf("Uninstalling plugin #%d...", pluginID)),
			uninstallCmd(t.market, pluginID),
		)
	}
	return *t, nil
}

func (t *PluginsTab) handleLoaded(msg loadedMsg) (tabs.Tab, tea.Cmd) {
	t.loading = false
	t.loaded = true
	if msg.err != nil {
		return t, tabs.SetStatus(fmt.Sprintf("Failed to load plugins: %v", msg.err))
	}
	if msg.list == nil {
		return t, tabs.SetStatus("Failed to load plugins: empty response")
	}

	t.items = msg.list.Items
	t.total = msg.list.Total
	if t.listCursor >= len(t.items) {
		t.listCursor = max(0, len(t.items)-1)
	}
	if msg.list.Installed {
		t.source = "installed"
		return t, tabs.SetStatus(fmt.Sprintf("Loaded %d installed plugins", len(msg.list.Items)))
	}
	t.source = "search"
	return t, tabs.SetStatus(fmt.Sprintf("Found %d plugins", len(msg.list.Items)))
}

func (t *PluginsTab) handleActionCompleted(msg actionCompletedMsg) (tabs.Tab, tea.Cmd) {
	if msg.err != nil {
		return t, tabs.SetStatus(fmt.Sprintf("Plugin action failed: %v", msg.err))
	}
	if msg.reload {
		return t, tea.Batch(tabs.SetStatus(msg.message), t.startLoad())
	}
	return t, tabs.SetStatus(msg.message)
}

func (t *PluginsTab) handleArtifactUploaded(msg artifactUploadedMsg) (tabs.Tab, tea.Cmd) {
	if msg.err != nil {
		return t, tabs.SetStatus(fmt.Sprintf("Artifact upload failed: %v", msg.err))
	}
	if msg.artifact == nil {
		return t, tabs.SetStatus("Artifact upload failed: empty response")
	}
	return t, tabs.SetStatus(fmt.Sprintf("Artifact #%d uploaded (%d bytes, sha256 %s)",
		msg.artifact.ID, msg.artifact.Size, shortChecksum(msg.artifact.Checksum)))
}

func (t *PluginsTab) handleDeveloperPluginCreated(msg developerPluginCreatedMsg) (tabs.Tab, tea.Cmd) {
	t.wizard.submitting = false
	if msg.err != nil {
		return t, tabs.SetStatus(fmt.Sprintf("Plugin creation failed: %v", msg.err))
	}
	if msg.plugin == nil {
		return t, tabs.SetStatus("Plugin creation failed: empty response")
	}
	t.wizard = wizard{}
	t.search = msg.plugin.Name
	return t, tea.Batch(
		tabs.SetStatus(fmt.Sprintf("Created plugin #%d release #%d artifact #%d",
			msg.plugin.PluginID, msg.plugin.ReleaseID, msg.plugin.ArtifactID)),
		t.startLoad(),
	)
}

func (t *PluginsTab) RunCommand(args []string) tea.Cmd {
	return dispatchCommand(t, args)
}
