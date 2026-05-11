package profile

import (
	"fmt"
	"strings"
	"time"

	"k8s-manager/cli/internal/model/styles"
)

func (t *ProfileTab) View(width, height int) string {
	lines := []string{
		styles.Title.Render(t.Title()),
		"",
	}

	if t.session != nil {
		lines = append(lines,
			"Signed in as "+styles.SelectedItem.Render(t.session.DisplayName()),
			"",
			styles.Subtle.Render(fmt.Sprintf("  user-id : %s", fallback(t.session.UserID, "-"))),
			styles.Subtle.Render(fmt.Sprintf("  email   : %s", fallback(t.session.Email, "-"))),
			styles.Subtle.Render(fmt.Sprintf("  roles   : %s", fallback(formatRoles(t.session.Roles), "-"))),
			styles.Subtle.Render(fmt.Sprintf("  expires : %s", formatExpiry(t.session.Expiry))),
		)
	} else {
		lines = append(lines, styles.Subtle.Render("Not signed in."))
	}

	lines = append(lines, "", t.renderButtons())

	if t.authInProgress {
		lines = append(lines, "", styles.Warning.Render("Authorizing with Keycloak..."))
	}

	return strings.Join(lines, "\n")
}

// renderButtons рендерит кнопки из набора доступных действий
func (t *ProfileTab) renderButtons() string {
	actions := t.availableActions()
	rendered := make([]string, 0, len(actions))
	for i, a := range actions {
		style := styles.Button
		if i == t.cursor && !t.authInProgress {
			style = styles.ActiveButton
		}
		rendered = append(rendered, style.Render(a.label))
	}
	return strings.Join(rendered, "  ")
}

// fallback возвращает def, если value пустое после Trim
func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

func formatRoles(roles []string) string {
	cleaned := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" || contains(cleaned, role) {
			continue
		}
		cleaned = append(cleaned, role)
	}
	return strings.Join(cleaned, ", ")
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// formatExpiry рисует время истечения сессии по RFC3339
func formatExpiry(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Until(t)
	if d < 0 {
		return t.Format(time.RFC3339) + " (expired)"
	}
	return fmt.Sprintf("%s (in %s)", t.Format(time.RFC3339), d.Truncate(time.Second))
}
