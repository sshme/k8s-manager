package detail

import (
	"context"
	"fmt"
	"time"

	"k8s-manager/cli/internal/market"

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
