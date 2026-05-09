package plugins

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/model/styles"
)

func (t *PluginsTab) View(width, height int) string {
	if t.wizard.open {
		return t.renderWizardModal(width, height)
	}
	return strings.Join(t.renderContent(), "\n")
}

func (t *PluginsTab) renderContent() []string {
	focus := " "
	if t.focused {
		focus = ">"
	}

	mode := "installed"
	if strings.TrimSpace(t.search) != "" {
		mode = "search"
	}

	lines := []string{
		styles.Title.Render(t.Title()),
		"",
		t.renderButtons(),
		"",
		styles.Search.Render(fmt.Sprintf("%s Search: %s", focus, t.search)),
		styles.Subtle.Render(fmt.Sprintf("Mode: %s", mode)),
		"",
	}

	if t.loading {
		return append(lines, styles.Warning.Render("Loading plugins..."))
	}

	if !t.loaded {
		return append(lines, styles.Subtle.Render("Press Enter to load plugins."))
	}

	if len(t.items) == 0 {
		if t.source == "installed" {
			return append(lines, styles.Subtle.Render("No installed plugins."))
		}
		return append(lines, styles.Subtle.Render("No plugins found."))
	}

	lines = append(lines, styles.Subtle.Render(fmt.Sprintf("Total: %d", t.total)), "")
	for i, plugin := range t.items {
		prefix := "  "
		lineStyle := styles.NormalItem
		if i == t.listCursor {
			prefix = "> "
			lineStyle = styles.SelectedItem
		}
		title := fmt.Sprintf("#%d  %s", plugin.ID, plugin.Name)
		if plugin.Identifier != "" {
			title += fmt.Sprintf(" (%s)", plugin.Identifier)
		}
		meta := strings.Trim(strings.Join(nonEmpty(plugin.Category, plugin.TrustStatus, plugin.Status), " | "), " | ")
		if plugin.Installed && t.source == "search" {
			meta = strings.Trim(strings.Join(nonEmpty(meta, "INSTALLED"), " | "), " | ")
		}
		lines = append(lines, lineStyle.Render(prefix+title))
		if meta != "" {
			lines = append(lines, styles.Subtle.Render("  "+meta))
		}
		if plugin.Description != "" {
			lines = append(lines, styles.Subtle.Render("  "+truncate(plugin.Description, 96)))
		}
		if plugin.InstalledAt != "" && t.source == "search" {
			lines = append(lines, styles.Installed.Render("installed")+" "+styles.Subtle.Render(plugin.InstalledAt))
		}
		lines = append(lines, "")
	}

	return lines
}

func (t *PluginsTab) renderButtons() string {
	rendered := make([]string, 0, len(buttonLabels))
	for i, button := range buttonLabels {
		style := styles.Button
		if i == t.button {
			style = styles.ActiveButton
		}
		rendered = append(rendered, style.Render(button))
	}
	return strings.Join(rendered, " ")
}
