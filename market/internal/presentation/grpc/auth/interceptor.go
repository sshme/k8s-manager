package auth

import (
	"context"
	"errors"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenParser func(ctx context.Context, token string) (*Claims, error)

func UnaryAuthInterceptor(rules map[string]Rule, parseToken TokenParser) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		rule, ok := rules[info.FullMethod]
		if !ok {
			rule = Rule{} // по умолчанию: нужен просто валидный пользователь
		}

		token, err := ExtractBearerToken(ctx)
		if err != nil {
			if rule.Public {
				return handler(ctx, req)
			}
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}

		claims, err := parseToken(ctx, token)
		log.Println(claims)
		if err != nil {
			if rule.Public {
				return handler(ctx, req)
			}
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = WithClaims(ctx, claims)

		if !rule.Public && !hasRequiredRole(claims.Roles, rule.Roles) {
			return nil, status.Error(codes.PermissionDenied, "insufficient role")
		}

		return handler(ctx, req)
	}
}
