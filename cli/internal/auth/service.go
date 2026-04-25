package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Service struct {
	cfg   Config
	store *Store
}

func NewService(cfg Config) *Service {
	return &Service{
		cfg:   cfg,
		store: NewStore(cfg.KeyringName, cfg.KeyringEntry, cfg.SessionPath),
	}
}

func (s *Service) LoadSession(ctx context.Context) (*Session, error) {
	session, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	if !session.NeedsRefresh() {
		return session, nil
	}

	refreshed, err := s.refreshSession(ctx, session)
	if err != nil {
		_ = s.store.Delete()
		return nil, err
	}

	return refreshed, nil
}

func (s *Service) Authenticate(ctx context.Context) (*Session, error) {
	return s.authorize(ctx, false)
}

func (s *Service) Register(ctx context.Context) (*Session, error) {
	return s.authorize(ctx, true)
}

func (s *Service) authorize(ctx context.Context, register bool) (*Session, error) {
	provider, oauthCfg, err := s.providerConfig(ctx)
	if err != nil {
		return nil, err
	}

	state, err := randomBase64URL(32)
	if err != nil {
		return nil, err
	}

	codeVerifier, err := randomBase64URL(64)
	if err != nil {
		return nil, err
	}

	codeChallenge := pkceChallenge(codeVerifier)

	redirectURL, err := url.Parse(s.cfg.RedirectURL)
	if err != nil {
		return nil, err
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	server, listener, err := s.startCallbackServer(redirectURL, state, codeCh, errCh)
	if err != nil {
		return nil, err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL := oauthCfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	if register {
		authURL, err = registrationURL(authURL)
		if err != nil {
			return nil, err
		}
	}

	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	var code string
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case code = <-codeCh:
	}

	_ = listener.Close()

	token, err := oauthCfg.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange auth code: %w", err)
	}

	session, err := s.sessionFromToken(ctx, provider, oauthCfg, token)
	if err != nil {
		return nil, err
	}

	if err := s.store.Save(session); err != nil {
		return nil, fmt.Errorf("failed to save session to keyring: %w", err)
	}

	return session, nil
}

func (s *Service) AttachToken(ctx context.Context) (string, error) {
	session, err := s.LoadSession(ctx)
	if err != nil {
		return "", err
	}

	return session.AccessToken, nil
}

func (s *Service) providerConfig(ctx context.Context) (*oidc.Provider, *oauth2.Config, error) {
	provider, err := oidc.NewProvider(ctx, s.cfg.IssuerURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to discover oidc provider: %w", err)
	}

	oauthCfg := &oauth2.Config{
		ClientID:    s.cfg.ClientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: s.cfg.RedirectURL,
		Scopes: []string{
			oidc.ScopeOpenID,
			"profile",
			"email",
		},
	}

	return provider, oauthCfg, nil
}

func (s *Service) refreshSession(ctx context.Context, session *Session) (*Session, error) {
	if session == nil || session.RefreshToken == "" {
		return nil, ErrSessionNotFound
	}

	provider, oauthCfg, err := s.providerConfig(ctx)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		TokenType:    session.TokenType,
		Expiry:       session.Expiry,
	}

	refreshed, err := oauthCfg.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	nextSession, err := s.sessionFromToken(ctx, provider, oauthCfg, refreshed)
	if err != nil {
		return nil, err
	}

	if nextSession.RefreshToken == "" {
		nextSession.RefreshToken = session.RefreshToken
	}
	if nextSession.IDToken == "" {
		nextSession.IDToken = session.IDToken
	}
	if nextSession.Username == "" {
		nextSession.Username = session.Username
	}
	if nextSession.Email == "" {
		nextSession.Email = session.Email
	}
	if nextSession.UserID == "" {
		nextSession.UserID = session.UserID
	}

	if err := s.store.Save(nextSession); err != nil {
		return nil, fmt.Errorf("failed to persist refreshed session: %w", err)
	}

	return nextSession, nil
}

func (s *Service) sessionFromToken(ctx context.Context, provider *oidc.Provider, oauthCfg *oauth2.Config, token *oauth2.Token) (*Session, error) {
	session := &Session{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}

	if rawIDToken, ok := token.Extra("id_token").(string); ok {
		session.IDToken = rawIDToken

		verifier := provider.Verifier(&oidc.Config{ClientID: oauthCfg.ClientID})
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			return nil, fmt.Errorf("failed to verify id token: %w", err)
		}

		var claims struct {
			Subject           string `json:"sub"`
			Email             string `json:"email"`
			PreferredUsername string `json:"preferred_username"`
		}
		if err := idToken.Claims(&claims); err != nil {
			return nil, fmt.Errorf("failed to parse id token claims: %w", err)
		}

		session.UserID = claims.Subject
		session.Email = claims.Email
		session.Username = claims.PreferredUsername
	}

	if session.Username != "" || session.Email != "" || session.UserID != "" {
		return session, nil
	}

	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return nil, fmt.Errorf("failed to load user info: %w", err)
	}

	var claims struct {
		Subject           string `json:"sub"`
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := userInfo.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse user info claims: %w", err)
	}

	session.UserID = claims.Subject
	session.Email = claims.Email
	session.Username = claims.PreferredUsername

	return session, nil
}

func (s *Service) startCallbackServer(
	redirectURL *url.URL,
	expectedState string,
	codeCh chan<- string,
	errCh chan<- error,
) (*http.Server, net.Listener, error) {
	hostPort := redirectURL.Host
	if !strings.Contains(hostPort, ":") {
		hostPort += ":80"
	}

	mux := http.NewServeMux()
	mux.HandleFunc(redirectURL.Path, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if authErr := query.Get("error"); authErr != "" {
			description := query.Get("error_description")
			if description == "" {
				description = authErr
			}
			select {
			case errCh <- errors.New(description):
			default:
			}
			http.Error(w, description, http.StatusBadRequest)
			return
		}

		if state := query.Get("state"); state != expectedState {
			select {
			case errCh <- errors.New("invalid auth state"):
			default:
			}
			http.Error(w, "invalid auth state", http.StatusBadRequest)
			return
		}

		code := query.Get("code")
		if code == "" {
			select {
			case errCh <- errors.New("authorization code is missing"):
			default:
			}
			http.Error(w, "authorization code is missing", http.StatusBadRequest)
			return
		}

		_, _ = w.Write([]byte("Authentication completed. You can return to k8s-manager."))

		select {
		case codeCh <- code:
		default:
		}
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", hostPort)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen on callback address %s: %w", hostPort, err)
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	return server, listener, nil
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}

	return cmd.Start()
}

func randomBase64URL(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func registrationURL(authURL string) (string, error) {
	parsed, err := url.Parse(authURL)
	if err != nil {
		return "", fmt.Errorf("failed to build registration url: %w", err)
	}

	parsed.Path = strings.Replace(parsed.Path, "/auth", "/registrations", 1)
	return parsed.String(), nil
}
