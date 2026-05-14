// Package dialog реализует окно действий для установленного плагина
package dialog

import (
	pluginsmgr "k8s-manager/cli/internal/plugins"

	tea "github.com/charmbracelet/bubbletea"
)

// Action - действие, выбор пользователя
type Action int

const (
	ActionNone Action = iota
	ActionCancel
	ActionDelete
	ActionPlan
	ActionApply
)

const (
	btnDelete = iota
	btnPlan
	btnApply
)

var buttonLabels = []string{"Delete", "Plan", "Apply"}

type Model struct {
	open     bool
	cursor   int
	artifact pluginsmgr.InstalledArtifact
	working  bool
}

func New() Model { return Model{} }

func (m Model) Open(a pluginsmgr.InstalledArtifact) Model {
	m.open = true
	m.cursor = 0
	m.artifact = a
	m.working = false
	return m
}

func (m Model) Cancel() Model { return Model{} }

func (m Model) Active() bool  { return m.open }
func (m Model) Working() bool { return m.working }

// SetWorking блокирует кнопки на время асинхронной операции
func (m Model) SetWorking(v bool) Model {
	m.working = v
	return m
}

func (m Model) Artifact() pluginsmgr.InstalledArtifact { return m.artifact }

// Update обрабатывает клавиши и возвращает Action для родителя.
func (m Model) Update(msg tea.KeyMsg) (Model, Action) {
	if !m.open {
		return m, ActionNone
	}
	if m.working {
		// Пока идёт операция игнорируем всё кроме явной отмены
		if msg.String() == "esc" {
			return m, ActionNone
		}
		return m, ActionNone
	}
	switch msg.String() {
	case "esc":
		return m.Cancel(), ActionCancel
	case "left", "h":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, ActionNone
	case "right", "l":
		if m.cursor < len(buttonLabels)-1 {
			m.cursor++
		}
		return m, ActionNone
	case "enter":
		switch m.cursor {
		case btnDelete:
			return m, ActionDelete
		case btnPlan:
			return m, ActionPlan
		case btnApply:
			return m, ActionApply
		}
	}
	return m, ActionNone
}
