package detail

import (
	"strings"

	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/styles"
	"k8s-manager/cli/internal/model/tabs/plugins/shared"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View(width, height int) string {
	if m.width == 0 {
		m = m.SetSize(width, height)
	}

	if m.loadingPlugin && m.loadingReleases {
		return styles.Warning.Render("Loading plugin details...")
	}

	sections := []string{
		m.renderHeader(width),
		"",
		m.renderMeta(width),
		"",
		m.renderDescription(width),
		"",
		m.renderSelectors(),
		"",
		m.renderReleaseAndArtifact(width),
	}

	if changelog := m.renderChangelog(width); changelog != "" {
		sections = append(sections, "", changelog)
	}

	sections = append(sections, "", m.renderButtons())

	result := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(result)
}

// renderHeader - "Esc back | <name>" слева, бейджи trust/installed справа
func (m Model) renderHeader(width int) string {
	back := styles.Subtle.Render("Esc back")
	title := styles.Title.Render(m.summary.Name)
	left := lipgloss.JoinHorizontal(lipgloss.Top, back, "  ", title)

	right := ""
	switch m.summary.TrustStatus {
	case "VERIFIED":
		right = styles.Verified.Render("VERIFIED")
	case "OFFICIAL":
		right = styles.Official.Render("OFFICIAL")
	}
	if m.summary.Installed {
		if right != "" {
			right = lipgloss.JoinHorizontal(lipgloss.Top, right, " ", styles.Installed.Render("INSTALLED"))
		} else {
			right = styles.Installed.Render("INSTALLED")
		}
	}

	rightW := lipgloss.Width(right)
	leftW := width - rightW
	if leftW < lipgloss.Width(left) {
		return lipgloss.JoinVertical(lipgloss.Left, left, right)
	}
	leftBlock := lipgloss.NewStyle().Width(leftW).Render(left)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, right)
}

// renderMeta - два столбца с Publisher/Identifier/Category/Trust
func (m Model) renderMeta(width int) string {
	p := m.plugin
	publisher := m.summary.PublisherName
	identifier := m.summary.Identifier
	category := m.summary.Category
	trust := m.summary.TrustStatus
	if p != nil {
		if p.PublisherName != "" {
			publisher = p.PublisherName
		}
		identifier = p.Identifier
		category = p.Category
		trust = p.TrustStatus
	}

	col1 := []string{
		labelRow("Publisher", publisher),
		labelRow("Identifier", identifier),
	}
	col2 := []string{
		labelRow("Category", category),
		labelRow("Trust", trust),
	}
	return joinColumns(col1, col2, width)
}

// renderDescription - описание плагина, ширина по width
func (m Model) renderDescription(width int) string {
	text := m.summary.Description
	if m.plugin != nil && m.plugin.Description != "" {
		text = m.plugin.Description
	}
	if strings.TrimSpace(text) == "" {
		text = "No description."
	}
	header := styles.Subtle.Render("Description:")
	body := lipgloss.NewStyle().Width(width).Foreground(styles.ColorInk).Render(text)
	return header + "\n" + body
}

// renderSelectors рисует два бокса с версией и архитектурой
func (m Model) renderSelectors() string {
	versionBox := m.renderSelector("Version", m.currentVersion(),
		len(m.releases) > 1, m.focus == focusVersion)
	archBox := m.renderSelector("Architecture", m.currentPlatform(),
		len(m.artifacts) > 1, m.focus == focusArch)
	return lipgloss.JoinHorizontal(lipgloss.Top, versionBox, "  ", archBox)
}

// renderSelector - один селектор "< value >" с подсказкой выбора при фокусе
func (m Model) renderSelector(label, value string, hasChoices, focused bool) string {
	if value == "" {
		value = "-"
	}
	arrow := "  "
	if hasChoices {
		arrow = "<>"
	}

	style := styles.Card.Width(22)
	if focused {
		style = styles.SelectedCard.Width(22)
	}

	content := styles.Subtle.Render(label) + "\n" +
		arrowLine(value, arrow, focused)
	return style.Render(content)
}

func arrowLine(value, arrow string, focused bool) string {
	if !focused {
		return "  " + value
	}
	if len(arrow) < 2 {
		return "  " + value
	}
	return string(arrow[0]) + " " + value + " " + string(arrow[1])
}

func (m Model) currentVersion() string {
	if len(m.releases) == 0 {
		if m.loadingReleases {
			return "loading..."
		}
		return "no releases"
	}
	return m.releases[m.releaseIdx].Version
}

func (m Model) currentPlatform() string {
	if len(m.artifacts) == 0 {
		if m.loadingArtifacts {
			return "loading..."
		}
		return "no artifacts"
	}
	a := m.artifacts[m.artifactIdx]
	return a.OS + "/" + a.Arch
}

// renderReleaseAndArtifact - две колонки: данные релиза и данные артефакта
func (m Model) renderReleaseAndArtifact(width int) string {
	release := market.Release{}
	if len(m.releases) > 0 {
		release = m.releases[m.releaseIdx]
	}
	artifact := market.Artifact{}
	if len(m.artifacts) > 0 {
		artifact = m.artifacts[m.artifactIdx]
	}

	col1 := []string{
		styles.Subtle.Render("Release info:"),
		labelRow("  Published", release.PublishedAt),
		labelRow("  Min CLI", release.MinCLIVersion),
		labelRow("  Min K8s", release.MinK8sVersion),
	}
	col2 := []string{
		styles.Subtle.Render("Artifact info:"),
		labelRow("  Size", shared.FormatBytes(artifact.Size)),
		labelRow("  Checksum", shared.ShortChecksum(artifact.Checksum)),
		labelRow("  Type", artifact.Type),
	}
	return joinColumns(col1, col2, width)
}

func (m Model) renderChangelog(width int) string {
	if len(m.releases) == 0 {
		return ""
	}
	text := strings.TrimSpace(m.releases[m.releaseIdx].Changelog)
	if text == "" {
		return ""
	}
	header := styles.Subtle.Render("Changelog:")
	body := lipgloss.NewStyle().Width(width).Foreground(styles.ColorInk).Render(text)
	return header + "\n" + body
}

// renderButtons - три кнопки внизу. Лейблы вычисляются по текущему state
func (m Model) renderButtons() string {
	labels := []string{
		m.libraryButtonLabel(),
		m.artifactButtonLabel(),
		m.verifyButtonLabel(),
	}
	rendered := make([]string, 0, len(labels))
	for i, label := range labels {
		style := styles.Button
		if m.focus == focusButtons && i == m.buttonCursor {
			style = styles.ActiveButton
		}
		rendered = append(rendered, style.Render(label))
	}
	return strings.Join(rendered, "  ")
}

// labelRow - "Label: value" с серым лейблом
func labelRow(label, value string) string {
	if strings.TrimSpace(value) == "" {
		value = "-"
	}
	return styles.Subtle.Render(label+":") + " " + value
}

// joinColumns склеивает две колонки в одну строку, выравнивая по высоте
func joinColumns(left, right []string, width int) string {
	colWidth := width / 2
	if colWidth < 16 {
		colWidth = 16
	}
	leftBlock := lipgloss.NewStyle().Width(colWidth).Render(strings.Join(left, "\n"))
	rightBlock := lipgloss.NewStyle().Width(colWidth).Render(strings.Join(right, "\n"))
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock)
}

// Help возвращает помощь по клавишам для отображения в footer
func (m Model) Help() string {
	return "q quit | esc back | ↑/↓ focus zone | ←/→ change/buttons | enter activate"
}
