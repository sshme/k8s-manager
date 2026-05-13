package detail

import (
	"k8s-manager/cli/internal/model/tabs"
	pluginsmgr "k8s-manager/cli/internal/plugins"

	tea "github.com/charmbracelet/bubbletea"
)

// activateButton исполняет действие выбранной кнопки
func (m Model) activateButton() (Model, tea.Cmd) {
	if m.actionInProgress {
		return m, tabs.SetStatus("Another action is already in progress")
	}

	switch m.buttonCursor {
	case btnLibrary:
		return m.toggleLibrary()
	case btnArtifact:
		return m.startArtifactDownload()
	case btnVerify:
		// TODO: Заглушка, модерация с ролью
		return m, tabs.SetStatus("Server-side moderation is not implemented yet")
	}
	return m, nil
}

// startArtifactDownload запускает downloadArtifactCmd.
// Если артефакт уже стоит локально с тем же checksum, сразу выходит сообщением
// без сетевого вызова.
func (m Model) startArtifactDownload() (Model, tea.Cmd) {
	if m.plugins == nil {
		return m, tabs.SetStatus("Plugins manager is not configured")
	}
	if m.plugin == nil {
		return m, tabs.SetStatus("Plugin is still loading, try again in a moment")
	}
	if len(m.releases) == 0 || m.releaseIdx < 0 || m.releaseIdx >= len(m.releases) {
		return m, tabs.SetStatus("No release selected")
	}
	if len(m.artifacts) == 0 || m.artifactIdx < 0 || m.artifactIdx >= len(m.artifacts) {
		return m, tabs.SetStatus("No artifact for the current platform")
	}

	release := m.releases[m.releaseIdx]
	artifact := m.artifacts[m.artifactIdx]

	// Если уже скачано, то не скачиваем повторно
	if m.plugins.IsInstalled(m.plugin.ID, release.Version, artifact.OS, artifact.Arch) {
		return m, tabs.SetStatus("Artifact already installed locally")
	}

	ref := pluginsmgr.InstallRef{
		PluginID:         m.plugin.ID,
		PluginIdentifier: m.plugin.Identifier,
		PluginName:       m.plugin.Name,
		PublisherID:      m.plugin.PublisherID,
		PublisherName:    m.plugin.PublisherName,
		ReleaseID:        release.ID,
		Version:          release.Version,
		Artifact:         artifact,
	}

	m.actionInProgress = true
	return m, tea.Batch(
		tabs.SetStatus("Downloading artifact "+release.Version+" for "+artifact.OS+"/"+artifact.Arch+"..."),
		downloadArtifactCmd(m.plugins, ref),
	)
}

// toggleLibrary запускает Install или Uninstall в зависимости от текущего
// флага Installed. До получения ответа блокирует повторные нажатия.
func (m Model) toggleLibrary() (Model, tea.Cmd) {
	m.actionInProgress = true

	if m.summary.Installed {
		return m, tea.Batch(
			tabs.SetStatus("Removing from library..."),
			removeFromLibraryCmd(m.market, m.summary.ID),
		)
	}
	return m, tea.Batch(
		tabs.SetStatus("Adding to library..."),
		addToLibraryCmd(m.market, m.summary.ID),
	)
}

// libraryButtonLabel возвращает актуальную надпись на кнопке библиотеки
func (m Model) libraryButtonLabel() string {
	if m.summary.Installed {
		return "Remove from library"
	}
	return "Add to library"
}

// verifyButtonLabel возвращает Verify или Unverify в зависимости от
// текущего trust-статуса
func (m Model) verifyButtonLabel() string {
	if m.summary.TrustStatus == "VERIFIED" {
		return "Unverify"
	}
	return "Verify"
}

// artifactButtonLabel - всегда "Install artifact" пока скачивание заглушка
func (m Model) artifactButtonLabel() string {
	return "Install artifact"
}
