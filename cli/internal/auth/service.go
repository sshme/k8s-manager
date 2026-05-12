package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// Service - сервис для аутентификации через OIDC.
type Service struct {
	cfg   Config
	store *Store
}

// NewService собирает Service по конфигу и инициализирует Store
// с теми же данными из этого конфига.
func NewService(cfg Config) *Service {
	return &Service{
		cfg:   cfg,
		store: NewStore(cfg.KeyringName, cfg.KeyringEntry, cfg.SessionPath),
	}
}

// LoadSession читает сохранённую сессию. Если Session.NeedsRefresh, то
// рефрещ происходит автоматически. Если рефреш не удался, то локальная
// сессия удаляется, поскольку она протухшая.
func (s *Service) LoadSession(ctx context.Context) (*Session, error) {
	session, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	if !session.NeedsRefresh() {
		s.enrichSessionRoles(session)
		return session, nil
	}

	refreshed, err := s.refreshSession(ctx, session)
	if err != nil {
		_ = s.store.Delete()
		return nil, err
	}

	return refreshed, nil
}

// Authenticate запускает обычный логин-флоу. Keycloak покажет форму входа.
func (s *Service) Authenticate(ctx context.Context) (*Session, error) {
	return s.authorize(ctx, false)
}

// Register запускает тот же Authenticate флоу, но Keycloak сразу откроет форму
// регистрации вместо логина.
func (s *Service) Register(ctx context.Context) (*Session, error) {
	return s.authorize(ctx, true)
}

// authorize - OAuth2 Authorization Code Flow + PKCE.
//
// Сначала генерирует state и PKCE verifier/challenge, затем
// поднимает локальный HTTP-сервер на RedirectURL чтобы поймать code,
// открывает браузер с auth URL, пользователь логинится в Keycloak,
// Keycloak редиректит браузер на callback с code,
// code + verifier меняются на токены через token endpoint,
// достаются claims пользователя (id_token или userinfo)
// и всё сохранятся в Store.
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

	// Сервер живёт ровно на время флоу, после выхода из функции освобождаем порт
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

	// Либо юзер закрыл TUI (ctx), либо callback-сервер вернул ошибку,
	// либо нам прилетел code из редиректа
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

// AttachToken возвращает актуальный access-токен
func (s *Service) AttachToken(ctx context.Context) (string, error) {
	session, err := s.LoadSession(ctx)
	if err != nil {
		return "", err
	}

	return session.AccessToken, nil
}

// RevocationFailedError означает что локально logout прошёл успешно, но
// сервер про это не узнал, то есть refresh-токен формально ещё валиден
type RevocationFailedError struct {
	Err error
}

func (e *RevocationFailedError) Error() string {
	return fmt.Sprintf("server-side revocation failed: %v", e.Err)
}

func (e *RevocationFailedError) Unwrap() error { return e.Err }

// Logout завершает сессию и локально (удаляет keyring-запись и зашифрованный
// файл сессии), и на OIDC-сервере.
func (s *Service) Logout(ctx context.Context) error {
	session, err := s.store.Load()
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil
		}
		return fmt.Errorf("load session for logout: %w", err)
	}

	revokeErr := s.revokeRefreshToken(ctx, session.RefreshToken)

	if err := s.store.Delete(); err != nil {
		return fmt.Errorf("delete local session: %w", err)
	}

	if revokeErr != nil {
		return &RevocationFailedError{Err: revokeErr}
	}
	return nil
}

// revokeRefreshToken отправляет refresh-токен на revocation OIDC-эндпоинт.
// Адрес эндпоинта берётся из discovery Keycloak.
func (s *Service) revokeRefreshToken(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return errors.New("no refresh token to revoke")
	}

	provider, _, err := s.providerConfig(ctx)
	if err != nil {
		return err
	}

	var meta struct {
		RevocationEndpoint string `json:"revocation_endpoint"`
	}
	if err := provider.Claims(&meta); err != nil {
		return fmt.Errorf("read provider metadata: %w", err)
	}
	if meta.RevocationEndpoint == "" {
		return errors.New("provider does not advertise revocation_endpoint")
	}

	form := url.Values{}
	form.Set("client_id", s.cfg.ClientID)
	form.Set("token", refreshToken)
	form.Set("token_type_hint", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		meta.RevocationEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build revocation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("call revocation endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("revocation endpoint returned %s: %s",
		resp.Status, strings.TrimSpace(string(body)))
}

// providerConfig делает OIDC discovery и собирает oauth2.Config с нужными scope
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

// refreshSession принудительно обменивает refresh_token на новую пару токенов
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

		// Во избежание непредвиденного поведения, принудительно
		// делаем токен как будто истекшим, потому что в golang.org/x/oauth2
		// использует собственное значение delta для проверки свежести токена
		Expiry: time.Now().Add(-1 * time.Second),
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
	if len(nextSession.Roles) == 0 {
		nextSession.Roles = session.Roles
	}
	s.enrichSessionRoles(nextSession)

	if err := s.store.Save(nextSession); err != nil {
		return nil, fmt.Errorf("failed to persist refreshed session: %w", err)
	}

	return nextSession, nil
}

// sessionFromToken достаёт claims пользователя (sub, email, имя) из
// id_token. Если id_token отсутствует или claims в нём пустые, то
// производится попытка достать необходимое из userinfo endpoint.
// Заодно verifier проверяет подпись id_token
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
			RealmAccess       struct {
				Roles []string `json:"roles"`
			} `json:"realm_access"`
			ResourceAccess map[string]struct {
				Roles []string `json:"roles"`
			} `json:"resource_access"`
		}
		if err := idToken.Claims(&claims); err != nil {
			return nil, fmt.Errorf("failed to parse id token claims: %w", err)
		}

		session.UserID = claims.Subject
		session.Email = claims.Email
		session.Username = claims.PreferredUsername
		session.Roles = mergeRoles(session.Roles, claims.RealmAccess.Roles)
		for _, access := range claims.ResourceAccess {
			session.Roles = mergeRoles(session.Roles, access.Roles)
		}
	}

	if session.Username != "" || session.Email != "" || session.UserID != "" {
		s.enrichSessionRoles(session)
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
	s.enrichSessionRoles(session)

	return session, nil
}

func (s *Service) enrichSessionRoles(session *Session) {
	if session == nil {
		return
	}
	session.Roles = mergeRoles(session.Roles, rolesFromJWT(session.IDToken))
	session.Roles = mergeRoles(session.Roles, rolesFromJWT(session.AccessToken))
}

func rolesFromJWT(rawToken string) []string {
	parts := strings.Split(rawToken, ".")
	if len(parts) < 2 {
		return nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}

	var claims struct {
		RealmAccess struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
		ResourceAccess map[string]struct {
			Roles []string `json:"roles"`
		} `json:"resource_access"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}

	roles := mergeRoles(nil, claims.RealmAccess.Roles)
	for _, access := range claims.ResourceAccess {
		roles = mergeRoles(roles, access.Roles)
	}
	return roles
}

func mergeRoles(current, next []string) []string {
	for _, role := range next {
		role = strings.TrimSpace(role)
		if role == "" || containsString(current, role) {
			continue
		}
		current = append(current, role)
	}
	return current
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// startCallbackServer поднимает локальный HTTP-сервер на RedirectURL
// и ждёт ровно один редирект от Keycloak. Полученный code и любые
// ошибки направляет в каналы (неблокирующе). State-токен сверяется
// здесь же, чтобы отсеять CSRF и просто чужие редиректы, прилетевшие
// на порт
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

// openBrowser открывает url в дефолтном браузере ОС
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

// randomBase64URL - size случайных байт в base64url
func randomBase64URL(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// pkceChallenge = base64url(sha256(verifier))
func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// registrationURL переписывает /auth на /registrations в authURL
func registrationURL(authURL string) (string, error) {
	parsed, err := url.Parse(authURL)
	if err != nil {
		return "", fmt.Errorf("failed to build registration url: %w", err)
	}

	parsed.Path = strings.Replace(parsed.Path, "/auth", "/registrations", 1)
	return parsed.String(), nil
}
