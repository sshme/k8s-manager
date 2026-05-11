package auth

import (
	"context"
	"k8s-manager/market/internal/infrastructure/auth"
	"log"
	"slices"
)

func NewOIDCTokenParser(oidc *auth.OIDCClient) TokenParser {
	return func(ctx context.Context, token string) (*Claims, error) {
		tc, err := oidc.ValidateToken(ctx, token)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		return &Claims{
			UserID:   tc.Sub,
			Email:    tc.Email,
			Username: tc.PreferredUsername,
			Roles:    collectRoles(tc),
		}, nil
	}
}

func collectRoles(tc *auth.TokenClaims) []string {
	if tc == nil {
		return nil
	}

	roles := make([]string, 0, len(tc.RealmAccess.Roles))
	for _, role := range tc.RealmAccess.Roles {
		if !slices.Contains(roles, role) {
			roles = append(roles, role)
		}
	}
	for _, access := range tc.ResourceAccess {
		for _, role := range access.Roles {
			if !slices.Contains(roles, role) {
				roles = append(roles, role)
			}
		}
	}

	return roles
}
