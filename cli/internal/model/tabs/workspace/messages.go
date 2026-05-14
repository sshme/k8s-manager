package workspace

import pluginsmgr "k8s-manager/cli/internal/plugins"

type namespacesLoadedMsg struct {
	items []string
	err   error
}

type deploymentsLoadedMsg struct {
	items []string
	err   error
}

type installedReloadedMsg struct {
	items []pluginsmgr.InstalledArtifact
}

type uninstallDoneMsg struct {
	plugin string
	err    error
}

type pluginFinishedMsg struct {
	subcommand string
	plugin     string
	err        error
}
