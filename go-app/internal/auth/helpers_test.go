package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testConfig is a complete configuration whose GitHub URLs point at githubURL
// (a fakeGitHub's server URL, or "" where GitHub must never be called).
func testConfig(githubURL string) Config {
	if githubURL == "" {
		githubURL = "http://github.invalid"
	}
	return Config{
		ClientID:      "id",
		ClientSecret:  "secret",
		SessionKey:    testKey,
		PublicBaseURL: "http://service.test",
		SiteBaseURL:   "http://site.test",
		AuthorizeURL:  githubURL + "/login/oauth/authorize",
		TokenURL:      githubURL + "/login/oauth/access_token",
		APIBaseURL:    githubURL + "/api",
	}
}

func newTestAuth(t *testing.T, githubURL string) *Auth {
	t.Helper()
	a := New(testConfig(githubURL))
	a.now = func() time.Time { return sessionNow }
	return a
}

func findCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func assertPrivateHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	h := rec.Header()
	if got := h.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want none", got)
	}
	if got := h.Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q", got)
	}
	csp := h.Get("Content-Security-Policy")
	for _, want := range []string{"frame-ancestors 'none'", "frame-src https://plausible.io", "form-action 'self' http://site.test"} {
		if !strings.Contains(csp, want) {
			t.Errorf("Content-Security-Policy %q lacks %q", csp, want)
		}
	}
}
