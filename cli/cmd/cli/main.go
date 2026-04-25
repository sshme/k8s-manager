package main

import (
	"fmt"
	"k8s-manager/cli/internal/auth"
	"k8s-manager/cli/internal/model"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	authService := auth.NewService(auth.LoadConfig())
	p := tea.NewProgram(model.New(authService), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
