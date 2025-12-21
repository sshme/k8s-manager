package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

// TokenClaims represents JWT token claims
type TokenClaims struct {
	Sub   string   `json:"sub"`   // User ID
	Email string   `json:"email"`
	Roles []string `json:"roles"`
	Exp   int64    `json:"exp"`
	Iat   int64    `json:"iat"`
}

// OIDCClient handles OIDC token validation
type OIDCClient struct {
	issuerURL    string
	clientID     string
	httpClient   *http.Client
	skipVerify   bool // For development/testing
}

// NewOIDCClient creates a new OIDC client
func NewOIDCClient(issuerURL, clientID string, skipVerify bool) *OIDCClient {
	return &OIDCClient{
		issuerURL:  issuerURL,
		clientID:   clientID,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		skipVerify: skipVerify,
	}
}

// ValidateToken validates an OIDC/OAuth 2.0 token
// In production, this would:
// 1. Validate token signature using JWKS from issuer
// 2. Check token expiration
// 3. Verify issuer and audience
// For now, we'll do basic validation and extract claims
func (c *OIDCClient) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	if c.skipVerify {
		// Development mode: parse token as JWT and extract claims
		// In production, use a proper JWT library like github.com/golang-jwt/jwt/v5
		return c.parseTokenClaims(token)
	}
	
	// Production mode: validate with OIDC provider
	// This would typically involve:
	// 1. Calling the introspection endpoint
	// 2. Or validating JWT signature with JWKS
	return c.validateWithProvider(ctx, token)
}

// parseTokenClaims parses JWT token claims (simplified for development)
func (c *OIDCClient) parseTokenClaims(token string) (*TokenClaims, error) {
	// JWT format: header.payload.signature
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	
	// Decode payload (base64url)
	payload := parts[1]
	// Add padding if needed
	for len(payload)%4 != 0 {
		payload += "="
	}
	
	// Decode base64url
	decoded, err := base64URLDecode(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}
	
	var claims TokenClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}
	
	// Check expiration
	if claims.Exp > 0 {
		expTime := time.Unix(claims.Exp, 0)
		if time.Now().After(expTime) {
			return nil, ErrTokenExpired
		}
	}
	
	return &claims, nil
}

// validateWithProvider validates token with OIDC provider
func (c *OIDCClient) validateWithProvider(ctx context.Context, token string) (*TokenClaims, error) {
	// In production, implement actual OIDC validation
	// For now, fall back to parsing
	return c.parseTokenClaims(token)
}

// base64URLDecode decodes base64url encoded string
func base64URLDecode(s string) ([]byte, error) {
	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	
	// Add padding
	for len(s)%4 != 0 {
		s += "="
	}
	
	// Use standard base64 decoding
	return base64.StdEncoding.DecodeString(s)
}

// GetUserIDFromToken extracts user ID from token
func (c *OIDCClient) GetUserIDFromToken(ctx context.Context, token string) (string, error) {
	claims, err := c.ValidateToken(ctx, token)
	if err != nil {
		return "", err
	}
	return claims.Sub, nil
}

// GetUserRolesFromToken extracts user roles from token
func (c *OIDCClient) GetUserRolesFromToken(ctx context.Context, token string) ([]string, error) {
	claims, err := c.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return claims.Roles, nil
}

