package auth

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultIssuerURL    = "http://localhost:8081/realms/k8s-manager"
	defaultClientID     = "k8s-manager-cli"
	defaultRedirectURL  = "http://127.0.0.1:18080/callback"
	defaultKeyringName  = "k8s-manager"
	defaultKeyringEntry = "keycloak-session-key"
)

type Config struct {
	IssuerURL    string
	ClientID     string
	RedirectURL  string
	KeyringName  string
	KeyringEntry string
	SessionPath  string
}

func LoadConfig() Config {
	cfg := Config{
		IssuerURL:    getEnv("KEYCLOAK_ISSUER_URL", defaultIssuerURL),
		ClientID:     getEnv("KEYCLOAK_CLIENT_ID", defaultClientID),
		RedirectURL:  getEnv("KEYCLOAK_REDIRECT_URL", defaultRedirectURL),
		KeyringName:  getEnv("KEYRING_SERVICE_NAME", defaultKeyringName),
		KeyringEntry: getEnv("KEYRING_SESSION_ENTRY", defaultKeyringEntry),
		SessionPath:  getEnv("KEYCLOAK_SESSION_PATH", defaultSessionPath()),
	}

	if !strings.HasPrefix(cfg.RedirectURL, "http://") && !strings.HasPrefix(cfg.RedirectURL, "https://") {
		cfg.RedirectURL = defaultRedirectURL
	}

	if _, err := url.Parse(cfg.RedirectURL); err != nil {
		cfg.RedirectURL = defaultRedirectURL
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func defaultSessionPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		return filepath.Join(os.TempDir(), "k8s-manager", "session.enc")
	}

	return filepath.Join(configDir, "k8s-manager", "session.enc")
}
