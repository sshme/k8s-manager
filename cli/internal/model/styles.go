package model

import "github.com/charmbracelet/lipgloss"

var (
	colorInk      = lipgloss.Color("#D6DEEB")
	colorMuted    = lipgloss.Color("#7D8799")
	colorTeal     = lipgloss.Color("#2DD4BF")
	colorBlue     = lipgloss.Color("#60A5FA")
	colorGreen    = lipgloss.Color("#8BD450")
	colorAmber    = lipgloss.Color("#FBBF24")
	colorRose     = lipgloss.Color("#F87171")
	colorPanel    = lipgloss.Color("#151A21")
	colorPanelAlt = lipgloss.Color("#1D2430")
	colorBorder   = lipgloss.Color("#334155")

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorInk).
			Background(colorPanelAlt).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorTeal)

	sidebarStyle = lipgloss.NewStyle().
			Padding(0, 0, 0, 1).
			Border(lipgloss.RoundedBorder(), false, true, false, false).
			BorderForeground(colorBorder).
			Height(0)

	contentStyle = lipgloss.NewStyle().
			Padding(1).
			Foreground(colorInk)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(colorBorder)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorTeal)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTeal)

	subtleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	searchStyle = lipgloss.NewStyle().
			Foreground(colorBlue)

	activeButtonStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#0B1220")).
				Background(colorTeal).
				Padding(0, 1)

	buttonStyle = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorPanelAlt).
			Padding(0, 1)

	installedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#102015")).
			Background(colorGreen).
			Padding(0, 1)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorAmber)

	dangerStyle = lipgloss.NewStyle().
			Foreground(colorRose)
)
