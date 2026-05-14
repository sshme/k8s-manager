// Package workspace - вкладка выбора k8s target и работы с локально
// установленными плагинами
package workspace

import (
	"errors"
	"fmt"
	"os/exec"

	"k8s-manager/cli/internal/k8s"
	"k8s-manager/cli/internal/model/tabs"
	"k8s-manager/cli/internal/model/tabs/workspace/dialog"
	"k8s-manager/cli/internal/model/tabs/workspace/selector"
	pluginsmgr "k8s-manager/cli/internal/plugins"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

var errNoClient = errors.New("kubernetes client is not configured")

type focusZone int

const (
	focusContext focusZone = iota
	focusNamespace
	focusDeployment
	focusList
)

// Tab - вкладка Workspace
type Tab struct {
	plugins *pluginsmgr.Manager
	client  k8s.Client

	ctxSelector  selector.Model
	nsSelector   selector.Model
	deplSelector selector.Model
	dialog       dialog.Model

	contexts []k8s.ContextInfo

	installedItems []pluginsmgr.InstalledArtifact
	listCursor     int
	viewport       viewport.Model
	viewportInit   bool

	focus         focusZone
	width, height int
}

// New создаёт вкладку
func New(mgr *pluginsmgr.Manager) *Tab {
	return &Tab{
		plugins:      mgr,
		ctxSelector:  selector.New("Context"),
		nsSelector:   selector.New("Namespace"),
		deplSelector: selector.New("Deployment"),
		dialog:       dialog.New(),
		viewport:     viewport.New(0, 0),
		focus:        focusContext,
	}
}

func (t *Tab) Title() string { return "Workspace" }

func (t *Tab) Init() tea.Cmd {
	contexts, current, err := k8s.LoadContexts()
	if err != nil {
		return tabs.SetStatus(fmt.Sprintf("Kubeconfig load failed: %v", err))
	}
	t.contexts = contexts

	names := make([]string, 0, len(contexts))
	currentIdx := 0
	for i, c := range contexts {
		names = append(names, c.Name)
		if c.Name == current {
			currentIdx = i
		}
	}
	t.ctxSelector = t.ctxSelector.SetItems(names, currentIdx)

	cmds := []tea.Cmd{reloadInstalledCmd(t.plugins), t.emitTarget()}
	if t.ctxSelector.Selected() != "" {
		cmd, err := t.activateContext(t.ctxSelector.Selected())
		if err != nil {
			cmds = append(cmds, tabs.SetStatus(fmt.Sprintf("Cluster connect failed: %v", err)))
		} else if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (t *Tab) emitTarget() tea.Cmd {
	return tabs.SetWorkspaceTarget(t.ctxSelector.Selected(), t.nsSelector.Selected())
}

func (t *Tab) CapturingInput() bool {
	if t.dialog.Active() {
		return true
	}
	if t.ctxSelector.IsOpen() || t.nsSelector.IsOpen() || t.deplSelector.IsOpen() {
		return true
	}
	return false
}

func (t *Tab) Help() string {
	if t.dialog.Active() {
		return "esc cancel | ←/→ buttons | enter activate"
	}
	if t.ctxSelector.IsOpen() || t.nsSelector.IsOpen() || t.deplSelector.IsOpen() {
		return "esc close | ↑/↓ select | enter confirm"
	}
	if t.focus == focusList {
		return "q quit | ↑/↓ list | enter open | r reload | ←/→ selectors"
	}
	return "q quit | ←/→ selectors | ↓ list | enter dropdown | r reload"
}

func (t *Tab) Update(msg tea.Msg) (tabs.Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		t.viewport.Width = msg.Width
		return t, nil

	case tabs.TabActivatedMsg:
		return t, reloadInstalledCmd(t.plugins)

	case installedReloadedMsg:
		t.installedItems = msg.items
		if t.listCursor >= len(t.installedItems) {
			if len(t.installedItems) == 0 {
				t.listCursor = 0
			} else {
				t.listCursor = len(t.installedItems) - 1
			}
		}
		return t, nil

	case namespacesLoadedMsg:
		return t.handleNamespacesLoaded(msg)

	case deploymentsLoadedMsg:
		return t.handleDeploymentsLoaded(msg)

	case uninstallDoneMsg:
		return t.handleUninstallDone(msg)

	case pluginFinishedMsg:
		return t.handlePluginFinished(msg)

	case tea.KeyMsg:
		return t.handleKey(msg)
	}
	return t, nil
}

func (t *Tab) handlePluginFinished(msg pluginFinishedMsg) (tabs.Tab, tea.Cmd) {
	if msg.err == nil {
		return t, tabs.SetStatus(fmt.Sprintf("Plugin %s %s finished successfully", msg.plugin, msg.subcommand))
	}
	var exitErr *exec.ExitError
	if errors.As(msg.err, &exitErr) {
		return t, tabs.SetStatus(fmt.Sprintf("Plugin %s %s failed (exit code %d)", msg.plugin, msg.subcommand, exitErr.ExitCode()))
	}
	return t, tabs.SetStatus(fmt.Sprintf("Plugin %s %s failed: %v", msg.plugin, msg.subcommand, msg.err))
}

// activateContext (пере-)создаёт k8s.Client под выбранный контекст и запускает
// загрузку namespaces. Возвращает Cmd либо ошибку построения клиента.
func (t *Tab) activateContext(name string) (tea.Cmd, error) {
	client, err := k8s.NewClient(name)
	if err != nil {
		t.client = nil
		t.nsSelector = t.nsSelector.SetItems(nil, 0)
		t.deplSelector = t.deplSelector.SetItems(nil, 0)
		return nil, err
	}
	t.client = client
	t.nsSelector = t.nsSelector.SetLoading(true)
	t.deplSelector = t.deplSelector.SetLoading(true)
	return loadNamespacesCmd(client), nil
}

func (t *Tab) handleNamespacesLoaded(msg namespacesLoadedMsg) (tabs.Tab, tea.Cmd) {
	if msg.err != nil {
		t.nsSelector = t.nsSelector.SetItems(nil, 0)
		t.deplSelector = t.deplSelector.SetItems(nil, 0)
		return t, tea.Batch(
			t.emitTarget(),
			tabs.SetStatus(fmt.Sprintf("Namespaces load failed: %v", msg.err)),
		)
	}

	// namespace из текущего контекста kubeconfig, иначе default
	defaultNS := t.defaultNamespaceForCurrentContext()
	idx := indexOf(msg.items, defaultNS)
	if idx < 0 {
		idx = indexOf(msg.items, "default")
	}
	if idx < 0 {
		idx = 0
	}
	t.nsSelector = t.nsSelector.SetItems(msg.items, idx)

	cmds := []tea.Cmd{t.emitTarget()}
	if t.nsSelector.Selected() == "" {
		t.deplSelector = t.deplSelector.SetItems(nil, 0)
		return t, tea.Batch(cmds...)
	}
	t.deplSelector = t.deplSelector.SetLoading(true)
	cmds = append(cmds, loadDeploymentsCmd(t.client, t.nsSelector.Selected()))
	return t, tea.Batch(cmds...)
}

func (t *Tab) handleDeploymentsLoaded(msg deploymentsLoadedMsg) (tabs.Tab, tea.Cmd) {
	if msg.err != nil {
		t.deplSelector = t.deplSelector.SetItems(nil, 0)
		return t, tabs.SetStatus(fmt.Sprintf("Deployments load failed: %v", msg.err))
	}
	t.deplSelector = t.deplSelector.SetItems(msg.items, 0)
	return t, nil
}

func (t *Tab) handleUninstallDone(msg uninstallDoneMsg) (tabs.Tab, tea.Cmd) {
	t.dialog = t.dialog.Cancel()
	if msg.err != nil {
		return t, tabs.SetStatus(fmt.Sprintf("Uninstall failed: %v", msg.err))
	}
	return t, tea.Batch(
		tabs.SetStatus(fmt.Sprintf("Uninstalled: %s", msg.plugin)),
		reloadInstalledCmd(t.plugins),
	)
}

func (t *Tab) defaultNamespaceForCurrentContext() string {
	selected := t.ctxSelector.Selected()
	for _, c := range t.contexts {
		if c.Name == selected {
			if c.DefaultNamespace != "" {
				return c.DefaultNamespace
			}
			return "default"
		}
	}
	return "default"
}

func indexOf(items []string, target string) int {
	if target == "" {
		return -1
	}
	for i, v := range items {
		if v == target {
			return i
		}
	}
	return -1
}
