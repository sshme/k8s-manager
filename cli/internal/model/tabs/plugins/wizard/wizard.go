// Package wizard - мастер создания плагина.
package wizard

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/tabs"

	tea "github.com/charmbracelet/bubbletea"
)

type focusState int

const (
	focusInput focusState = iota
	focusButtons
)

// индексы шагов в массиве полей
const (
	fieldPublisherName = iota
	fieldIdentifier
	fieldPluginName
	fieldDescription
	fieldCategory
	fieldVersion
	fieldZipPath
	fieldOS
	fieldArch
)

type field struct {
	label        string
	value        string
	placeholder  string
	defaultValue string
	required     bool
}

// Model - модель мастера
type Model struct {
	market     *market.Service
	open       bool
	step       int
	button     int
	focus      focusState
	submitting bool
	fields     []field
}

// New создаёт пустой мастер. Чтобы показать его, нужно вызвать Open
func New(market *market.Service) Model {
	return Model{market: market}
}

// Open включает модальный режим и сбрасывает состояние полей
func (m Model) Open() Model {
	m.open = true
	m.step = 0
	m.button = 0
	m.focus = focusInput
	m.submitting = false
	m.fields = defaultFields()
	return m
}

// Cancel закрывает мастер и стирает введённые данные
func (m Model) Cancel() Model {
	return Model{market: m.market}
}

// Active возвращает true, когда мастер видим и должен ловить ввод
func (m Model) Active() bool { return m.open }

// Submitting возвращает true пока идёт сетевой запрос на создание плагина
func (m Model) Submitting() bool { return m.submitting }

func defaultFields() []field {
	return []field{
		{label: "Publisher", placeholder: "acme", required: true},
		{label: "Identifier", placeholder: "acme.example", required: true},
		{label: "Plugin name", placeholder: "Example Plugin", required: true},
		{label: "Description", placeholder: "Short description"},
		{label: "Category", placeholder: "networking"},
		{label: "Version", placeholder: "1.0.0", defaultValue: "1.0.0", required: true},
		{label: "Zip artifact", placeholder: "/path/to/plugin.zip", required: true},
		{label: "OS", placeholder: "empty = current OS"},
		{label: "Arch", placeholder: "empty = current arch"},
	}
}

func (m Model) buttons() []string {
	if m.step == len(m.fields)-1 {
		return []string{"Back", "Create", "Cancel"}
	}
	if m.step == 0 {
		return []string{"Next", "Cancel"}
	}
	return []string{"Back", "Next", "Cancel"}
}

func (m Model) draft() market.DeveloperPluginDraft {
	value := func(i int) string { return strings.TrimSpace(m.fields[i].value) }
	return market.DeveloperPluginDraft{
		PublisherName: value(fieldPublisherName),
		Identifier:    value(fieldIdentifier),
		Name:          value(fieldPluginName),
		Description:   value(fieldDescription),
		Category:      value(fieldCategory),
		Version:       value(fieldVersion),
		ZipPath:       value(fieldZipPath),
		OS:            value(fieldOS),
		Arch:          value(fieldArch),
	}
}

func (m Model) validate() error {
	for i := range m.fields {
		if err := validateField(m.fields[i]); err != nil {
			return err
		}
	}
	return nil
}

func validateField(f field) error {
	if f.required && strings.TrimSpace(f.value) == "" {
		return fmt.Errorf("%s is required", f.label)
	}
	return nil
}

// Update обрабатывает события мастера
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case CreatedMsg:
		m.submitting = false
		if msg.Err != nil {
			return m, tabs.SetStatus(fmt.Sprintf("Plugin creation failed: %v", msg.Err))
		}
		if msg.Plugin == nil {
			return m, tabs.SetStatus("Plugin creation failed: empty response")
		}

		closed := m.Cancel()
		return closed, tabs.SetStatus(fmt.Sprintf(
			"Created plugin #%d release #%d artifact #%d",
			msg.Plugin.PluginID, msg.Plugin.ReleaseID, msg.Plugin.ArtifactID))
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.submitting {
		return m, nil
	}

	switch msg.String() {
	case "esc":
		return m.Cancel(), tabs.SetStatus("Plugin creation canceled")
	case "down":
		m.focus = focusButtons
		return m, nil
	case "up":
		m.focus = focusInput
		return m, nil
	}

	if m.focus == focusButtons {
		switch msg.String() {
		case "left", "h":
			m.button--
			if m.button < 0 {
				m.button = len(m.buttons()) - 1
			}
			return m, nil
		case "right", "l":
			m.button = (m.button + 1) % len(m.buttons())
			return m, nil
		case "enter":
			return m.activateButton()
		}
		return m, nil
	}

	switch msg.String() {
	case "enter":
		return m.advance()
	case "backspace", "delete":
		f := &m.fields[m.step]
		runes := []rune(f.value)
		if len(runes) > 0 {
			f.value = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	if len(msg.Runes) > 0 {
		m.fields[m.step].value += string(msg.Runes)
	}
	return m, nil
}

func (m Model) activateButton() (Model, tea.Cmd) {
	switch m.buttons()[m.button] {
	case "Back":
		if m.step > 0 {
			m.step--
			m.button = 0
			m.focus = focusInput
		}
		return m, nil
	case "Next":
		return m.advance()
	case "Create":
		return m.submit()
	case "Cancel":
		return m.Cancel(), tabs.SetStatus("Plugin creation canceled")
	}
	return m, nil
}

// advance проверяет текущее поле и идёт к следующему. На последнем шаге
// автоматически вызывает submit
func (m Model) advance() (Model, tea.Cmd) {
	m.applyDefault()
	if err := validateField(m.fields[m.step]); err != nil {
		return m, tabs.SetStatus(err.Error())
	}
	if m.step == len(m.fields)-1 {
		return m.submit()
	}
	m.step++
	m.button = 0
	m.focus = focusInput
	return m, nil
}

func (m Model) applyDefault() {
	f := &m.fields[m.step]
	if strings.TrimSpace(f.value) == "" && f.defaultValue != "" {
		f.value = f.defaultValue
	}
}

func (m Model) submit() (Model, tea.Cmd) {
	if err := m.validate(); err != nil {
		return m, tabs.SetStatus(err.Error())
	}
	m.submitting = true
	return m, tea.Batch(
		tabs.SetStatus("Creating plugin, release, and artifact..."),
		createCmd(m.market, m.draft()),
	)
}
