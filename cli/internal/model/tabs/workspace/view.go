package workspace

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/model/styles"
	pluginsmgr "k8s-manager/cli/internal/plugins"

	"github.com/charmbracelet/lipgloss"
)

// Высота одного установленного плагина 2 строки контента + рамка
const installedCardHeight = 4

func (t *Tab) View(width, height int) string {
	if t.dialog.Active() {
		return t.dialog.View(width, height)
	}

	header := styles.Title.Render("Workspace")
	chips := t.renderChips(width)

	// Когда селектор открыт, под ним показываем выбор вместо списка плагинов
	if t.ctxSelector.IsOpen() || t.nsSelector.IsOpen() || t.deplSelector.IsOpen() {
		popupArea := t.renderPopupArea(width, height-lipgloss.Height(header)-lipgloss.Height(chips)-2)
		out := lipgloss.JoinVertical(lipgloss.Left, header, "", chips, "", popupArea)
		return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(out)
	}

	listTitle := styles.Search.Render("Installed Plugins")
	top := lipgloss.JoinVertical(lipgloss.Left, header, "", chips, "", listTitle)

	listHeight := height - lipgloss.Height(top)
	if listHeight < 1 {
		listHeight = 1
	}
	if !t.viewportInit || t.viewport.Width != width {
		t.viewport.Width = width
		t.viewportInit = true
	}
	t.viewport.Height = listHeight
	t.viewport.SetContent(t.renderInstalled(width))
	t.ensureCursorVisible()

	var body string
	switch {
	case len(t.installedItems) == 0:
		body = styles.Subtle.Render("No installed plugins. Install one from the Plugins tab.")
	default:
		body = t.viewport.View()
	}

	out := lipgloss.JoinVertical(lipgloss.Left, top, body)
	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(out)
}

func (t *Tab) renderChips(width int) string {
	chipW := chipWidth(width)
	ctx := t.ctxSelector.ViewChip(t.focus == focusContext, chipW)
	ns := t.nsSelector.ViewChip(t.focus == focusNamespace, chipW)
	depl := t.deplSelector.ViewChip(t.focus == focusDeployment, chipW)
	gap := "  "
	row := lipgloss.JoinHorizontal(lipgloss.Top, ctx, gap, ns, gap, depl)
	if lipgloss.Width(row) > width {
		return lipgloss.JoinVertical(lipgloss.Left, ctx, ns, depl)
	}
	return row
}

func chipWidth(total int) int {
	w := total / 3
	if w < 20 {
		return 20
	}
	return w
}

// renderPopupArea рендерит список активного селектора с отступом, чтобы он
// визуально привязывался к своей кнопке
func (t *Tab) renderPopupArea(width, available int) string {
	maxH := available
	if maxH < 6 {
		maxH = 6
	}
	if maxH > 16 {
		maxH = 16
	}

	var popup string
	var column int
	switch {
	case t.ctxSelector.IsOpen():
		popup = t.ctxSelector.ViewPopup(maxH)
		column = 0
	case t.nsSelector.IsOpen():
		popup = t.nsSelector.ViewPopup(maxH)
		column = 1
	case t.deplSelector.IsOpen():
		popup = t.deplSelector.ViewPopup(maxH)
		column = 2
	}
	if popup == "" {
		return ""
	}
	leftPad := column * (chipWidth(width) + 2)
	if leftPad+lipgloss.Width(popup) > width {
		leftPad = max(0, width-lipgloss.Width(popup)-1)
	}
	return lipgloss.NewStyle().MarginLeft(leftPad).Render(popup)
}

func (t *Tab) renderInstalled(width int) string {
	if len(t.installedItems) == 0 {
		return ""
	}
	cards := make([]string, 0, len(t.installedItems))
	for i, item := range t.installedItems {
		cards = append(cards, renderInstalledCard(item, i == t.listCursor && t.focus == focusList, width))
	}
	return lipgloss.JoinVertical(lipgloss.Left, cards...)
}

func renderInstalledCard(a pluginsmgr.InstalledArtifact, selected bool, width int) string {
	style := styles.Card
	if selected {
		style = styles.SelectedCard
	}
	frameW, _ := style.GetFrameSize()
	inner := width - frameW
	if inner < 10 {
		inner = 10
	}

	fill := lipgloss.NewStyle().
		Background(style.GetBackground()).
		Foreground(style.GetForeground()).
		Bold(style.GetBold())

	left := fill.Render(a.PluginName)
	right := fill.Foreground(styles.ColorMuted).Render(
		strings.TrimSpace(fmt.Sprintf("%s · %s", a.PublisherName, a.Version)))
	header := layoutLine(left, right, inner, fill)

	metaText := truncate(fmt.Sprintf("%s/%s · %s", a.OS, a.Arch, a.PluginIdentifier), inner)
	meta := fill.Foreground(styles.ColorMuted).Render(metaText)
	metaGap := inner - lipgloss.Width(meta)
	if metaGap > 0 {
		meta += fill.Render(strings.Repeat(" ", metaGap))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, meta)
	return style.Width(width - style.GetHorizontalBorderSize()).Render(content)
}

func layoutLine(left, right string, width int, fill lipgloss.Style) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if lw+rw+1 > width {
		left = fill.Render(truncate(left, width-rw-1))
		lw = lipgloss.Width(left)
	}
	gap := width - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + fill.Render(strings.Repeat(" ", gap)) + right
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	runes := []rune(s)
	return string(runes[:max-3]) + "..."
}

func (t *Tab) ensureCursorVisible() {
	if len(t.installedItems) == 0 || !t.viewportInit {
		return
	}
	cursorTop := t.listCursor * installedCardHeight
	cursorBottom := cursorTop + installedCardHeight

	if cursorTop < t.viewport.YOffset {
		t.viewport.SetYOffset(cursorTop)
	} else if cursorBottom > t.viewport.YOffset+t.viewport.Height {
		t.viewport.SetYOffset(cursorBottom - t.viewport.Height)
	}
}
