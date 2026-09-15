// Package auth lets only maintainers see protected pages: people with push
// access to GateRepo, established by a GitHub login (ADR-0022). It knows
// nothing about statistics.
package auth

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"arc42-status/internal/env"
)

// GateRepo is the repository whose push permission grants access.
const GateRepo = "arc42/status.arc42.org-site"

const (
	SessionCookie = "rollup_session"
	StateCookie   = "rollup_oauth"

	minKeyBytes = 32
)

// Config is everything the login needs. FromEnv fills it; tests build it
// directly and point the GitHub URLs at a fake.
type Config struct {
	ClientID      string
	ClientSecret  string
	SessionKey    []byte
	PublicBaseURL string // where this service is reachable, no trailing slash
	SiteBaseURL   string // where the Jekyll site is reachable, no trailing slash
	SecureCookies bool

	AuthorizeURL string
	TokenURL     string
	APIBaseURL   string

	keyErr error
}

// FromEnv reads the login configuration from the environment (ADR-0018).
// It never fails: whatever is missing or malformed is reported by Problem,
// so the rest of the service starts regardless.
func FromEnv(environment string) Config {
	cfg := Config{
		ClientID:      strings.TrimSpace(os.Getenv("GITHUB_OAUTH_CLIENT_ID")),
		ClientSecret:  strings.TrimSpace(os.Getenv("GITHUB_OAUTH_CLIENT_SECRET")),
		PublicBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/"),
		SiteBaseURL:   env.SiteBaseURL(environment),
		SecureCookies: environment == "PROD",
		AuthorizeURL:  "https://github.com/login/oauth/authorize",
		TokenURL:      "https://github.com/login/oauth/access_token",
		APIBaseURL:    "https://api.github.com",
	}
	if raw := strings.TrimSpace(os.Getenv("SESSION_KEY")); raw != "" {
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			cfg.keyErr = fmt.Errorf("SESSION_KEY is not valid base64: %w", err)
		}
		cfg.SessionKey = key
	}
	return cfg
}

// Problem says why the login cannot be used, or returns nil when it can.
func (c Config) Problem() error {
	var missing []string
	if c.ClientID == "" {
		missing = append(missing, "GITHUB_OAUTH_CLIENT_ID")
	}
	if c.ClientSecret == "" {
		missing = append(missing, "GITHUB_OAUTH_CLIENT_SECRET")
	}
	if len(c.SessionKey) == 0 && c.keyErr == nil {
		missing = append(missing, "SESSION_KEY")
	}
	if c.PublicBaseURL == "" {
		missing = append(missing, "PUBLIC_BASE_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("login not configured, missing %s", strings.Join(missing, ", "))
	}
	if c.keyErr != nil {
		return c.keyErr
	}
	if len(c.SessionKey) < minKeyBytes {
		return fmt.Errorf("SESSION_KEY must decode to at least %d bytes, has %d", minKeyBytes, len(c.SessionKey))
	}
	return nil
}
