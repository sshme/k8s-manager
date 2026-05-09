package plugins

import (
	"context"
	"fmt"
	"time"

	"k8s-manager/cli/internal/market"

	tea "github.com/charmbracelet/bubbletea"
)

type loadedMsg struct {
	list *market.PluginList
	err  error
}

type actionCompletedMsg struct {
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

func loadCmd(svc *market.Service, query string) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return loadedMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		list, err := svc.ListPlugins(ctx, query)
		return loadedMsg{list: list, err: err}
	}
}

func installCmd(svc *market.Service, pluginID int64) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return actionCompletedMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		plugin, err := svc.InstallPlugin(ctx, pluginID)
		if err != nil {
			return actionCompletedMsg{err: err}
		}
		return actionCompletedMsg{
			message: fmt.Sprintf("Installed plugin #%d %s", plugin.ID, plugin.Name),
			reload:  true,
		}
	}
}

func uninstallCmd(svc *market.Service, pluginID int64) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return actionCompletedMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := svc.UninstallPlugin(ctx, pluginID); err != nil {
			return actionCompletedMsg{err: err}
		}
		return actionCompletedMsg{
			message: fmt.Sprintf("Uninstalled plugin #%d", pluginID),
			reload:  true,
		}
	}
}

func uploadArtifactCmd(svc *market.Service, releaseID int64, zipPath, osName, arch string) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return artifactUploadedMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		artifact, err := svc.UploadArtifact(ctx, releaseID, zipPath, osName, arch)
		return artifactUploadedMsg{artifact: artifact, err: err}
	}
}

func createDeveloperPluginCmd(svc *market.Service, draft market.DeveloperPluginDraft) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return developerPluginCreatedMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		plugin, err := svc.CreateDeveloperPlugin(ctx, draft)
		return developerPluginCreatedMsg{plugin: plugin, err: err}
	}
}
