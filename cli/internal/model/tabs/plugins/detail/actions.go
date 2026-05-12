package detail

import (
	"k8s-manager/cli/internal/model/tabs"

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
		// TODO: Заглушка, скачивание плагина
		return m, tabs.SetStatus("Plugin install is not implemented yet")
	case btnVerify:
		// TODO: Заглушка, модерация с ролью
		return m, tabs.SetStatus("Server-side moderation is not implemented yet")
	}
	return m, nil
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
