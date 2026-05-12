package list

import (
	"context"
	"fmt"
	"time"

	"k8s-manager/cli/internal/market"

	tea "github.com/charmbracelet/bubbletea"
)

// loadedMsg - результат ListPlugins
type loadedMsg struct {
	list *market.PluginList
	err  error
}

func (loadedMsg) PluginMsg() {}

// loadCmd обращается к сервису маркетплейса за списком плагинов.
// После ответа от маркетплейса возвращает loadedMsg со списком
// плагинов.
func loadCmd(svc *market.Service, query string) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return loadedMsg{err: fmt.Errorf("market service is not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		list, err := svc.ListPlugins(ctx, query)
		return loadedMsg{list: list, err: err}
	}
}
