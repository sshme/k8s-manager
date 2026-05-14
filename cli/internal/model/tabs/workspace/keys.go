package workspace

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"k8s-manager/cli/internal/model/tabs"
	"k8s-manager/cli/internal/model/tabs/workspace/dialog"
	"k8s-manager/cli/internal/model/tabs/workspace/runner"
	"k8s-manager/cli/internal/model/tabs/workspace/selector"
	pluginsmgr "k8s-manager/cli/internal/plugins"

	tea "github.com/charmbracelet/bubbletea"
)

func (t *Tab) handleKey(msg tea.KeyMsg) (tabs.Tab, tea.Cmd) {
	if t.dialog.Active() {
		return t.handleDialogKey(msg)
	}
	if t.ctxSelector.IsOpen() {
		return t.handleSelectorKey(&t.ctxSelector, msg, t.afterContextSelected)
	}
	if t.nsSelector.IsOpen() {
		return t.handleSelectorKey(&t.nsSelector, msg, t.afterNamespaceSelected)
	}
	if t.deplSelector.IsOpen() {
		return t.handleSelectorKey(&t.deplSelector, msg, nil)
	}

	switch msg.String() {
	case "left", "h":
		t.moveFocusLeft()
		return t, nil
	case "right", "l":
		t.moveFocusRight()
		return t, nil
	case "up", "k":
		if t.focus == focusList {
			if t.listCursor > 0 {
				t.listCursor--
			} else {
				t.focus = focusDeployment
			}
		}
		return t, nil
	case "down", "j":
		if t.focus == focusList {
			if t.listCursor < len(t.installedItems)-1 {
				t.listCursor++
			}
		} else {
			t.focus = focusList
		}
		return t, nil
	case "enter":
		return t.activateFocus()
	case "r":
		return t, reloadInstalledCmd(t.plugins)
	}
	return t, nil
}

func (t *Tab) moveFocusLeft() {
	switch t.focus {
	case focusNamespace:
		t.focus = focusContext
	case focusDeployment:
		t.focus = focusNamespace
	case focusList:
		t.focus = focusDeployment
	}
}

func (t *Tab) moveFocusRight() {
	switch t.focus {
	case focusContext:
		t.focus = focusNamespace
	case focusNamespace:
		t.focus = focusDeployment
	case focusDeployment:
		t.focus = focusList
	}
}

func (t *Tab) activateFocus() (tabs.Tab, tea.Cmd) {
	switch t.focus {
	case focusContext:
		t.ctxSelector = t.ctxSelector.Open()
	case focusNamespace:
		t.nsSelector = t.nsSelector.Open()
	case focusDeployment:
		t.deplSelector = t.deplSelector.Open()
	case focusList:
		return t.openDialog()
	}
	return t, nil
}

func (t *Tab) openDialog() (tabs.Tab, tea.Cmd) {
	if len(t.installedItems) == 0 {
		return t, tabs.SetStatus("No installed plugins")
	}
	if t.listCursor >= len(t.installedItems) {
		t.listCursor = len(t.installedItems) - 1
	}
	t.dialog = t.dialog.Open(t.installedItems[t.listCursor])
	return t, nil
}

// handleSelectorKey перенаправляет клавишу в выбранный селектор и, если был
// подтверждён выбор, вызывает onConfirm.
func (t *Tab) handleSelectorKey(sel *selector.Model, msg tea.KeyMsg, onConfirm func() tea.Cmd) (tabs.Tab, tea.Cmd) {
	next, confirmed := sel.Update(msg)
	*sel = next
	if confirmed && onConfirm != nil {
		return t, onConfirm()
	}
	return t, nil
}

func (t *Tab) afterContextSelected() tea.Cmd {
	cmd, err := t.activateContext(t.ctxSelector.Selected())
	if err != nil {
		return tea.Batch(
			t.emitTarget(),
			tabs.SetStatus(fmt.Sprintf("Cluster connect failed: %v", err)),
		)
	}
	if cmd == nil {
		return t.emitTarget()
	}
	return tea.Batch(t.emitTarget(), cmd)
}

func (t *Tab) afterNamespaceSelected() tea.Cmd {
	if t.client == nil {
		return t.emitTarget()
	}
	t.deplSelector = t.deplSelector.SetLoading(true)
	return tea.Batch(t.emitTarget(), loadDeploymentsCmd(t.client, t.nsSelector.Selected()))
}

func (t *Tab) handleDialogKey(msg tea.KeyMsg) (tabs.Tab, tea.Cmd) {
	next, action := t.dialog.Update(msg)
	t.dialog = next
	switch action {
	case dialog.ActionNone, dialog.ActionCancel:
		return t, nil
	case dialog.ActionDelete:
		t.dialog = t.dialog.SetWorking(true)
		a := t.dialog.Artifact()
		return t, tea.Batch(
			tabs.SetStatus(fmt.Sprintf("Deleting %s...", a.PluginName)),
			uninstallCmd(t.plugins, a),
		)
	case dialog.ActionPlan:
		return t.startPluginRun("plan")
	case dialog.ActionApply:
		return t.startPluginRun("apply")
	}
	return t, nil
}

func (t *Tab) startPluginRun(subcommand string) (tabs.Tab, tea.Cmd) {
	a := t.dialog.Artifact()
	t.dialog = t.dialog.Cancel()

	if t.ctxSelector.Selected() == "" {
		return t, tabs.SetStatus("Select a context first")
	}
	if t.nsSelector.Selected() == "" {
		return t, tabs.SetStatus("Select a namespace first")
	}
	if t.deplSelector.Selected() == "" {
		return t, tabs.SetStatus("Select a deployment first")
	}
	if _, err := exec.LookPath("go"); err != nil {
		return t, tabs.SetStatus("Cannot run plugin: 'go' not found in PATH")
	}

	logDir := filepath.Join(pluginsmgr.LoadConfig().Root, "logs")
	cmd, err := runner.Build(a, subcommand,
		t.ctxSelector.Selected(),
		t.nsSelector.Selected(),
		t.deplSelector.Selected(),
		logDir,
	)
	if err != nil {
		return t, tabs.SetStatus(fmt.Sprintf("Cannot start plugin: %v", err))
	}

	return t, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return pluginFinishedMsg{subcommand: subcommand, plugin: a.PluginName, err: err}
	})
}
