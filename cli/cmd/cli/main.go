package main

import (
	"fmt"
	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	authService := auth.NewService(auth.LoadConfig())
	marketService := market.NewService(market.LoadConfig(), authService)
	p := tea.NewProgram(model.New(authService, marketService), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
