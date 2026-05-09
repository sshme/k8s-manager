package styles

import "github.com/charmbracelet/lipgloss"

var (
	ColorInk      = lipgloss.Color("#D6DEEB")
	ColorMuted    = lipgloss.Color("#7D8799")
	ColorTeal     = lipgloss.Color("#2DD4BF")
	ColorBlue     = lipgloss.Color("#60A5FA")
	ColorGreen    = lipgloss.Color("#8BD450")
	ColorAmber    = lipgloss.Color("#FBBF24")
	ColorRose     = lipgloss.Color("#F87171")
	ColorPanel    = lipgloss.Color("#151A21")
	ColorPanelAlt = lipgloss.Color("#1D2430")
	ColorBorder   = lipgloss.Color("#334155")

	Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorInk).
		Background(ColorPanelAlt).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ColorTeal)

	Sidebar = lipgloss.NewStyle().
		Padding(0, 0, 0, 1).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(ColorBorder).
		Height(0)

	Content = lipgloss.NewStyle().
		Padding(1).
		Foreground(ColorInk)

	Footer = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(ColorBorder)

	SelectedItem = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorTeal)

	NormalItem = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorTeal)

	Subtle = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Search = lipgloss.NewStyle().
		Foreground(ColorBlue)

	ActiveButton = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B1220")).
		Background(ColorTeal).
		Padding(0, 1)

	Button = lipgloss.NewStyle().
		Foreground(ColorInk).
		Background(ColorPanelAlt).
		Padding(0, 1)

	Installed = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#102015")).
		Background(ColorGreen).
		Padding(0, 1)

	Warning = lipgloss.NewStyle().
		Foreground(ColorAmber)

	Danger = lipgloss.NewStyle().
		Foreground(ColorRose)
)
