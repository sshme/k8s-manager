package list

import (
	"k8s-manager/cli/internal/model/tabs"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKey распределяет нажатие в зависимости от того, какой элемент сейчас
// в фокусе.
func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.focus == focusSearch {
		return m.handleSearchKey(msg)
	}
	return m.handleListKey(msg)
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.focus = focusList
		m.search.Blur()
		return m, tabs.SetStatus("Search blurred")
	case "enter":
		m.focus = focusList
		m.search.Blur()
		return m.startLoad()
	}

	// Пробрасываем символ в textinput и сразу отправляем запрос с новой строкой
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	loadedModel, loadCmd := m.startLoad()
	if cmd != nil {
		return loadedModel, tea.Batch(cmd, loadCmd)
	}
	return loadedModel, loadCmd
}

func (m Model) handleListKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "/":
		m.focus = focusSearch
		m.search.Focus()
		return m, tabs.SetStatus("Plugin search")
	case "r":
		// Refresh - перезагрузить с тем же запросом
		return m.refreshAction()
	case "c":
		// Create - открыть wizard
		return m.createAction()
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
			m = m.refresh()
		}
		return m, nil
	case "down", "j":
		if m.listCursor < len(m.items)-1 {
			m.listCursor++
			m = m.refresh()
		}
		return m, nil
	case "enter":
		return m, m.openDetail()
	}
	return m, nil
}
