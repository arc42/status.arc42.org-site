package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// fakeGitHub answers the three GitHub endpoints the login uses. Its fields
// are read by the server's goroutines, so change them through set and read
// what it recorded through recorded.
type fakeGitHub struct {
	server *httptest.Server

	mu          sync.Mutex
	login       string
	push        bool
	tokenStatus int // non-zero: the token endpoint answers with this status
	apiStatus   int // non-zero: both API endpoints answer with this status
	apiDelay    time.Duration
	gotVerifier string
	gotAuth     string
}

func (f *fakeGitHub) set(change func(*fakeGitHub)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	change(f)
}

func (f *fakeGitHub) recorded() (verifier, authorization string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.gotVerifier, f.gotAuth
}

func newFakeGitHub(t *testing.T) *fakeGitHub {
	t.Helper()
	f := &fakeGitHub{login: "octocat", push: true}

	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		status := f.tokenStatus
		f.gotVerifier = r.Form.Get("code_verifier")
		f.mu.Unlock()
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"gho_test","token_type":"bearer"}`)
	})

	api := func(body func(*fakeGitHub) string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			f.mu.Lock()
			status, delay, payload := f.apiStatus, f.apiDelay, body(f)
			f.gotAuth = r.Header.Get("Authorization")
			f.mu.Unlock()
			time.Sleep(delay)
			if status != 0 {
				w.WriteHeader(status)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, payload)
		}
	}
	mux.HandleFunc("/api/user", api(func(f *fakeGitHub) string {
		return fmt.Sprintf(`{"login":%q}`, f.login)
	}))
	mux.HandleFunc("/api/repos/"+GateRepo, api(func(f *fakeGitHub) string {
		return fmt.Sprintf(`{"permissions":{"admin":false,"push":%t,"pull":true}}`, f.push)
	}))

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}
