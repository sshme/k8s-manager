package auth

import "context"

type Claims struct {
	UserID   string
	Email    string
	Username string
	Roles    []string
}

type contextKey string

const claimsKey contextKey = "claims"

func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func GetClaims(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

func (c *Claims) HasAnyRole(roles ...string) bool {
	if c == nil {
		return false
	}
	if len(roles) == 0 {
		return true
	}

	userRoles := make(map[string]struct{}, len(c.Roles))
	for _, role := range c.Roles {
		userRoles[role] = struct{}{}
	}
	for _, role := range roles {
		if _, ok := userRoles[role]; ok {
			return true
		}
	}

	return false
}
