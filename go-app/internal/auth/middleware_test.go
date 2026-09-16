package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func protected(a *Auth) http.Handler {
	return a.RequirePush(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello "+LoginFrom(r.Context()))
	}))
}

func TestRequirePushRedirectsWithoutSession(t *testing.T) {
	a := newTestAuth(t, "")
	rec := httptest.NewRecorder()

	protected(a).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/rollup", nil))

	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/auth/login" {
		t.Errorf("got %d to %q, want 302 to /auth/login", rec.Code, rec.Header().Get("Location"))
	}
	assertPrivateHeaders(t, rec)
}

func TestRequirePushRedirectsOnInvalidSession(t *testing.T) {
	a := newTestAuth(t, "")
	req := httptest.NewRequest(http.MethodGet, "/rollup", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookie, Value: "garbage"})
	rec := httptest.NewRecorder()

	protected(a).ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("got %d, want 302", rec.Code)
	}
}

func TestRequirePushPassesLogin(t *testing.T) {
	a := newTestAuth(t, "")
	req := httptest.NewRequest(http.MethodGet, "/rollup", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookie, Value: encodeSession(session{Login: "octocat", Expires: sessionNow.Add(time.Hour)}, testKey)})
	rec := httptest.NewRecorder()

	protected(a).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "hello octocat" {
		t.Errorf("got %d %q, want 200 \"hello octocat\"", rec.Code, rec.Body.String())
	}
	assertPrivateHeaders(t, rec)
}

func TestRequirePushUnconfigured(t *testing.T) {
	a := New(Config{SiteBaseURL: "http://site.test"})
	rec := httptest.NewRecorder()

	protected(a).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/rollup", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("got %d, want 503", rec.Code)
	}
}
