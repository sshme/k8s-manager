package workspace

import (
	"context"
	"time"

	"k8s-manager/cli/internal/k8s"
	pluginsmgr "k8s-manager/cli/internal/plugins"

	tea "github.com/charmbracelet/bubbletea"
)

const k8sCallTimeout = 10 * time.Second

func loadNamespacesCmd(client k8s.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return namespacesLoadedMsg{err: errNoClient}
		}
		ctx, cancel := context.WithTimeout(context.Background(), k8sCallTimeout)
		defer cancel()
		items, err := client.ListNamespaces(ctx)
		return namespacesLoadedMsg{items: items, err: err}
	}
}

func loadDeploymentsCmd(client k8s.Client, namespace string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return deploymentsLoadedMsg{err: errNoClient}
		}
		if namespace == "" {
			return deploymentsLoadedMsg{items: nil}
		}
		ctx, cancel := context.WithTimeout(context.Background(), k8sCallTimeout)
		defer cancel()
		items, err := client.ListDeployments(ctx, namespace)
		return deploymentsLoadedMsg{items: items, err: err}
	}
}

func reloadInstalledCmd(mgr *pluginsmgr.Manager) tea.Cmd {
	return func() tea.Msg {
		return installedReloadedMsg{items: mgr.ListInstalled()}
	}
}

func uninstallCmd(mgr *pluginsmgr.Manager, a pluginsmgr.InstalledArtifact) tea.Cmd {
	return func() tea.Msg {
		err := mgr.Uninstall(a.PluginID, a.Version, a.OS, a.Arch)
		return uninstallDoneMsg{plugin: a.PluginName, err: err}
	}
}
