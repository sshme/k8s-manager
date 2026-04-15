package auth

import (
	"context"
	"k8s-manager/market/internal/infrastructure/auth"
	"log"
)

func NewOIDCTokenParser(oidc *auth.OIDCClient) TokenParser {
	return func(ctx context.Context, token string) (*Claims, error) {
		tc, err := oidc.ValidateToken(ctx, token)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		return &Claims{
			UserID: tc.Sub,
			Email:  tc.Email,
			Roles:  tc.Roles,
		}, nil
	}
}
