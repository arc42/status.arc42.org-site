package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

func setCompleteEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GITHUB_OAUTH_CLIENT_ID", "id")
	t.Setenv("GITHUB_OAUTH_CLIENT_SECRET", "secret")
	t.Setenv("SESSION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:8043/")
}

func TestFromEnvComplete(t *testing.T) {
	setCompleteEnv(t)

	c := FromEnv("DEV")
	if err := c.Problem(); err != nil {
		t.Fatalf("complete configuration reported a problem: %v", err)
	}
	if c.PublicBaseURL != "http://localhost:8043" {
		t.Errorf("PublicBaseURL = %q, want the trailing slash trimmed", c.PublicBaseURL)
	}
	if c.SiteBaseURL != "http://localhost:4270" {
		t.Errorf("SiteBaseURL = %q", c.SiteBaseURL)
	}
	if c.SecureCookies {
		t.Error("DEV must not mark cookies Secure: localhost runs on plain http")
	}
	if c.AuthorizeURL != "https://github.com/login/oauth/authorize" ||
		c.TokenURL != "https://github.com/login/oauth/access_token" ||
		c.APIBaseURL != "https://api.github.com" {
		t.Errorf("GitHub URLs not defaulted: %q %q %q", c.AuthorizeURL, c.TokenURL, c.APIBaseURL)
	}
	if !FromEnv("PROD").SecureCookies {
		t.Error("PROD must mark cookies Secure")
	}
}

func TestProblemNamesEachMissingVariable(t *testing.T) {
	for _, name := range []string{"GITHUB_OAUTH_CLIENT_ID", "GITHUB_OAUTH_CLIENT_SECRET", "SESSION_KEY", "PUBLIC_BASE_URL"} {
		t.Run(name, func(t *testing.T) {
			setCompleteEnv(t)
			t.Setenv(name, "")

			err := FromEnv("DEV").Problem()
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("Problem() = %v, want an error naming %s", err, name)
			}
		})
	}
}

func TestProblemRejectsShortOrMalformedKey(t *testing.T) {
	setCompleteEnv(t)

	t.Setenv("SESSION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 31)))
	if FromEnv("DEV").Problem() == nil {
		t.Error("a 31-byte key was accepted")
	}

	t.Setenv("SESSION_KEY", "not base64 !!")
	if FromEnv("DEV").Problem() == nil {
		t.Error("a key that is not base64 was accepted")
	}
}
