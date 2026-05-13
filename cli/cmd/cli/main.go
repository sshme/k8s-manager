package main

import (
	"fmt"
	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model"
	"k8s-manager/cli/internal/plugins"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	authService := auth.NewService(auth.LoadConfig())
	marketService := market.NewService(market.LoadConfig(), authService)

	pluginsMgr, err := plugins.NewManager(plugins.LoadConfig(), marketService)
	if err != nil {
		fmt.Println("Failed to init plugins manager:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model.New(authService, marketService, pluginsMgr), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
