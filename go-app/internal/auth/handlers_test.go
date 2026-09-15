package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// startLogin runs /auth/login and returns the state GitHub would echo back and
// the cookie the browser would carry to the callback.
func startLogin(t *testing.T, a *Auth) (string, *http.Cookie) {
	t.Helper()
	rec := httptest.NewRecorder()
	a.Login(rec, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	u, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("login redirect: %v", err)
	}
	return u.Query().Get("state"), findCookie(rec, StateCookie)
}

func callback(a *Auth, query string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/auth/callback?"+query, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	a.Callback(rec, req)
	return rec
}

func TestLoginRedirectsToGitHubWithStateAndPKCE(t *testing.T) {
	f := newFakeGitHub(t)
	a := newTestAuth(t, f.server.URL)
	rec := httptest.NewRecorder()

	a.Login(rec, httptest.NewRequest(http.MethodGet, "/auth/login", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("got %d, want 302", rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, f.server.URL+"/login/oauth/authorize?") {
		t.Fatalf("redirect to %q, want GitHub's authorize URL", location)
	}
	u, _ := url.Parse(location)
	q := u.Query()
	if q.Get("state") == "" {
		t.Error("no state")
	}
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		t.Errorf("no S256 PKCE challenge: %q %q", q.Get("code_challenge"), q.Get("code_challenge_method"))
	}
	if q.Get("redirect_uri") != "http://service.test/auth/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Has("scope") {
		t.Errorf("login must request no scope, got %q", q.Get("scope"))
	}
	c := findCookie(rec, StateCookie)
	if c == nil || !c.HttpOnly || c.Path != "/auth" || c.MaxAge != 600 || c.SameSite != http.SameSiteLaxMode {
		t.Errorf("state cookie = %+v", c)
	}
	assertPrivateHeaders(t, rec)
}

func TestCallbackWithPushSetsSessionAndRedirects(t *testing.T) {
	f := newFakeGitHub(t)
	a := newTestAuth(t, f.server.URL)
	state, stateCookie := startLogin(t, a)

	rec := callback(a, "code=abc&state="+state+"&next=https://evil.example/", stateCookie)

	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/rollup" {
		t.Fatalf("got %d to %q, want 302 to /rollup", rec.Code, rec.Header().Get("Location"))
	}
	c := findCookie(rec, SessionCookie)
	if c == nil || !c.HttpOnly || c.Path != "/" || c.MaxAge != 28800 {
		t.Fatalf("session cookie = %+v", c)
	}
	s, ok := decodeSession(c.Value, testKey, sessionNow)
	if !ok || s.Login != "octocat" {
		t.Errorf("session = %+v ok=%t, want octocat", s, ok)
	}
	verifier, authorization := f.recorded()
	if verifier == "" {
		t.Error("the code exchange sent no PKCE verifier")
	}
	if authorization != "Bearer gho_test" {
		t.Errorf("API called with Authorization %q", authorization)
	}
	if strings.Contains(rec.Body.String(), "gho_test") {
		t.Error("the token leaked into the response")
	}
	assertPrivateHeaders(t, rec)
}

func TestCallbackWithoutPushIsForbidden(t *testing.T) {
	f := newFakeGitHub(t)
	f.set(func(g *fakeGitHub) { g.push = false })
	a := newTestAuth(t, f.server.URL)
	state, stateCookie := startLogin(t, a)

	rec := callback(a, "code=abc&state="+state, stateCookie)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
	if findCookie(rec, SessionCookie) != nil {
		t.Error("a session cookie was set without push access")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "octocat") || !strings.Contains(body, GateRepo) {
		t.Errorf("403 page does not name the login and the repository: %s", body)
	}
}

func TestCallbackRejectsBadState(t *testing.T) {
	f := newFakeGitHub(t)
	a := newTestAuth(t, f.server.URL)
	state, stateCookie := startLogin(t, a)

	cases := map[string]func() *httptest.ResponseRecorder{
		"wrong state":     func() *httptest.ResponseRecorder { return callback(a, "code=abc&state=other", stateCookie) },
		"no state cookie": func() *httptest.ResponseRecorder { return callback(a, "code=abc&state="+state, nil) },
		"no state":        func() *httptest.ResponseRecorder { return callback(a, "code=abc", stateCookie) },
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			if rec := run(); rec.Code != http.StatusBadRequest {
				t.Errorf("got %d, want 400", rec.Code)
			}
		})
	}
}

func TestCallbackRejectsExpiredState(t *testing.T) {
	f := newFakeGitHub(t)
	a := newTestAuth(t, f.server.URL)
	state, stateCookie := startLogin(t, a)
	a.now = func() time.Time { return sessionNow.Add(11 * time.Minute) }

	if rec := callback(a, "code=abc&state="+state, stateCookie); rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestCallbackCancelled(t *testing.T) {
	f := newFakeGitHub(t)
	a := newTestAuth(t, f.server.URL)
	state, stateCookie := startLogin(t, a)

	rec := callback(a, "error=access_denied&state="+state, stateCookie)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Login cancelled") {
		t.Errorf("got %d %q, want 200 with \"Login cancelled\"", rec.Code, rec.Body.String())
	}
}

func TestCallbackGitHubFailures(t *testing.T) {
	cases := map[string]func(*fakeGitHub, *Auth){
		"token endpoint 500": func(f *fakeGitHub, _ *Auth) {
			f.set(func(g *fakeGitHub) { g.tokenStatus = http.StatusInternalServerError })
		},
		"API 500": func(f *fakeGitHub, _ *Auth) {
			f.set(func(g *fakeGitHub) { g.apiStatus = http.StatusInternalServerError })
		},
		"API slower than the timeout": func(f *fakeGitHub, a *Auth) {
			f.set(func(g *fakeGitHub) { g.apiDelay = 200 * time.Millisecond })
			a.client.Timeout = 50 * time.Millisecond
		},
	}
	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFakeGitHub(t)
			a := newTestAuth(t, f.server.URL)
			state, stateCookie := startLogin(t, a)
			breakIt(f, a)

			rec := callback(a, "code=abc&state="+state, stateCookie)

			if rec.Code != http.StatusBadGateway {
				t.Errorf("got %d, want 502", rec.Code)
			}
			if findCookie(rec, SessionCookie) != nil {
				t.Error("a session cookie was set although GitHub failed")
			}
		})
	}
}

func TestHandlersUnconfigured(t *testing.T) {
	a := New(Config{SiteBaseURL: "http://site.test"})

	rec := httptest.NewRecorder()
	a.Login(rec, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("login: got %d, want 503", rec.Code)
	}

	rec = httptest.NewRecorder()
	a.Callback(rec, httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state=x", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("callback: got %d, want 503", rec.Code)
	}
}

func TestLogout(t *testing.T) {
	a := newTestAuth(t, "")

	rec := httptest.NewRecorder()
	a.Logout(rec, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "http://site.test/" {
		t.Errorf("got %d to %q, want 302 to http://site.test/", rec.Code, rec.Header().Get("Location"))
	}
	if c := findCookie(rec, SessionCookie); c == nil || c.MaxAge >= 0 {
		t.Errorf("session cookie not cleared: %+v", c)
	}
	assertPrivateHeaders(t, rec)

	rec = httptest.NewRecorder()
	a.Logout(rec, httptest.NewRequest(http.MethodGet, "/auth/logout", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET logout: got %d, want 405", rec.Code)
	}
}

func TestLoginAndCallbackRejectPost(t *testing.T) {
	a := newTestAuth(t, "")

	rec := httptest.NewRecorder()
	a.Login(rec, httptest.NewRequest(http.MethodPost, "/auth/login", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST login: got %d, want 405", rec.Code)
	}

	rec = httptest.NewRecorder()
	a.Callback(rec, httptest.NewRequest(http.MethodPost, "/auth/callback", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST callback: got %d, want 405", rec.Code)
	}
}
