package plugins

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/tabs"
	"k8s-manager/cli/internal/model/tabs/plugins/shared"

	tea "github.com/charmbracelet/bubbletea"
)

// RunCommand - точка входа из command-mode (:plugin ...).
// Поддерживает refresh, install <id>, uninstall <id>, upload <release-id> <zip> [os] [arch]
func (t *Tab) RunCommand(args []string) tea.Cmd {
	if len(args) == 0 {
		return tabs.SetStatus("Usage: :plugin install <id> | :plugin uninstall <id> | :plugin refresh | :plugin upload <release-id> <zip> [os] [arch]")
	}

	switch args[0] {
	case "refresh":
		return t.Reload()

	case "install":
		return runLibraryCmd(t.market, args, true)

	case "uninstall":
		return runLibraryCmd(t.market, args, false)

	case "upload":
		return runUploadCmd(t.market, args)
	}

	return tabs.SetStatus(fmt.Sprintf("Unknown plugin command: %s", args[0]))
}

// runUploadCmd парсит "upload <release-id> <zip> [os] [arch]" и шлёт артефакт
func runUploadCmd(svc *market.Service, args []string) tea.Cmd {
	if len(args) < 3 {
		return tabs.SetStatus("Usage: :plugin upload <release-id> <zip-path> [os] [arch]")
	}
	releaseID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil || releaseID <= 0 {
		return tabs.SetStatus("Release id must be a positive number")
	}
	zipPath := args[2]
	osName := ""
	arch := ""
	if len(args) > 3 {
		osName = args[3]
	}
	if len(args) > 4 {
		arch = args[4]
	}

	action := func() tea.Msg {
		if svc == nil {
			return tabs.StatusMsg("market service is not configured")
		}
		// Артефакты могут быть большие, поэтому 2 минуты на загрузку
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		artifact, err := svc.UploadArtifact(ctx, releaseID, zipPath, osName, arch)
		if err != nil {
			return tabs.StatusMsg(fmt.Sprintf("Artifact upload failed: %v", err))
		}
		return tabs.StatusMsg(fmt.Sprintf("Artifact #%d uploaded (%d bytes)", artifact.ID, artifact.Size))
	}

	return tea.Batch(
		tabs.SetStatus(fmt.Sprintf("Uploading artifact for release #%d...", releaseID)),
		action,
	)
}

// runLibraryCmd парсит plugin-id и вызывает Install/Uninstall на сервере.
// После успеха возвращает LibraryChangedMsg, чтобы список перерисовался
func runLibraryCmd(svc *market.Service, args []string, install bool) tea.Cmd {
	verb := "uninstall"
	if install {
		verb = "install"
	}
	if len(args) < 2 {
		return tabs.SetStatus("Usage: :plugin " + verb + " <id>")
	}
	pluginID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil || pluginID <= 0 {
		return tabs.SetStatus("Plugin id must be a positive number")
	}

	action := func() tea.Msg {
		if svc == nil {
			return tabs.StatusMsg("market service is not configured")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if install {
			if _, err := svc.InstallPlugin(ctx, pluginID); err != nil {
				return tabs.StatusMsg(fmt.Sprintf("Install failed: %v", err))
			}
		} else {
			if err := svc.UninstallPlugin(ctx, pluginID); err != nil {
				return tabs.StatusMsg(fmt.Sprintf("Uninstall failed: %v", err))
			}
		}
		return shared.LibraryChangedMsg{PluginID: pluginID, Installed: install}
	}

	return tea.Batch(
		tabs.SetStatus(fmt.Sprintf("Running %s on plugin #%d...", verb, pluginID)),
		action,
	)
}
