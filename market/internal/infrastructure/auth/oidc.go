package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

type TokenClaims struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
}

type OIDCClient struct {
	issuerURL       string
	clientID        string
	insecureSkipTLS bool

	mu       sync.RWMutex
	verifier *oidc.IDTokenVerifier
}

func NewOIDCClient(issuerURL, clientID string, insecureSkipTLS bool) *OIDCClient {
	return &OIDCClient{
		issuerURL:       issuerURL,
		clientID:        clientID,
		insecureSkipTLS: insecureSkipTLS,
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
	})

	c.mu.Lock()
	c.verifier = verifier
	c.mu.Unlock()

	return verifier, nil
}
