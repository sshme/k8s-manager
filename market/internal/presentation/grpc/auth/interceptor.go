package auth

import (
	"context"
	"errors"

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
		if err != nil {
			if rule.Public {
				return handler(ctx, req)
			}
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = WithClaims(ctx, claims)

		return handler(ctx, req)
	}
}

func StreamAuthInterceptor(rules map[string]Rule, parseToken TokenParser) grpc.StreamServerInterceptor {
	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		rule, ok := rules[info.FullMethod]
		if !ok {
			rule = Rule{}
		}

		ctx := stream.Context()
		token, err := ExtractBearerToken(ctx)
		if err != nil {
			if rule.Public {
				return handler(srv, stream)
			}
			return status.Error(codes.Unauthenticated, "missing token")
		}

		claims, err := parseToken(ctx, token)
		if err != nil {
			if rule.Public {
				return handler(srv, stream)
			}
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(srv, &authServerStream{
			ServerStream: stream,
			ctx:          WithClaims(ctx, claims),
		})
	}
}

type authServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authServerStream) Context() context.Context {
	return s.ctx
}
