package auth

import "time"

type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
}

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

func (s *Session) NeedsRefresh() bool {
	if s == nil {
		return true
	}

	if s.Expiry.IsZero() {
		return false
	}

	return time.Now().After(s.Expiry.Add(-30 * time.Second))
}
