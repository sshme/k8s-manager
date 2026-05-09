package tabs

import (
	tea "github.com/charmbracelet/bubbletea"
)

type ServicesTab struct{}

func NewServices() ServicesTab { return ServicesTab{} }

func (ServicesTab) Title() string                       { return "Services" }
func (ServicesTab) Init() tea.Cmd                       { return nil }
func (t ServicesTab) Update(tea.Msg) (Tab, tea.Cmd)     { return t, nil }
func (ServicesTab) View(width, height int) string       { return "Selected tab: Services\n\nMain workspace." }
func (ServicesTab) Help() string                        { return "q quit | tab tabs | j/k move" }
func (ServicesTab) CapturingInput() bool                { return false }
