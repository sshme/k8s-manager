package plugins

import (
	"fmt"
	"strconv"

	"k8s-manager/cli/internal/model/tabs"

	tea "github.com/charmbracelet/bubbletea"
)

// dispatchCommand обрабатывает команды из command-mode для плагинов.
// Поддерживает: refresh, install <id>, uninstall <id>, upload <release-id> <zip-path> [os] [arch].
func dispatchCommand(t *PluginsTab, args []string) tea.Cmd {
	if len(args) == 0 {
		return tabs.SetStatus("Usage: :plugin install <id> | :plugin uninstall <id> | :plugin upload <release-id> <zip-path> [os] [arch]")
	}

	switch args[0] {
	case "refresh":
		return t.startLoad()

	case "install":
		if len(args) < 2 {
			return tabs.SetStatus("Usage: :plugin install <id>")
		}
		pluginID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || pluginID <= 0 {
			return tabs.SetStatus("Plugin id must be a positive number")
		}
		return tea.Batch(
			tabs.SetStatus(fmt.Sprintf("Installing plugin #%d...", pluginID)),
			installCmd(t.market, pluginID),
		)

	case "uninstall":
		if len(args) < 2 {
			return tabs.SetStatus("Usage: :plugin uninstall <id>")
		}
		pluginID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || pluginID <= 0 {
			return tabs.SetStatus("Plugin id must be a positive number")
		}
		return tea.Batch(
			tabs.SetStatus(fmt.Sprintf("Uninstalling plugin #%d...", pluginID)),
			uninstallCmd(t.market, pluginID),
		)

	case "upload":
		if len(args) < 3 {
			return tabs.SetStatus("Usage: :plugin upload <release-id> <zip-path> [os] [arch]")
		}
		releaseID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || releaseID <= 0 {
			return tabs.SetStatus("Release id must be a positive number")
		}
		osName := ""
		arch := ""
		if len(args) > 3 {
			osName = args[3]
		}
		if len(args) > 4 {
			arch = args[4]
		}
		return tea.Batch(
			tabs.SetStatus(fmt.Sprintf("Uploading artifact for release #%d...", releaseID)),
			uploadArtifactCmd(t.market, releaseID, args[2], osName, arch),
		)
	}

	return tabs.SetStatus(fmt.Sprintf("Unknown plugin command: %s", args[0]))
}

// Reload запускает перезагрузку списка (аналог команды :plugins).
func (t *PluginsTab) Reload() tea.Cmd {
	return t.startLoad()
}
