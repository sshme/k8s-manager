package detail

import (
	"context"
	"fmt"
	"time"

	"k8s-manager/cli/internal/market"
	pluginsmgr "k8s-manager/cli/internal/plugins"

	tea "github.com/charmbracelet/bubbletea"
)

type pluginLoadedMsg struct {
	plugin *market.PluginDetails
	err    error
}

type releasesLoadedMsg struct {
	releases []market.Release
	err      error
}

type artifactsLoadedMsg struct {
	artifacts []market.Artifact
	err       error
}

// libraryActionMsg - результат Install/Uninstall на сервере
type libraryActionMsg struct {
	installed bool
	err       error
}

// artifactDownloadedMsg - результат скачивания и распаковки артефакта.
// installed заполняется только при успехе
type artifactDownloadedMsg struct {
	installed *pluginsmgr.InstalledArtifact
	err       error
}

func (pluginLoadedMsg) PluginMsg()       {}
func (releasesLoadedMsg) PluginMsg()     {}
func (artifactsLoadedMsg) PluginMsg()    {}
func (libraryActionMsg) PluginMsg()      {}
func (artifactDownloadedMsg) PluginMsg() {}

func loadPluginCmd(svc *market.Service, pluginID int64) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return pluginLoadedMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p, err := svc.GetPlugin(ctx, pluginID)
		return pluginLoadedMsg{plugin: p, err: err}
	}
}

func loadReleasesCmd(svc *market.Service, pluginID int64) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return releasesLoadedMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		releases, err := svc.ListReleases(ctx, pluginID)
		return releasesLoadedMsg{releases: releases, err: err}
	}
}

func loadArtifactsCmd(svc *market.Service, releaseID int64) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return artifactsLoadedMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		arts, err := svc.ListArtifacts(ctx, releaseID)
		return artifactsLoadedMsg{artifacts: arts, err: err}
	}
}

func addToLibraryCmd(svc *market.Service, pluginID int64) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return libraryActionMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := svc.InstallPlugin(ctx, pluginID); err != nil {
			return libraryActionMsg{err: err}
		}
		return libraryActionMsg{installed: true}
	}
}

func removeFromLibraryCmd(svc *market.Service, pluginID int64) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return libraryActionMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := svc.UninstallPlugin(ctx, pluginID); err != nil {
			return libraryActionMsg{err: err}
		}
		return libraryActionMsg{installed: false}
	}
}

// downloadArtifactCmd запускает скачивание и распаковку через plugins.Manager.
// Таймаут 5 минут.
func downloadArtifactCmd(mgr *pluginsmgr.Manager, ref pluginsmgr.InstallRef) tea.Cmd {
	return func() tea.Msg {
		if mgr == nil {
			return artifactDownloadedMsg{err: fmt.Errorf("plugins manager is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		installed, err := mgr.Install(ctx, ref)
		return artifactDownloadedMsg{installed: installed, err: err}
	}
}
