package list

import (
	"fmt"
	"strings"

	"k8s-manager/cli/internal/market"
	"k8s-manager/cli/internal/model/styles"
	"k8s-manager/cli/internal/model/tabs/plugins/shared"

	"github.com/charmbracelet/lipgloss"
)

// Высота одной карточки.
// 2 строки контента (header + description) + 2 строки на обводку.
const (
	cardContentHeight = 2
	cardHeight        = cardContentHeight + 2

	searchMaxWidth = 60 // Максимальная ширина поиска
)

// SetSize настраивает ширину textinput и вычисляет доступную высоту
// под viewport
func (m Model) SetSize(width, height int) Model {
	m.viewport.Width = width

	frameW, _ := styles.SearchBox.GetFrameSize()
	inner := searchInner(width, frameW)
	m.search.Width = inner

	// Точная высота выставляется в View по фактическим размерам блоков
	if m.viewport.Height == 0 {
		m.viewport.Height = max(height-6, 1)
	}

	m.viewportInit = true
	m.viewport.SetContent(m.renderCards(width))
	m.ensureCursorVisible()
	return m
}

// View собирает главный экран.
func (m Model) View(width, height int) string {
	if !m.viewportInit {
		m = m.SetSize(width, height)
	}

	header := m.renderHeader(width)
	search := m.renderSearch(width)

	totalLine := ""
	if m.loaded && len(m.items) > 0 {
		totalLine = styles.Subtle.Render(fmt.Sprintf("Showing %d of %d", len(m.items), m.total))
	}

	// header + пустая + search + (опционально) total
	topParts := []string{header, "", search}
	if totalLine != "" {
		topParts = append(topParts, totalLine)
	}
	top := lipgloss.JoinVertical(lipgloss.Left, topParts...)

	listHeight := height - lipgloss.Height(top)
	if listHeight < 1 {
		listHeight = 1
	}
	m.viewport.Height = listHeight
	m.viewport.SetContent(m.renderCards(width))
	m.ensureCursorVisible()

	var body string
	switch {
	case m.loading:
		body = styles.Warning.Render("Loading plugins...")
	case !m.loaded:
		body = styles.Subtle.Render("Press / to search or wait for the first load")
	case len(m.items) == 0 && m.source == "installed":
		body = styles.Subtle.Render("Your library is empty. Press / to search the marketplace.")
	case len(m.items) == 0:
		body = styles.Subtle.Render("No plugins found.")
	default:
		body = m.viewport.View()
	}

	result := lipgloss.JoinVertical(lipgloss.Left, top, body)

	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(result)
}

// renderHeader - рендерит хедер. title слева, кнопки cправа
func (m Model) renderHeader(width int) string {
	title := styles.Title.Render("Plugins")
	buttons := m.renderButtons()

	buttonsW := lipgloss.Width(buttons)
	leftW := width - buttonsW
	if leftW < lipgloss.Width(title) {
		return lipgloss.JoinVertical(lipgloss.Left, title, buttons)
	}

	left := lipgloss.NewStyle().Width(leftW).Render(title)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, buttons)
}

// renderButtons возвращает строку с кнопками для перезагрузки и создания плагина
func (m Model) renderButtons() string {
	refresh := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.Subtle.Render("[r] "),
		styles.Button.Render("Refresh"))
	create := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.Subtle.Render("[c] "),
		styles.Button.Render("Create"))
	return lipgloss.JoinHorizontal(lipgloss.Top, refresh, "  ", create)
}

// renderSearch - поле поиска ограниченной ширины
func (m Model) renderSearch(width int) string {
	boxStyle := styles.SearchBox
	if m.focus == focusSearch {
		boxStyle = styles.SearchBoxFocused
	}
	frameW, _ := boxStyle.GetFrameSize()
	inner := searchInner(width, frameW)
	return boxStyle.Width(inner + boxStyle.GetHorizontalPadding()).Render(m.search.View())
}

// searchInner возвращает внутреннюю ширину поля ввода, ограниченное searchMaxWidth.
// frameW - сколько занимают border + padding (одинаково для focused и non-focused)
func searchInner(bodyWidth, frameW int) int {
	maxInner := searchMaxWidth - frameW
	avail := bodyWidth - frameW
	if avail < maxInner {
		return max(avail, 1)
	}
	return maxInner
}

// renderCards склеивает карточки вертикально
func (m Model) renderCards(width int) string {
	if len(m.items) == 0 {
		return ""
	}

	cards := make([]string, 0, len(m.items))
	for i, plugin := range m.items {
		cards = append(cards, m.renderCard(plugin, i == m.listCursor, width))
	}
	return lipgloss.JoinVertical(lipgloss.Left, cards...)
}

// renderCard рендерит одну карточку плагина в списке плагинов
func (m Model) renderCard(plugin market.PluginSummary, selected bool, width int) string {
	cardStyle := styles.Card
	if selected {
		cardStyle = styles.SelectedCard
	}

	fill := lipgloss.NewStyle().
		Background(cardStyle.GetBackground()).
		Foreground(cardStyle.GetForeground()).
		Bold(cardStyle.GetBold())

	frameW, _ := cardStyle.GetFrameSize()
	innerWidth := width - frameW
	if innerWidth < 10 {
		innerWidth = 10
	}

	header := m.renderCardHeader(plugin, innerWidth, fill)
	description := m.renderCardDescription(plugin.Description, innerWidth, fill)

	content := lipgloss.JoinVertical(lipgloss.Left, header, description)

	return cardStyle.Width(width - cardStyle.GetHorizontalBorderSize()).Render(content)
}

// renderCardHeader - имя и бейджи слева, publisher справа.
func (m Model) renderCardHeader(plugin market.PluginSummary, width int, fill lipgloss.Style) string {
	segs := []string{fill.Render(plugin.Name)}
	if badge := trustBadge(plugin.TrustStatus); badge != "" {
		segs = append(segs, fill.Render(" "), badge)
	}
	if plugin.Installed {
		segs = append(segs, fill.Render(" "), styles.Installed.Render("INSTALLED"))
	}
	left := strings.Join(segs, "")

	right := ""
	if plugin.PublisherName != "" {
		right = fill.Foreground(styles.ColorMuted).Render(plugin.PublisherName)
	}
	rightW := lipgloss.Width(right)

	leftW := width - rightW
	if leftW < lipgloss.Width(left) {
		left = fill.Render(shared.Truncate(plugin.Name, width-rightW-1))
	}

	gap := leftW - lipgloss.Width(left)
	if gap < 0 {
		gap = 0
	}
	return left + fill.Render(strings.Repeat(" ", gap)) + right
}

// renderCardDescription - одна строка описания, обрезает если не помещается
func (m Model) renderCardDescription(text string, width int, fill lipgloss.Style) string {
	if strings.TrimSpace(text) == "" {
		text = "No description."
	}
	body := fill.Foreground(styles.ColorMuted).Render(shared.Truncate(text, width))
	gap := width - lipgloss.Width(body)
	if gap > 0 {
		body += fill.Render(strings.Repeat(" ", gap))
	}
	return body
}

func trustBadge(trust string) string {
	switch trust {
	case "VERIFIED":
		return styles.Verified.Render("VERIFIED")
	case "OFFICIAL":
		return styles.Official.Render("OFFICIAL")
	}
	return ""
}

// ensureCursorVisible двигает viewport так, чтобы выделенная карточка
// оказалась в видимой области
func (m *Model) ensureCursorVisible() {
	if len(m.items) == 0 || !m.viewportInit {
		return
	}
	cursorTop := m.listCursor * cardHeight
	cursorBottom := cursorTop + cardHeight

	if cursorTop < m.viewport.YOffset {
		m.viewport.SetYOffset(cursorTop)
	} else if cursorBottom > m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(cursorBottom - m.viewport.Height)
	}
}

// refresh пересобирает контент viewport
func (m Model) refresh() Model {
	if !m.viewportInit {
		return m
	}
	m.viewport.SetContent(m.renderCards(m.viewport.Width))
	m.ensureCursorVisible()
	return m
}
