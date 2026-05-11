package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

type TokenClaims struct {
	Issuer            string `json:"iss"`
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

type OIDCClient struct {
	issuerURL       string
	clientID        string
	insecureSkipTLS bool
	tokenIssuers    []string

	mu       sync.RWMutex
	verifier *oidc.IDTokenVerifier
}

func NewOIDCClient(issuerURL, clientID string, insecureSkipTLS bool, tokenIssuers ...string) *OIDCClient {
	issuers := make([]string, 0, len(tokenIssuers)+1)
	issuers = append(issuers, issuerURL)
	for _, issuer := range tokenIssuers {
		if issuer != "" && !slices.Contains(issuers, issuer) {
			issuers = append(issuers, issuer)
		}
	}

	return &OIDCClient{
		issuerURL:       issuerURL,
		clientID:        clientID,
		insecureSkipTLS: insecureSkipTLS,
		tokenIssuers:    issuers,
	}
}

func (c *OIDCClient) ValidateToken(ctx context.Context, rawToken string) (*TokenClaims, error) {
	verifier, err := c.verifierFor(ctx)
	if err != nil {
		return nil, err
	}

	token, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	var claims TokenClaims
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	if !slices.Contains(c.tokenIssuers, claims.Issuer) {
		return nil, fmt.Errorf("unexpected token issuer %q", claims.Issuer)
	}

	return &claims, nil
}

func (c *OIDCClient) verifierFor(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	c.mu.RLock()
	verifier := c.verifier
	c.mu.RUnlock()
	if verifier != nil {
		return verifier, nil
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	if c.insecureSkipTLS {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, httpClient), c.issuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	verifier = provider.Verifier(&oidc.Config{
		ClientID:          c.clientID,
		SkipClientIDCheck: true,
		SkipIssuerCheck:   len(c.tokenIssuers) > 1,
	})

	c.mu.Lock()
	c.verifier = verifier
	c.mu.Unlock()

	return verifier, nil
}
