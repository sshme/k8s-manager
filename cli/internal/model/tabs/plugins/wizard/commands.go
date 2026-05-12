package wizard

import (
	"context"
	"fmt"
	"time"

	"k8s-manager/cli/internal/market"

	tea "github.com/charmbracelet/bubbletea"
)

// CreatedMsg - результат публикации плагина мастером
type CreatedMsg struct {
	Plugin *market.DeveloperPlugin
	Err    error
}

func createCmd(svc *market.Service, draft market.DeveloperPluginDraft) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return CreatedMsg{Err: fmt.Errorf("market service is not configured")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		plugin, err := svc.CreateDeveloperPlugin(ctx, draft)
		return CreatedMsg{Plugin: plugin, Err: err}
	}
}
