// Package shared собирает утилиты, переиспользуемые между подпакетами
// plugins/list, plugins/detail и plugins/wizard
package shared

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// NonEmpty фильтрует строки: пустые и "UNKNOWN" убирает, остальные триммит
func NonEmpty(values ...string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "UNKNOWN" {
			items = append(items, value)
		}
	}
	return items
}

// Truncate обрезает строку по ширине терминала с многоточием.
// Многострочные значения сначала склеиваются в одну строку через пробел
func Truncate(value string, maxWidth int) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if lipgloss.Width(value) <= maxWidth {
		return value
	}

	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxWidth-3 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

// ShortChecksum обрезает sha256 до первых 12 символов для компактного вывода
func ShortChecksum(checksum string) string {
	if len(checksum) <= 12 {
		return checksum
	}
	return checksum[:12]
}

// FormatBytes преобразует количество байт в "5.4 MB", "123 KB" и т.д.
func FormatBytes(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n >= gb:
		return formatFloat(float64(n)/float64(gb)) + " GB"
	case n >= mb:
		return formatFloat(float64(n)/float64(mb)) + " MB"
	case n >= kb:
		return formatFloat(float64(n)/float64(kb)) + " KB"
	default:
		return formatInt(n) + " B"
	}
}

func formatFloat(f float64) string {
	// одна цифра после запятой
	intPart := int64(f)
	frac := int64((f - float64(intPart)) * 10)
	if frac < 0 {
		frac = -frac
	}
	return formatInt(intPart) + "." + formatInt(frac)
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
