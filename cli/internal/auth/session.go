package auth

import "time"

// Session - информация о сессии после успешного OIDC
type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"` // для access токена
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
}

// DisplayName - имя для отображение в UI. Берётся первое непустое из
// username > email > UserID. В случае пустой сессии отдаётся пустая строка.
func (s *Session) DisplayName() string {
	if s == nil {
		return ""
	}

	if s.Username != "" {
		return s.Username
	}

	if s.Email != "" {
		return s.Email
	}

	return s.UserID
}

// NeedsRefresh говорит, пора ли менять access. Считаем что пора уже с
// запасом в 30 секунд до фактического истечения
func (s *Session) NeedsRefresh() bool {
	if s == nil {
		return true
	}

	if s.Expiry.IsZero() {
		return false
	}

	return time.Now().After(s.Expiry.Add(-30 * time.Second))
}
