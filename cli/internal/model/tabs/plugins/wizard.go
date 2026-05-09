package plugins

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/styles"
	"k8s-manager/cli/internal/model/tabs"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type wizardFocus int

const (
	wizardFocusInput wizardFocus = iota
	wizardFocusButtons
)

const (
	wizardPublisherName = iota
	wizardIdentifier
	wizardPluginName
	wizardDescription
	wizardCategory
	wizardVersion
	wizardZipPath
	wizardOS
	wizardArch
)

type wizardField struct {
	label        string
	value        string
	placeholder  string
	defaultValue string
	required     bool
}

type wizard struct {
	open       bool
	step       int
	button     int
	focus      wizardFocus
	submitting bool
	fields     []wizardField
}

func newWizard() wizard {
	return wizard{
		open:  true,
		focus: wizardFocusInput,
		fields: []wizardField{
			{label: "Publisher", placeholder: "acme", required: true},
			{label: "Identifier", placeholder: "acme.example", required: true},
			{label: "Plugin name", placeholder: "Example Plugin", required: true},
			{label: "Description", placeholder: "Short description"},
			{label: "Category", placeholder: "networking"},
			{label: "Version", placeholder: "1.0.0", defaultValue: "1.0.0", required: true},
			{label: "Zip artifact", placeholder: "/path/to/plugin.zip", required: true},
			{label: "OS", placeholder: "empty = current OS"},
			{label: "Arch", placeholder: "empty = current arch"},
		},
	}
}

func (w wizard) buttons() []string {
	if w.step == len(w.fields)-1 {
		return []string{"Back", "Create", "Cancel"}
	}
	if w.step == 0 {
		return []string{"Next", "Cancel"}
	}
	return []string{"Back", "Next", "Cancel"}
}

func (w wizard) draft() market.DeveloperPluginDraft {
	value := func(index int) string {
		return strings.TrimSpace(w.fields[index].value)
	}

	return market.DeveloperPluginDraft{
		PublisherName: value(wizardPublisherName),
		Identifier:    value(wizardIdentifier),
		Name:          value(wizardPluginName),
		Description:   value(wizardDescription),
		Category:      value(wizardCategory),
		Version:       value(wizardVersion),
		ZipPath:       value(wizardZipPath),
		OS:            value(wizardOS),
		Arch:          value(wizardArch),
	}
}

func (w wizard) validate() error {
	for i := range w.fields {
		if err := validateField(w.fields[i]); err != nil {
			return err
		}
	}
	return nil
}

func validateField(f wizardField) error {
	if f.required && strings.TrimSpace(f.value) == "" {
		return fmt.Errorf("%s is required", f.label)
	}
	return nil
}

func (t *PluginsTab) updateWizardKey(msg tea.KeyMsg) (PluginsTab, tea.Cmd) {
	if t.wizard.submitting {
		return *t, nil
	}

	switch msg.String() {
	case "esc":
		t.wizard = wizard{}
		return *t, tabs.SetStatus("Plugin creation canceled")
	case "down":
		t.wizard.focus = wizardFocusButtons
		return *t, nil
	case "up":
		t.wizard.focus = wizardFocusInput
		return *t, nil
	}

	if t.wizard.focus == wizardFocusButtons {
		switch msg.String() {
		case "left", "h":
			t.wizard.button--
			if t.wizard.button < 0 {
				t.wizard.button = len(t.wizard.buttons()) - 1
			}
			return *t, nil
		case "right", "l":
			t.wizard.button = (t.wizard.button + 1) % len(t.wizard.buttons())
			return *t, nil
		case "enter":
			return t.activateWizardButton()
		}
		return *t, nil
	}

	switch msg.String() {
	case "enter":
		return t.advanceWizard()
	case "backspace", "delete":
		field := &t.wizard.fields[t.wizard.step]
		runes := []rune(field.value)
		if len(runes) > 0 {
			field.value = string(runes[:len(runes)-1])
		}
		return *t, nil
	}

	if len(msg.Runes) > 0 {
		field := &t.wizard.fields[t.wizard.step]
		field.value += string(msg.Runes)
	}

	return *t, nil
}

func (t *PluginsTab) activateWizardButton() (PluginsTab, tea.Cmd) {
	switch t.wizard.buttons()[t.wizard.button] {
	case "Back":
		if t.wizard.step > 0 {
			t.wizard.step--
			t.wizard.button = 0
			t.wizard.focus = wizardFocusInput
		}
		return *t, nil
	case "Next":
		return t.advanceWizard()
	case "Create":
		return t.submitWizard()
	case "Cancel":
		t.wizard = wizard{}
		return *t, tabs.SetStatus("Plugin creation canceled")
	}
	return *t, nil
}

func (t *PluginsTab) advanceWizard() (PluginsTab, tea.Cmd) {
	t.applyWizardDefault()
	if err := validateField(t.wizard.fields[t.wizard.step]); err != nil {
		return *t, tabs.SetStatus(err.Error())
	}
	if t.wizard.step == len(t.wizard.fields)-1 {
		return t.submitWizard()
	}
	t.wizard.step++
	t.wizard.button = 0
	t.wizard.focus = wizardFocusInput
	return *t, nil
}

func (t *PluginsTab) applyWizardDefault() {
	field := &t.wizard.fields[t.wizard.step]
	if strings.TrimSpace(field.value) == "" && field.defaultValue != "" {
		field.value = field.defaultValue
	}
}

func (t *PluginsTab) submitWizard() (PluginsTab, tea.Cmd) {
	if err := t.wizard.validate(); err != nil {
		return *t, tabs.SetStatus(err.Error())
	}

	t.wizard.submitting = true
	return *t, tea.Batch(
		tabs.SetStatus("Creating plugin, release, and artifact..."),
		createDeveloperPluginCmd(t.market, t.wizard.draft()),
	)
}

func (t PluginsTab) renderWizardModal(width, height int) string {
	modalWidth := min(max(48, width-8), 76)
	field := t.wizard.fields[t.wizard.step]
	inputPrefix := "  "
	if t.wizard.focus == wizardFocusInput {
		inputPrefix = "> "
	}

	value := field.value
	if value == "" {
		value = styles.Subtle.Render(field.placeholder)
	}

	lines := []string{
		styles.Title.Render(fmt.Sprintf("Create Plugin  %d/%d", t.wizard.step+1, len(t.wizard.fields))),
		"",
		styles.Search.Render(field.label),
		inputPrefix + value,
		"",
		t.renderWizardButtons(),
	}
	if t.wizard.submitting {
		lines = append(lines, "", styles.Warning.Render("Submitting..."))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorTeal).
		Background(styles.ColorPanel).
		Padding(1, 2).
		Width(modalWidth).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (t PluginsTab) renderWizardButtons() string {
	buttons := t.wizard.buttons()
	rendered := make([]string, 0, len(buttons))
	for i, button := range buttons {
		style := styles.Button
		if t.wizard.focus == wizardFocusButtons && i == t.wizard.button {
			style = styles.ActiveButton
		}
		rendered = append(rendered, style.Render(button))
	}
	return strings.Join(rendered, " ")
}
