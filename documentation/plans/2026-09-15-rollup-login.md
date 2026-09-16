# Maintainers-only Rollup Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve the rollup page only to GitHub users with push access to `arc42/status.arc42.org-site`, and remove every public trace of the rollup figures.

**Architecture:** The Go service renders `/rollup` as a complete HTML page behind a new `internal/auth` package: GitHub OAuth App login (PKCE, `state`), a push check against the GitHub REST API with the user's token, and an HMAC-signed session cookie. The public Jekyll page, the table's "Unique across sites" footer row and the public `/rollup` fragment are deleted.

**Tech Stack:** Go 1.21 (stdlib `net/http`, `html/template`, `crypto/hmac`, `testing`, `httptest`), `golang.org/x/oauth2` v0.14.0 (already a dependency), zerolog, Jekyll site on GitHub Pages, fly.io.

**Spec:** `documentation/specs/2026-09-15-rollup-login-design.md`

## Global Constraints

- Go version in `go.mod` is 1.21: no `http.ServeMux` method patterns (`"GET /x"`), check `r.Method` by hand.
- No new module dependencies. `golang.org/x/oauth2` is already required.
- Tests use the standard `testing` package only, and never reach the network: GitHub is an `httptest.Server`.
- Gate repository: `arc42/status.arc42.org-site`, constant `auth.GateRepo`.
- Cookie names: `rollup_session` (Path `/`, 8 h) and `rollup_oauth` (Path `/auth`, 10 min). Both HttpOnly, SameSite=Lax, Secure only when the environment is `PROD`.
- `SESSION_KEY` is base64 (standard encoding) and must decode to at least 32 bytes.
- Environment variables: `GITHUB_OAUTH_CLIENT_ID`, `GITHUB_OAUTH_CLIENT_SECRET`, `SESSION_KEY`, `PUBLIC_BASE_URL`, `PLAUSIBLE_ROLLUP_SHARE_URL`.
- `env.SiteBaseURL`: `PROD` → `https://status.arc42.org`, everything else → `http://localhost:4270`.
- Never log or store the OAuth code, the access token, or cookie values. Log the GitHub login and the outcome.
- `/rollup` and `/auth/*` send no `Access-Control-Allow-*` headers.
- Logging uses `github.com/rs/zerolog/log`, like the rest of the service.
- Commit messages: short lowercase summary line (house style, see `git log`), ending with
  `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`.
- Go commands (`go test`, `go run`, `source ./set-api-keys.sh`) run from `go-app/`. `make`, `git` and `grep` commands run from the repository root, and every path in them is relative to the root.

## File Structure

| File | Responsibility |
|---|---|
| `go-app/internal/env/environment.go` (modify) | `SiteBaseURL(environment)` |
| `go-app/internal/env/environment_test.go` (create) | its test |
| `go-app/internal/auth/config.go` (create) | `Config`, `FromEnv`, `Problem`, constants |
| `go-app/internal/auth/session.go` (create) | signing, verifying, session encode/decode |
| `go-app/internal/auth/github.go` (create) | `checkPush` against the GitHub REST API |
| `go-app/internal/auth/auth.go` (create) | `Auth`, `New`, lifetimes, embedded template |
| `go-app/internal/auth/pages.go` (create) | private headers, message pages, `render`, `unusable` |
| `go-app/internal/auth/middleware.go` (create) | `RequirePush`, `LoginFrom` |
| `go-app/internal/auth/handlers.go` (create) | `Login`, `Callback`, `Logout` |
| `go-app/internal/auth/templates/authMessage.gohtml` (create) | page for cancelled / expired / forbidden / unavailable |
| `go-app/internal/auth/*_test.go` (create) | tests, fake GitHub, helpers |
| `go-app/internal/api/rollupPage.gohtml` (create) | the maintainers-only page |
| `go-app/internal/api/rollup.gohtml` (delete) | the old public fragment |
| `go-app/internal/api/apiGateway.go` (modify) | `rollupPageHandler`, routes |
| `go-app/internal/types/rollup.go` (modify) | `RollupPageData` fields, comments |
| `go-app/cmd/rendercheck/main.go` (modify) | render the page, check its links are absolute |
| `go-app/internal/api/arc42statistics.gohtml` (modify) | drop the footer row |
| `docs/assets/css/arc42-status-style.css` (modify) | drop `.stats-rollup`, add page classes |
| `docs/_pages/rollup.md` (delete) | the old public page |
| `docs/_pages/home.md` (modify) | skeleton comment |
| `go-app/set-api-keys.sh.template`, `Makefile`, `README.md` (modify) | local setup |
| `documentation/adrs/0022-show-the-rollup-only-to-maintainers.md` (create), ADR-0021 (modify) | decision record |
| `go-app/main.go` (modify) | version 1.5.0 |

---

### Task 0: Commit pending work and branch

The working tree holds the uncommitted ADR-0021 rollup work, trainings back in the table, the spec and this plan. They go to `main` first, so the login work starts from a clean branch.

**Files:** none changed; git only.

- [ ] **Step 1: Review what is pending**

Run: `git status --short`
Expected: modified and untracked files including `documentation/adrs/0021-…`, `documentation/specs/…`, `documentation/plans/…`, `go-app/internal/api/rollup.gohtml`, `go-app/internal/types/rollup.go`.

- [ ] **Step 2: Stage everything and make sure no secret is included**

```bash
git add -A
git diff --cached --name-only | grep -E 'set-api-keys\.sh$' && echo "STOP: secrets staged" || echo "no secrets staged"
```
Expected: `no secrets staged`. If `STOP` appears, run `git restore --staged go-app/set-api-keys.sh` and ask the maintainer.

- [ ] **Step 3: Run the tests on what is about to be committed**

Run: `make test`
Expected: all packages `ok` or `[no test files]`.

- [ ] **Step 4: Commit on main (ask the maintainer to confirm first)**

```bash
git commit -m "rollup beside the summed totals (ADR-0021), trainings back in the table, rollup login spec and plan

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

- [ ] **Step 5: Create the feature branch**

Run: `git switch -c rollup-login`
Expected: `Switched to a new branch 'rollup-login'`

---

### Task 1: `env.SiteBaseURL`

**Files:**
- Modify: `go-app/internal/env/environment.go` (append)
- Test: `go-app/internal/env/environment_test.go` (create)

**Interfaces:**
- Produces: `func SiteBaseURL(environment string) string` in package `env`.

- [ ] **Step 1: Write the failing test**

`go-app/internal/env/environment_test.go`:
```go
package env

import "testing"

func TestSiteBaseURL(t *testing.T) {
	cases := map[string]string{
		"PROD": "https://status.arc42.org",
		"DEV":  "http://localhost:4270",
		"TEST": "http://localhost:4270",
	}
	for environment, want := range cases {
		if got := SiteBaseURL(environment); got != want {
			t.Errorf("SiteBaseURL(%q) = %q, want %q", environment, got, want)
		}
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/env/ -run TestSiteBaseURL -v`
Expected: FAIL, `undefined: SiteBaseURL`

- [ ] **Step 3: Implement**

Append to `go-app/internal/env/environment.go`:
```go
// SiteBaseURL is where the Jekyll site of the given environment is served.
// Pages the service renders itself (the rollup page, ADR-0022) are served from
// the service's own host, so their stylesheet and their links into the site
// are absolute against this.
func SiteBaseURL(environment string) string {
	if environment == "PROD" {
		return "https://status.arc42.org"
	}
	return "http://localhost:4270"
}
```

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./internal/env/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go-app/internal/env/environment.go go-app/internal/env/environment_test.go
git commit -m "env: site base url per environment

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 2: `auth.Config`

**Files:**
- Create: `go-app/internal/auth/config.go`
- Test: `go-app/internal/auth/config_test.go`

**Interfaces:**
- Consumes: `env.SiteBaseURL(environment string) string` (Task 1).
- Produces:
  - constants `GateRepo = "arc42/status.arc42.org-site"`, `SessionCookie = "rollup_session"`, `StateCookie = "rollup_oauth"`
  - `type Config struct { ClientID, ClientSecret string; SessionKey []byte; PublicBaseURL, SiteBaseURL string; SecureCookies bool; AuthorizeURL, TokenURL, APIBaseURL string; keyErr error }`
  - `func FromEnv(environment string) Config`
  - `func (c Config) Problem() error`

- [ ] **Step 1: Write the failing tests**

`go-app/internal/auth/config_test.go`:
```go
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
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/auth/ -v`
Expected: FAIL, `undefined: FromEnv`

- [ ] **Step 3: Implement**

`go-app/internal/auth/config.go`:
```go
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
```

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./internal/auth/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go-app/internal/auth/config.go go-app/internal/auth/config_test.go
git commit -m "auth: login configuration from the environment

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 3: Signed cookies and the session

**Files:**
- Create: `go-app/internal/auth/session.go`
- Test: `go-app/internal/auth/session_test.go`

**Interfaces:**
- Produces:
  - `func sign(value string, key []byte) string` — `base64url(value) + "." + base64url(HMAC-SHA256)`, unpadded
  - `func verify(cookieValue string, key []byte) (string, bool)`
  - `type session struct { Login string; Expires time.Time }`
  - `func encodeSession(s session, key []byte) string`
  - `func decodeSession(cookieValue string, key []byte, now time.Time) (session, bool)`
  - test fixtures (in `session_test.go`, used by later tasks): `var testKey []byte` (32 × `'k'`), `var sessionNow time.Time` (2026-09-15 12:00 UTC)

- [ ] **Step 1: Write the failing tests**

`go-app/internal/auth/session_test.go`:
```go
package auth

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	testKey    = bytes.Repeat([]byte("k"), 32)
	sessionNow = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
)

func TestSessionRoundTrip(t *testing.T) {
	cookie := encodeSession(session{Login: "octocat", Expires: sessionNow.Add(8 * time.Hour)}, testKey)

	s, ok := decodeSession(cookie, testKey, sessionNow)
	if !ok {
		t.Fatal("a freshly signed session was rejected")
	}
	if s.Login != "octocat" {
		t.Errorf("Login = %q, want octocat", s.Login)
	}
	if !s.Expires.Equal(sessionNow.Add(8 * time.Hour)) {
		t.Errorf("Expires = %v", s.Expires)
	}
}

func TestSessionRejects(t *testing.T) {
	inAnHour := strconv.FormatInt(sessionNow.Add(time.Hour).Unix(), 10)
	valid := encodeSession(session{Login: "octocat", Expires: sessionNow.Add(time.Hour)}, testKey)
	encValue, encMAC, _ := strings.Cut(valid, ".")

	// forged keeps the valid signature but swaps the signed value
	forged := func(value string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(value)) + "." + encMAC
	}

	cases := map[string]struct {
		cookie string
		key    []byte
		now    time.Time
	}{
		"login altered":  {forged("mallory|" + inAnHour), testKey, sessionNow},
		"expiry altered": {forged("octocat|" + strconv.FormatInt(sessionNow.Add(1000*time.Hour).Unix(), 10)), testKey, sessionNow},
		"expired":        {valid, testKey, sessionNow.Add(2 * time.Hour)},
		"other key":      {valid, bytes.Repeat([]byte("x"), 32), sessionNow},
		"no dot":         {encValue + encMAC, testKey, sessionNow},
		"bad base64":     {"!!!." + encMAC, testKey, sessionNow},
		"empty":          {"", testKey, sessionNow},
		"no login":       {sign("|"+inAnHour, testKey), testKey, sessionNow},
		"no expiry":      {sign("octocat", testKey), testKey, sessionNow},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, ok := decodeSession(c.cookie, c.key, c.now); ok {
				t.Errorf("cookie %q was accepted", c.cookie)
			}
		})
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/auth/ -run TestSession -v`
Expected: FAIL, `undefined: encodeSession`

- [ ] **Step 3: Implement**

`go-app/internal/auth/session.go`:
```go
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

// sign returns value and its HMAC-SHA256, each base64url-encoded, joined by a
// dot. The value is readable by anyone holding the cookie; it holds nothing
// secret, only nothing forgeable.
func sign(value string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(value)) + "." + enc.EncodeToString(mac.Sum(nil))
}

// verify returns the signed value when the signature matches.
func verify(cookieValue string, key []byte) (string, bool) {
	encValue, encMAC, found := strings.Cut(cookieValue, ".")
	if !found {
		return "", false
	}
	enc := base64.RawURLEncoding
	value, err := enc.DecodeString(encValue)
	if err != nil {
		return "", false
	}
	got, err := enc.DecodeString(encMAC)
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(value)
	if !hmac.Equal(got, mac.Sum(nil)) {
		return "", false
	}
	return string(value), true
}

// session is what rollup_session carries: who logged in, and until when.
// GitHub logins contain only letters, digits and hyphens, so "|" cannot
// occur inside one.
type session struct {
	Login   string
	Expires time.Time
}

func encodeSession(s session, key []byte) string {
	return sign(s.Login+"|"+strconv.FormatInt(s.Expires.Unix(), 10), key)
}

func decodeSession(cookieValue string, key []byte, now time.Time) (session, bool) {
	value, ok := verify(cookieValue, key)
	if !ok {
		return session{}, false
	}
	login, exp, found := strings.Cut(value, "|")
	if !found || login == "" {
		return session{}, false
	}
	unix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil {
		return session{}, false
	}
	expires := time.Unix(unix, 0)
	if !now.Before(expires) {
		return session{}, false
	}
	return session{Login: login, Expires: expires}, true
}
```

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./internal/auth/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go-app/internal/auth/session.go go-app/internal/auth/session_test.go
git commit -m "auth: signed session cookie

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 4: Push check against GitHub

**Files:**
- Create: `go-app/internal/auth/github.go`
- Create: `go-app/internal/auth/fake_github_test.go`
- Test: `go-app/internal/auth/github_test.go`

**Interfaces:**
- Consumes: `GateRepo` (Task 2).
- Produces:
  - `func checkPush(ctx context.Context, client *http.Client, apiBaseURL, token string) (login string, push bool, err error)`
  - test helper `func newFakeGitHub(t *testing.T) *fakeGitHub` serving `/login/oauth/authorize` (unused), `/login/oauth/access_token`, `/api/user`, `/api/repos/arc42/status.arc42.org-site`; `(*fakeGitHub).set(func(*fakeGitHub))`; `(*fakeGitHub).recorded() (verifier, authorization string)`; fields `login string`, `push bool`, `tokenStatus int`, `apiStatus int`, `apiDelay time.Duration`; `server *httptest.Server`. The API base URL for tests is `f.server.URL + "/api"`.

- [ ] **Step 1: Write the fake GitHub**

`go-app/internal/auth/fake_github_test.go`:
```go
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
```

- [ ] **Step 2: Write the failing tests**

`go-app/internal/auth/github_test.go`:
```go
package auth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestCheckPushGranted(t *testing.T) {
	f := newFakeGitHub(t)

	login, push, err := checkPush(context.Background(), f.server.Client(), f.server.URL+"/api", "gho_test")
	if err != nil {
		t.Fatalf("checkPush: %v", err)
	}
	if login != "octocat" || !push {
		t.Errorf("got login %q push %t, want octocat true", login, push)
	}
	if _, authorization := f.recorded(); authorization != "Bearer gho_test" {
		t.Errorf("Authorization header = %q", authorization)
	}
}

func TestCheckPushWithoutPermission(t *testing.T) {
	f := newFakeGitHub(t)
	f.set(func(g *fakeGitHub) { g.push = false })

	login, push, err := checkPush(context.Background(), f.server.Client(), f.server.URL+"/api", "gho_test")
	if err != nil {
		t.Fatalf("checkPush: %v", err)
	}
	if login != "octocat" || push {
		t.Errorf("got login %q push %t, want octocat false", login, push)
	}
}

func TestCheckPushAPIError(t *testing.T) {
	f := newFakeGitHub(t)
	f.set(func(g *fakeGitHub) { g.apiStatus = http.StatusInternalServerError })

	if _, _, err := checkPush(context.Background(), f.server.Client(), f.server.URL+"/api", "gho_test"); err == nil {
		t.Error("a 500 from GitHub was not reported")
	}
}

func TestCheckPushTimeout(t *testing.T) {
	f := newFakeGitHub(t)
	f.set(func(g *fakeGitHub) { g.apiDelay = 200 * time.Millisecond })
	client := &http.Client{Timeout: 50 * time.Millisecond}

	if _, _, err := checkPush(context.Background(), client, f.server.URL+"/api", "gho_test"); err == nil {
		t.Error("a GitHub slower than the timeout was not reported")
	}
}
```

- [ ] **Step 3: Run them to verify they fail**

Run: `go test ./internal/auth/ -run TestCheckPush -v`
Expected: FAIL, `undefined: checkPush`

- [ ] **Step 4: Implement**

`go-app/internal/auth/github.go`:
```go
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// checkPush asks GitHub, with the visitor's own token, who they are and
// whether they may push to GateRepo. GateRepo is public, so the token needs
// no scopes for this.
func checkPush(ctx context.Context, client *http.Client, apiBaseURL, token string) (login string, push bool, err error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := getJSON(ctx, client, apiBaseURL+"/user", token, &user); err != nil {
		return "", false, err
	}
	if user.Login == "" {
		return "", false, errors.New("GitHub /user returned no login")
	}

	var repo struct {
		Permissions struct {
			Push bool `json:"push"`
		} `json:"permissions"`
	}
	if err := getJSON(ctx, client, apiBaseURL+"/repos/"+GateRepo, token, &repo); err != nil {
		return user.Login, false, err
	}
	return user.Login, repo.Permissions.Push, nil
}

func getJSON(ctx context.Context, client *http.Client, url, token string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(into); err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	return nil
}
```

- [ ] **Step 5: Run them to verify they pass**

Run: `go test -race ./internal/auth/ -v`
Expected: PASS, no race reports

- [ ] **Step 6: Commit**

```bash
git add go-app/internal/auth/github.go go-app/internal/auth/github_test.go go-app/internal/auth/fake_github_test.go
git commit -m "auth: check push access with the visitor's token

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 5: `Auth`, private headers, message pages, `RequirePush`

**Files:**
- Create: `go-app/internal/auth/auth.go`
- Create: `go-app/internal/auth/pages.go`
- Create: `go-app/internal/auth/middleware.go`
- Create: `go-app/internal/auth/templates/authMessage.gohtml`
- Create: `go-app/internal/auth/helpers_test.go`
- Test: `go-app/internal/auth/middleware_test.go`

**Interfaces:**
- Consumes: `Config`, `SessionCookie`, `GateRepo` (Task 2); `decodeSession`, `testKey`, `sessionNow` (Task 3).
- Produces:
  - `type Auth struct { cfg Config; oauth *oauth2.Config; client *http.Client; now func() time.Time }`
  - `func New(cfg Config) *Auth`
  - `const sessionLifetime = 8 * time.Hour`, `const stateLifetime = 10 * time.Minute`
  - `func setPrivateHeaders(w http.ResponseWriter, siteBaseURL string)`
  - `type page struct { Title, Text string; Retry bool; SiteBaseURL string }`; constructors `notConfigured()`, `cancelled()`, `expired()`, `unreachable()`, `forbidden(login string)`
  - `func (a *Auth) render(w http.ResponseWriter, status int, p page)`
  - `func (a *Auth) unusable(w http.ResponseWriter) bool`
  - `func (a *Auth) RequirePush(next http.Handler) http.Handler`
  - `func LoginFrom(ctx context.Context) string`
  - test helpers: `testConfig(githubURL string) Config`, `newTestAuth(t *testing.T, githubURL string) *Auth`, `findCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie`, `assertPrivateHeaders(t *testing.T, rec *httptest.ResponseRecorder)`

- [ ] **Step 1: Write the test helpers**

`go-app/internal/auth/helpers_test.go`:
```go
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
```

- [ ] **Step 2: Write the failing middleware tests**

`go-app/internal/auth/middleware_test.go`:
```go
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
```

- [ ] **Step 3: Run them to verify they fail**

Run: `go test ./internal/auth/ -run TestRequirePush -v`
Expected: FAIL, `undefined: New`

- [ ] **Step 4: Implement `auth.go`**

`go-app/internal/auth/auth.go`:
```go
package auth

import (
	"embed"
	"html/template"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

const (
	sessionLifetime = 8 * time.Hour
	stateLifetime   = 10 * time.Minute
)

//go:embed templates/*.gohtml
var templatesFS embed.FS

var pageTemplate = template.Must(template.ParseFS(templatesFS, "templates/authMessage.gohtml"))

// Auth is the login in front of protected pages.
type Auth struct {
	cfg    Config
	oauth  *oauth2.Config
	client *http.Client
	now    func() time.Time
}

// New builds the login from a configuration. An unusable configuration is
// not an error here: every handler answers 503 instead (Config.Problem).
func New(cfg Config) *Auth {
	return &Auth{
		cfg: cfg,
		oauth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:   cfg.AuthorizeURL,
				TokenURL:  cfg.TokenURL,
				AuthStyle: oauth2.AuthStyleInParams,
			},
			RedirectURL: cfg.PublicBaseURL + "/auth/callback",
			// no Scopes: the push check reads a public repository only
		},
		client: &http.Client{Timeout: 10 * time.Second},
		now:    time.Now,
	}
}
```

- [ ] **Step 5: Implement `pages.go`**

`go-app/internal/auth/pages.go`:
```go
package auth

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

// setPrivateHeaders marks a response as for this visitor only: not cached,
// not readable by other origins, not framed.
func setPrivateHeaders(w http.ResponseWriter, siteBaseURL string) {
	h := w.Header()
	h.Set("Cache-Control", "private, no-store")
	h.Set("Content-Security-Policy",
		"default-src 'self'; "+
			"style-src 'self' "+siteBaseURL+"; "+
			"img-src 'self' "+siteBaseURL+" data:; "+
			"font-src "+siteBaseURL+" data:; "+
			"script-src https://plausible.io; "+
			"frame-src https://plausible.io; "+
			"frame-ancestors 'none'; "+
			// the logout form redirects to the site, which counts as a form target
			"form-action 'self' "+siteBaseURL)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "same-origin")
}

// page is every outcome of the login that is not the protected page itself.
type page struct {
	Title       string
	Text        string
	Retry       bool   // offer "sign in again"
	SiteBaseURL string // filled in by render
}

func notConfigured() page {
	return page{Title: "Login not configured",
		Text: "This service has no GitHub login set up, so the rollup page cannot be shown."}
}

func cancelled() page {
	return page{Title: "Login cancelled", Text: "You cancelled the GitHub login.", Retry: true}
}

func expired() page {
	return page{Title: "Login expired",
		Text: "The login took too long, or was started in another window. Please start again.", Retry: true}
}

func unreachable() page {
	return page{Title: "GitHub could not be reached",
		Text: "The login could not be completed. Please try again in a moment.", Retry: true}
}

func forbidden(login string) page {
	return page{Title: "No access",
		Text: "Signed in as " + login + ", but without push access to " + GateRepo + "."}
}

func (a *Auth) render(w http.ResponseWriter, status int, p page) {
	p.SiteBaseURL = a.cfg.SiteBaseURL
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := pageTemplate.Execute(w, p); err != nil {
		log.Error().Msgf("auth: rendering %q: %v", p.Title, err)
	}
}

// unusable answers 503 and returns true when the login is not configured.
func (a *Auth) unusable(w http.ResponseWriter) bool {
	if err := a.cfg.Problem(); err != nil {
		log.Error().Msgf("auth: %v", err)
		a.render(w, http.StatusServiceUnavailable, notConfigured())
		return true
	}
	return false
}
```

- [ ] **Step 6: Implement `middleware.go`**

`go-app/internal/auth/middleware.go`:
```go
package auth

import (
	"context"
	"net/http"
)

type loginKey struct{}

// RequirePush lets a request through only with a valid session cookie, and
// hands the GitHub login to the next handler (LoginFrom). Without one, the
// visitor is sent to the login. Push access was checked when the session was
// issued; it is not checked again until the session expires.
func (a *Auth) RequirePush(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setPrivateHeaders(w, a.cfg.SiteBaseURL)
		if a.unusable(w) {
			return
		}

		c, err := r.Cookie(SessionCookie)
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}
		s, ok := decodeSession(c.Value, a.cfg.SessionKey, a.now())
		if !ok {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), loginKey{}, s.Login)))
	})
}

// LoginFrom returns the GitHub login RequirePush let through, or "".
func LoginFrom(ctx context.Context) string {
	login, _ := ctx.Value(loginKey{}).(string)
	return login
}
```

- [ ] **Step 7: Create the message template**

`go-app/internal/auth/templates/authMessage.gohtml`:
```html
<!doctype html>
<!-- golang template for every login outcome that is not the rollup page
     itself (ADR-0022): not configured, cancelled, expired, GitHub unreachable,
     no push access. Served from the service's host, so the stylesheet and the
     way back are absolute against .SiteBaseURL. -->
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="robots" content="noindex">
    <title>{{ .Title }} · arc42 status</title>
    <link rel="stylesheet" href="{{ .SiteBaseURL }}/assets/css/main.css">
</head>
<body>
<main class="site-page" data-site="rollup">
    <div class="site-page__band">
        <h1 class="site-page__name">{{ .Title }}</h1>
    </div>
    <div class="site-page__body">
        <p class="site-page__tagline">{{ .Text }}</p>
        <p class="site-page__links">
            {{ if .Retry }}<a class="site-page__link" href="/auth/login">Sign in with GitHub again</a>{{ end }}
            <a class="site-page__link" href="{{ .SiteBaseURL }}/">&#8592; Back to the dashboard</a>
        </p>
    </div>
</main>
</body>
</html>
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test -race ./internal/auth/ -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add go-app/internal/auth/
git commit -m "auth: session middleware, private headers and message pages

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 6: Login, callback, logout

**Files:**
- Create: `go-app/internal/auth/handlers.go`
- Test: `go-app/internal/auth/handlers_test.go`

**Interfaces:**
- Consumes: everything from Tasks 2–5; `newFakeGitHub` (Task 4); `oauth2.GenerateVerifier`, `oauth2.S256ChallengeOption`, `oauth2.VerifierOption`, `oauth2.HTTPClient`.
- Produces: `func (a *Auth) Login(w http.ResponseWriter, r *http.Request)`, `func (a *Auth) Callback(w http.ResponseWriter, r *http.Request)`, `func (a *Auth) Logout(w http.ResponseWriter, r *http.Request)`.

- [ ] **Step 1: Write the failing tests**

`go-app/internal/auth/handlers_test.go`:
```go
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
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/auth/ -run 'TestLogin|TestCallback|TestLogout|TestHandlers' -v`
Expected: FAIL, `a.Login undefined`

- [ ] **Step 3: Implement**

`go-app/internal/auth/handlers.go`:
```go
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

// Login sends the visitor to GitHub. The state and the PKCE verifier travel
// in a signed, short-lived cookie and are checked on the way back.
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	setPrivateHeaders(w, a.cfg.SiteBaseURL)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if a.unusable(w) {
		return
	}

	state, err := randomToken()
	if err != nil {
		log.Error().Msgf("auth: no randomness for state: %v", err)
		http.Error(w, "login unavailable", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()
	issued := strconv.FormatInt(a.now().Unix(), 10)

	http.SetCookie(w, a.cookie(StateCookie, sign(state+"|"+verifier+"|"+issued, a.cfg.SessionKey), "/auth", stateLifetime))
	http.Redirect(w, r, a.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

// Callback finishes the login: state check, code exchange, push check, session.
// It always ends at /rollup and accepts no return URL, so it cannot be used
// as an open redirect.
func (a *Auth) Callback(w http.ResponseWriter, r *http.Request) {
	setPrivateHeaders(w, a.cfg.SiteBaseURL)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if a.unusable(w) {
		return
	}

	q := r.URL.Query()
	http.SetCookie(w, a.clearCookie(StateCookie, "/auth"))

	switch q.Get("error") {
	case "":
	case "access_denied":
		a.render(w, http.StatusOK, cancelled())
		return
	default:
		log.Error().Msgf("auth: GitHub refused the login: %s", q.Get("error"))
		a.render(w, http.StatusBadGateway, unreachable())
		return
	}

	verifier, ok := a.checkState(r, q.Get("state"))
	if !ok {
		log.Warn().Msg("auth: login state missing, expired or mismatched")
		a.render(w, http.StatusBadRequest, expired())
		return
	}

	ctx := context.WithValue(r.Context(), oauth2.HTTPClient, a.client)
	token, err := a.oauth.Exchange(ctx, q.Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		log.Error().Msgf("auth: code exchange failed: %v", err)
		a.render(w, http.StatusBadGateway, unreachable())
		return
	}

	login, push, err := checkPush(ctx, a.client, a.cfg.APIBaseURL, token.AccessToken)
	if err != nil {
		log.Error().Msgf("auth: push check failed: %v", err)
		a.render(w, http.StatusBadGateway, unreachable())
		return
	}
	if !push {
		log.Warn().Msgf("auth: %s forbidden", login)
		a.render(w, http.StatusForbidden, forbidden(login))
		return
	}

	log.Info().Msgf("auth: %s granted", login)
	value := encodeSession(session{Login: login, Expires: a.now().Add(sessionLifetime)}, a.cfg.SessionKey)
	http.SetCookie(w, a.cookie(SessionCookie, value, "/", sessionLifetime))
	http.Redirect(w, r, "/rollup", http.StatusFound)
}

// Logout ends the session and returns to the site. POST only.
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	setPrivateHeaders(w, a.cfg.SiteBaseURL)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.SetCookie(w, a.clearCookie(SessionCookie, "/"))
	http.Redirect(w, r, a.cfg.SiteBaseURL+"/", http.StatusFound)
}

// checkState compares the query's state with the one signed into the state
// cookie, and returns the PKCE verifier when they match and are fresh.
func (a *Auth) checkState(r *http.Request, state string) (string, bool) {
	c, err := r.Cookie(StateCookie)
	if err != nil || state == "" {
		return "", false
	}
	value, ok := verify(c.Value, a.cfg.SessionKey)
	if !ok {
		return "", false
	}
	parts := strings.Split(value, "|")
	if len(parts) != 3 {
		return "", false
	}
	issued, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || a.now().After(time.Unix(issued, 0).Add(stateLifetime)) {
		return "", false
	}
	if subtle.ConstantTimeCompare([]byte(parts[0]), []byte(state)) != 1 {
		return "", false
	}
	return parts[1], true
}

func (a *Auth) cookie(name, value, path string, lifetime time.Duration) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: value, Path: path,
		MaxAge:   int(lifetime.Seconds()),
		HttpOnly: true, Secure: a.cfg.SecureCookies, SameSite: http.SameSiteLaxMode,
	}
}

func (a *Auth) clearCookie(name, path string) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: "", Path: path, MaxAge: -1,
		HttpOnly: true, Secure: a.cfg.SecureCookies, SameSite: http.SameSiteLaxMode,
	}
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
```

- [ ] **Step 4: Run all auth tests with the race detector**

Run: `go test -race ./internal/auth/ -v`
Expected: PASS, no race reports

- [ ] **Step 5: Vet**

Run: `go vet ./internal/auth/`
Expected: no output

- [ ] **Step 6: Commit**

```bash
git add go-app/internal/auth/handlers.go go-app/internal/auth/handlers_test.go
git commit -m "auth: github login, callback with push check, logout

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 7: The maintainers-only rollup page in the service

**Files:**
- Create: `go-app/internal/api/rollupPage.gohtml`
- Delete: `go-app/internal/api/rollup.gohtml`
- Modify: `go-app/internal/api/apiGateway.go` (const `RollupTmpl`, `rollupHandler`, imports, `StartAPIServer`)
- Modify: `go-app/internal/types/rollup.go` (`RollupStats` comment, `RollupPageData`)
- Modify: `go-app/cmd/rendercheck/main.go` (rollup render, new link check, exit)
- Modify: `docs/assets/css/arc42-status-style.css` (add `.rollup__session`, `.site-plausible--rollup`)

**Interfaces:**
- Consumes: `auth.New`, `auth.FromEnv`, `(*Auth).RequirePush`, `Login`, `Callback`, `Logout`, `auth.LoginFrom`, `(Config).Problem` (Tasks 2–6); `env.SiteBaseURL`, `env.GetEnv` (Task 1).
- Produces: `types.RollupPageData{ Rollup RollupStats; LastUpdatedString, Login, SiteBaseURL, ShareURL string }`; route `/rollup` (full page, GET, protected); routes `/auth/login`, `/auth/callback`, `/auth/logout`.

- [ ] **Step 1: Make rendercheck expect the page (failing check first)**

In `go-app/cmd/rendercheck/main.go`, replace:
```go
	rollupPath := filepath.Join(outDir, "rollup.html")
	render("internal/api/rollup.gohtml", rollupPath, types.RollupPageData{
		Rollup:            stats.Rollup,
		LastUpdatedString: stats.LastUpdatedString,
	})
```
with:
```go
	// the maintainers-only rollup page (ADR-0022): a complete document served
	// from the service's host, so every link into the site must be absolute
	const fixtureSiteBaseURL = "https://status.arc42.org"
	rollupPath := filepath.Join(outDir, "rollupPage.html")
	rollupHTML := render("internal/api/rollupPage.gohtml", rollupPath, types.RollupPageData{
		Rollup:            stats.Rollup,
		LastUpdatedString: stats.LastUpdatedString,
		Login:             "octocat",
		SiteBaseURL:       fixtureSiteBaseURL,
		ShareURL:          "https://plausible.io/share/rollup.arc42.com?auth=fixture",
	})
```

Replace:
```go
	if !checkTableColumns(tableHTML) {
		os.Exit(1)
	}
}
```
with:
```go
	tableOK := checkTableColumns(tableHTML)
	linksOK := checkAbsoluteLinks(rollupHTML, fixtureSiteBaseURL)
	if !tableOK || !linksOK {
		os.Exit(1)
	}
}
```

Append after `rowWidth`:
```go
// ---------------------------------------------------------------------------
// links of the rollup page
// ---------------------------------------------------------------------------

var linkAttrRe = regexp.MustCompile(`(?i)\b(href|src)="([^"]*)"`)

// checkAbsoluteLinks reports every href and src of the rendered rollup page
// that is root-relative. The page is served from the service's host, so such
// a link would point into the service instead of the site.
func checkAbsoluteLinks(html, siteBaseURL string) bool {
	fmt.Println("\nlinks of the rendered rollup page")
	fmt.Println("---------------------------------")

	ok, stylesheet := true, false
	for _, m := range linkAttrRe.FindAllStringSubmatch(html, -1) {
		value := m[2]
		if value == siteBaseURL+"/assets/css/main.css" {
			stylesheet = true
		}
		if strings.HasPrefix(value, "/") {
			fmt.Printf("  RELATIVE %s=%q\n", m[1], value)
			ok = false
		}
	}
	if !stylesheet {
		fmt.Printf("  MISSING  stylesheet %s/assets/css/main.css\n", siteBaseURL)
		ok = false
	}
	if ok {
		fmt.Println("  every href and src is absolute, stylesheet present")
	}
	return ok
}
```
Add `"strings"` to the import block of `cmd/rendercheck/main.go` if it is not there yet.

- [ ] **Step 2: Extend `RollupPageData`**

In `go-app/internal/types/rollup.go`, replace:
```go
// RollupStats is everything the rollup row and the /rollup/ page render.
type RollupStats struct {
	// Unique holds the rollup's own figures, fetched exactly like a site's.
	// The table's rollup row reads them directly.
	Unique SiteStatsType
```
with:
```go
// RollupStats is everything the maintainers-only /rollup page renders (ADR-0022).
type RollupStats struct {
	// Unique holds the rollup's own figures, fetched exactly like a site's.
	Unique SiteStatsType
```

and replace:
```go
// RollupPageData is what the /rollup fragment renders.
type RollupPageData struct {
	Rollup            RollupStats
	LastUpdatedString string
}
```
with:
```go
// RollupPageData is what the maintainers-only /rollup page renders (ADR-0022).
type RollupPageData struct {
	Rollup            RollupStats
	LastUpdatedString string

	Login       string // GitHub login of the signed-in maintainer
	SiteBaseURL string // env.SiteBaseURL: the page is served from the service's host
	ShareURL    string // Plausible shared link for rollup.arc42.com; "" when not configured
}
```

- [ ] **Step 3: Run rendercheck to verify it fails**

Run: `source ./set-api-keys.sh >/dev/null && go run ./cmd/rendercheck /tmp/rendercheck`
Expected: panic `open internal/api/rollupPage.gohtml: no such file or directory`

- [ ] **Step 4: Create the page template**

`go-app/internal/api/rollupPage.gohtml`:
```html
<!doctype html>
<!-- golang template for the maintainers-only rollup page (ADR-0022) -->
<!--
  Served at /rollup behind auth.RequirePush. A complete document, not a
  fragment: nothing about the rollup may be published on the static site, so
  this is the one page the service renders itself. It is served from the
  service's host, so the stylesheet and every link into the site are absolute
  against .SiteBaseURL (cmd/rendercheck checks this).

  The explanation is the former docs/_pages/rollup.md; the numbers are the
  former rollup.gohtml fragment (ADR-0021).

  Class contract for the stylesheet:
    .site-page[data-site="rollup"]  band, tagline, links (neutral band)
    .rollup__session                "signed in as" line with the logout button
    .rollup                         the comparison, == #rollupFigures
      .rollup-figures, .rollup-figures__key, .rollup__note,
      .rollup__subhead, .rollup__roster > li, .rollup__since, .detail__foot
    .site-plausible--rollup         the embed's size (no inline style: CSP)
-->
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="robots" content="noindex">
    <title>All arc42 sites, counted once · arc42 status</title>
    <link rel="stylesheet" href="{{ .SiteBaseURL }}/assets/css/main.css">
</head>
<body>
<main class="page__content">

<div class="site-page" data-site="rollup">
    <div class="site-page__band">
        <h1 class="site-page__name">All arc42 sites, counted once</h1>
    </div>
    <div class="site-page__body">
        <p class="site-page__tagline">The rollup is one Plausible dashboard that every member site reports
            into besides its own. A person who reads arc42.org and then the docs is two visitors in
            the table on the home page, and one visitor here.</p>
        <p class="site-page__links">
            {{ if .ShareURL }}<a class="site-page__link" href="{{ .ShareURL }}">Open the rollup dashboard &#8599;</a>{{ end }}
            <a class="site-page__link" href="{{ .SiteBaseURL }}/">&#8592; Back to the dashboard</a>
        </p>
        <form class="rollup__session" method="post" action="/auth/logout">
            Signed in as <b>{{ .Login }}</b>, with push access to arc42/status.arc42.org-site.
            <button type="submit">Sign out</button>
        </form>
    </div>
</div>

<h2 id="numbers">The numbers</h2>

<div id="rollupFigures" class="rollup">

    <table class="rollup-figures">
        <caption class="visually-hidden">
            The rollup dashboard compared with the sum of its member sites' own dashboards,
            over 7 days, 30 days and 12 months. "n/a" means not measured, or not comparable.
        </caption>
        <thead>
        <tr>
            <th scope="col"><span class="visually-hidden">Measure</span></th>
            {{ range .Rollup.Windows }}<th scope="col">{{ .Label }}</th>{{ end }}
        </tr>
        </thead>
        <tbody>
        <tr>
            <th scope="row">Visitors, member sites' dashboards added up</th>
            {{ range .Rollup.Windows }}<td>{{ .MemberVisitors }}</td>{{ end }}
        </tr>
        <tr>
            <th scope="row">Unique visitors in the rollup</th>
            {{ range .Rollup.Windows }}<td>{{ .UniqueVisitors }}</td>{{ end }}
        </tr>
        <tr class="rollup-figures__key">
            <th scope="row">Counted on more than one site</th>
            {{ range .Rollup.Windows }}<td>{{ .MultiSite }}</td>{{ end }}
        </tr>
        <tr>
            <th scope="row">Page views, member sites added up</th>
            {{ range .Rollup.Windows }}<td>{{ .MemberPageViews }}</td>{{ end }}
        </tr>
        <tr>
            <th scope="row">Page views in the rollup</th>
            {{ range .Rollup.Windows }}<td>{{ .RollupPageViews }}</td>{{ end }}
        </tr>
        </tbody>
    </table>

    {{ range .Rollup.Windows }}{{ if not .Complete }}
    <p class="rollup__note"><b>{{ .Label }}: not comparable yet.</b>
        {{ range $i, $key := .Joiners }}{{ if $i }}, {{ end }}{{ $key }}{{ end }}
        joined the rollup inside this window, so their own dashboards count visits the
        rollup never received. The window becomes comparable once it lies entirely after
        the latest join.</p>
    {{ end }}{{ end }}

    <h3 class="rollup__subhead">Reporting into the rollup</h3>
    {{ if .Rollup.Members }}
    <ul class="rollup__roster">
        {{ range .Rollup.Members }}
        <li><a href="{{ $.SiteBaseURL }}/site/{{ .Key }}/">{{ .Key }}</a> <span class="rollup__since">since {{ .Since }}</span></li>
        {{ end }}
    </ul>
    {{ else }}
    <p class="detail__empty">No property is declared as reporting into the rollup.</p>
    {{ end }}

    {{ if .Rollup.Pending }}
    <h3 class="rollup__subhead">Joining: snippet changed, not deployed yet</h3>
    <ul class="rollup__roster">
        {{ range .Rollup.Pending }}<li><a href="{{ $.SiteBaseURL }}/site/{{ . }}/">{{ . }}</a></li>{{ end }}
    </ul>
    {{ end }}

    {{ if .Rollup.Outside }}
    <h3 class="rollup__subhead">Measured, but not in the rollup</h3>
    <ul class="rollup__roster">
        {{ range .Rollup.Outside }}<li><a href="{{ $.SiteBaseURL }}/site/{{ . }}/">{{ . }}</a></li>{{ end }}
    </ul>
    {{ end }}

    <p class="detail__foot">Collected {{ .LastUpdatedString }}.</p>
</div>

<h2 id="how">How the rollup works</h2>

<ul>
    <li><b>Every member reports twice.</b> Its snippet names two Plausible sites, its own first:
        <code>data-domain="docs.arc42.org,rollup.arc42.com"</code>. Each page view goes to both.
        The member's own dashboard, and every figure on the home page, stay exactly as they were.</li>
    <li><b>One person, one visitor, across sites.</b> Plausible sets no cookies. It recognises a
        visitor by a hash of the IP address, the browser's user agent, the site the page view is
        sent to, and a salt that changes every day. Every member sends to the same site,
        <code>rollup.arc42.com</code>, so the same person gets the same hash on every member site that day.</li>
    <li><b>One visit, across sites.</b> A visit ends after 30 minutes without a page view. Reading
        arc42.org/overview and then three docs pages within that time is one visit with four page
        views, in the rollup.</li>
    <li><b>The old script, on purpose.</b> Plausible's newer per-site snippet cannot report into two
        dashboards, so every member uses the legacy script (meta.arc42.org ADR-0005).</li>
    <li><b>Cost.</b> Every page view on a member site is billed twice.</li>
</ul>

<h2 id="reading">Reading the rollup</h2>

<table>
    <thead>
    <tr><th scope="col">Figure</th><th scope="col">Rollup compared with the members added up</th><th scope="col">What it tells you</th></tr>
    </thead>
    <tbody>
    <tr><td>Page views</td><td>Equal</td><td>Every page view is sent to both. A gap means a snippet is missing somewhere, or a member joined inside the window.</td></tr>
    <tr><td>Unique visitors</td><td>Lower</td><td>The realistic count of people. Members added up minus the rollup = people counted on more than one site.</td></tr>
    <tr><td>Visits</td><td>Lower</td><td>Moving from arc42.org into the docs continues the visit instead of starting a second one.</td></tr>
    <tr><td>Bounce rate</td><td>Lower</td><td>Clicking from <code>/overview</code> into the docs is no longer a bounce.</td></tr>
    <tr><td>Visit duration, pages per visit</td><td>Higher</td><td>Time and pages on every member site add up within one visit.</td></tr>
    <tr><td>Top sources</td><td>Different</td><td>Where the visit <i>started</i> (a search engine, LinkedIn, a bookmark). Moving to another member site inside a visit is not a source here, while a member's own dashboard lists arc42.org as a referrer.</td></tr>
    <tr><td>Entry and exit pages</td><td>Different</td><td>First and last page of the whole visit, across sites. An exit on arc42.org/overview now means the reader really left.</td></tr>
    <tr><td>Outbound link clicks</td><td>Include moves between members</td><td>A click from arc42.org to the docs is movement inside the family, not a departure. Filter those URLs out to see real exits.</td></tr>
    </tbody>
</table>

<h2 id="traps">Traps</h2>

<ol>
    <li><b>Page lists show paths without host names.</b> <code>/</code> from every member is one row, and
        arc42.de mirrors arc42.org's paths should it ever join. Filter by <i>Hostname</i> before
        reading Top Pages, Entry Pages or Exit Pages. A useful check: <i>Hostname is docs.arc42.org</i>
        should roughly match the docs dashboard's own figures.</li>
    <li><b>Long windows count returning people again.</b> The daily salt means someone who comes
        back on three days is three visitors in a 30-day window. That is true of every Plausible
        dashboard, the members' included, so compare windows with each other, not with a head count.</li>
    <li><b>Joins look like growth.</b> A member that starts reporting adds its traffic from that day.
        The table above says "not comparable yet" for every window that reaches back before the
        latest join, and lists each member's join date.</li>
    <li><b>The rollup began on 2026-07-30.</b> Before that no page view reached it, so the 12-month
        figures cover less than a year until 2027-07-30.</li>
    <li><b>Some readers are never counted.</b> Ad blockers and Firefox's tracking protection block
        Plausible. This affects the rollup and the members' dashboards alike.</li>
</ol>

<h2 id="questions">Three questions and where to look</h2>

<ul>
    <li><b>How many people use arc42's sites?</b> The rollup's <i>Unique visitors</i>, unfiltered.</li>
    <li><b>Where do people who reach the trainings start?</b> Rollup dashboard, filter
        <i>Hostname is trainings.arc42.org</i>, then <i>Entry Pages</i>.</li>
    <li><b>How many use more than one site?</b> "Counted on more than one site" in the table above.</li>
</ul>

<p>Plausible does not record page sequences, so no report here can show "the ten most common
    paths through arc42". What it can show honestly is where visits start and end, which sites
    they touch, and how pages are read.</p>

<h2 id="dashboard">The rollup dashboard</h2>

{{ if .ShareURL }}
<p class="site-page__dashlink">
    Live from Plausible, privacy-friendly and cookie-free.
    <a href="{{ .ShareURL }}">Open this dashboard on plausible.io &#8599;</a>
</p>
<iframe class="site-plausible site-plausible--rollup" plausible-embed
        title="Plausible analytics dashboard for the arc42 rollup"
        src="{{ .ShareURL }}&amp;embed=true&amp;theme=light"
        scrolling="no" frameborder="0" loading="lazy"></iframe>
<script async src="https://plausible.io/js/embed.host.js"></script>
{{ else }}
<p class="site-page__nodata">The dashboard link is not configured (PLAUSIBLE_ROLLUP_SHARE_URL).</p>
{{ end }}

</main>
</body>
</html>
```

- [ ] **Step 5: Delete the old fragment**

Run: `git rm go-app/internal/api/rollup.gohtml`

- [ ] **Step 6: Add the two CSS classes**

In `docs/assets/css/arc42-status-style.css`, directly after the `.site-plausible { … }` rule, insert:
```css
/* the rollup page's embed (ADR-0022): the size the other embeds carry inline,
   as a class, because the page's Content-Security-Policy forbids inline styles */
.site-plausible--rollup {
    width: 1px;
    min-width: 100%;
    height: 1600px;
}

/* "signed in as" on the maintainers-only rollup page, with the logout button */
.rollup__session {
    margin: 0.6em 0 0;
    font-size: 0.85em;
    color: var(--status-muted);
}

.rollup__session button {
    margin-left: 0.6em;
    font: inherit;
    cursor: pointer;
}
```

- [ ] **Step 7: Wire the handler and routes**

In `go-app/internal/api/apiGateway.go`:

Replace `const RollupTmpl = "rollup.gohtml"` with `const RollupPageTmpl = "rollupPage.gohtml"`.

Add to the import block: `"arc42-status/internal/auth"` and `"arc42-status/internal/env"`.

Replace the whole `rollupHandler` function (its comment included) with:
```go
// rollupPageHandler renders the maintainers-only rollup page (ADR-0022): the
// rollup's unique counts beside the sum of its members' own dashboards, which
// properties report into it since when, and the embedded dashboard.
//
// It runs behind auth.RequirePush, which has set the private headers and put
// the visitor's GitHub login into the request context. Unlike every fragment
// it sets no CORS headers: no other origin may read this page. It reads the
// same cached collection run as every other handler.
func rollupPageHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domain.ArcStats = domain.Stats4AllSites()

	go database.SaveInvocationParams(r.Host, r.RequestURI)

	executeTemplate(w, filepath.Join(TemplatesDir, RollupPageTmpl), types.RollupPageData{
		Rollup:            domain.ArcStats.Rollup,
		LastUpdatedString: domain.ArcStats.LastUpdatedString,
		Login:             auth.LoginFrom(r.Context()),
		SiteBaseURL:       env.SiteBaseURL(env.GetEnv()),
		ShareURL:          strings.TrimSpace(os.Getenv("PLAUSIBLE_ROLLUP_SHARE_URL")),
	})
}
```

In `StartAPIServer`, replace:
```go
	mux.HandleFunc("/rollup", rollupHandler)
```
with:
```go
	// the maintainers-only rollup page and its GitHub login (ADR-0022)
	authCfg := auth.FromEnv(env.GetEnv())
	if err := authCfg.Problem(); err != nil {
		log.Warn().Msgf("maintainer login unavailable, /rollup answers 503: %v", err)
	}
	gate := auth.New(authCfg)
	mux.Handle("/rollup", gate.RequirePush(http.HandlerFunc(rollupPageHandler)))
	mux.HandleFunc("/auth/login", gate.Login)
	mux.HandleFunc("/auth/callback", gate.Callback)
	mux.HandleFunc("/auth/logout", gate.Logout)
```

- [ ] **Step 8: Build, vet, test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: builds; vet silent; all packages `ok` or `[no test files]`.

- [ ] **Step 9: Run rendercheck to verify it passes**

Run: `source ./set-api-keys.sh >/dev/null && go run ./cmd/rendercheck /tmp/rendercheck`
Expected: ends with
```
links of the rendered rollup page
---------------------------------
  every href and src is absolute, stylesheet present
```
and exit status 0. The table section still shows two tfoot rows (removed in Task 8).

- [ ] **Step 10: Commit**

```bash
git add go-app/internal/api/ go-app/internal/types/rollup.go go-app/cmd/rendercheck/main.go docs/assets/css/arc42-status-style.css
git commit -m "rollup page served by the service behind the maintainer login

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 8: Remove the public rollup

**Files:**
- Modify: `go-app/internal/api/arc42statistics.gohtml:12-22,49,114-122`
- Modify: `go-app/internal/types/types.go:487-489` (comment)
- Modify: `docs/assets/css/arc42-status-style.css` (delete `.stats-rollup` rules)
- Delete: `docs/_pages/rollup.md`
- Modify: `docs/_pages/home.md:46-51` (comment)
- Modify: `documentation/adrs/0021-report-the-rollup-beside-the-summed-totals.md` (Status)

**Interfaces:** none new.

- [ ] **Step 1: Drop the footer row**

In `go-app/internal/api/arc42statistics.gohtml`, replace:
```html
    </tr>
    <tr class="stats-rollup">
        <th scope="row" class="text-right"><a href="/rollup/">Unique across sites</a></th>
        <td class="border-left-black"></td>
        <td class="border-left-black text-right">{{ .Rollup.Unique.Visitors7d }}</td>
        <td class="text-right">{{ .Rollup.Unique.PageViews7d }}</td>
        <td class="border-left-black text-right">{{ .Rollup.Unique.Visitors30d }}</td>
        <td class="text-right">{{ .Rollup.Unique.PageViews30d }}</td>
        <td class="border-left-black text-right">{{ .Rollup.Unique.Visitors12m }}</td>
        <td class="text-right">{{ .Rollup.Unique.PageViews12m }}</td>
    </tr> </tfoot>
```
with:
```html
    </tr> </tfoot>
```

Delete the caption line:
```html
        The last row is the rollup dashboard, which counts a person visiting several arc42 sites once; it is not a sum of the rows.
```

Replace the comment block:
```
  It also carries fewer rows than the dashboard has tiles. Only properties
  marked InTable appear here (types.Arc42properties): a column is read downwards,
  and rows that do not compare with each other - a CLI's landing page, a
  course-date feed, a repository with no site at all - make that reading worse
  rather than more complete. Their numbers are on their own subpages. The totals
  row sums exactly the rows above it, so the reader can check it by adding up.

  The second footer row is NOT a total, and is labelled and linked so it cannot
  be read as one: it is the rollup dashboard's own count (ADR-0021), where a
  person reading two sites counts once. Its roster differs from the rows above
  (see /rollup/), so it will not equal their sum, and is not meant to.
```
with:
```
  It also carries fewer rows than the dashboard has tiles. Only properties
  marked InTable appear here (types.Arc42properties): a column is read downwards,
  and rows that do not compare with each other - a CLI's landing page, a
  repository with no site at all - make that reading worse rather than more
  complete. Their numbers are on their own subpages. The totals row sums
  exactly the rows above it, so the reader can check it by adding up.

  The rollup's unique count is not shown here: it is for maintainers only, on
  the service's /rollup page (ADR-0022).
```

- [ ] **Step 2: Run rendercheck to verify the table now has one footer row**

Run: `source ./set-api-keys.sh >/dev/null && go run ./cmd/rendercheck /tmp/rendercheck`
Expected: `tfoot row 1:  8 columns   ok`, no `tfoot row 2`, `header, body and footer agree`, exit status 0.

- [ ] **Step 3: Update the `Rollup` field comment**

In `go-app/internal/types/types.go`, replace:
```go
	// Rollup: the shared rollup dashboard's own counts, compared with its
	// members' (ADR-0021). Not a total: a person on two sites counts once.
	Rollup RollupStats
```
with:
```go
	// Rollup: the shared rollup dashboard's own counts, compared with its
	// members' (ADR-0021). Not a total: a person on two sites counts once.
	// Rendered only on the maintainers-only /rollup page (ADR-0022).
	Rollup RollupStats
```

- [ ] **Step 4: Delete the `.stats-rollup` CSS**

In `docs/assets/css/arc42-status-style.css`, delete:
```css
/* the table's second footer row: quieter than the totals, because it is a
   different kind of number, and its label is the link that explains why */
.stats-rollup th,
.stats-rollup td {
    color: var(--status-muted);
    font-weight: 400;
}

.stats-rollup th a {
    font-weight: 600;
}

```

- [ ] **Step 5: Delete the public page**

Run: `git rm docs/_pages/rollup.md`

- [ ] **Step 6: Fix the home page skeleton comment**

In `docs/_pages/home.md`, replace:
```
      11 rows x 8 columns: two header rows, the seven sites the table carries
      (types.Arc42properties, InTable), the totals row and the rollup row
      (ADR-0021), and the site / status / 3x(visitors, pageviews) columns the
      service returns -- the same shape, so the page barely moves when the
      real table lands.
```
with:
```
      11 rows x 8 columns: two header rows, the eight sites the table carries
      (types.Arc42properties, InTable), the totals row, and the site / status /
      3x(visitors, pageviews) columns the service returns -- the same shape, so
      the page barely moves when the real table lands.
```

- [ ] **Step 7: Mark ADR-0021 as partly superseded**

In `documentation/adrs/0021-report-the-rollup-beside-the-summed-totals.md`, replace:
```
## Status

Accepted
```
with:
```
## Status

Accepted. The public page `/rollup/` and the table's "Unique across sites" row are
superseded by ADR-0022: the rollup is shown to maintainers only.
```

- [ ] **Step 8: Check nothing public still mentions the rollup page**

Run (from the repository root): `grep -rn 'rollup/\|stats-rollup\|Unique across sites' docs/_pages docs/_includes docs/assets go-app/internal/api`
Expected: no output.

- [ ] **Step 9: Test**

Run: `go test ./...`
Expected: all `ok`.

- [ ] **Step 10: Commit**

```bash
git add -A docs/_pages docs/_pages/home.md docs/assets/css/arc42-status-style.css go-app/internal/api/arc42statistics.gohtml go-app/internal/types/types.go documentation/adrs/0021-report-the-rollup-beside-the-summed-totals.md
git commit -m "remove the public rollup page and footer row (ADR-0022)

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 9: Local setup, ADR-0022, version

**Files:**
- Modify: `go-app/set-api-keys.sh.template`
- Modify: `Makefile` (`doctor`)
- Modify: `README.md`
- Create: `documentation/adrs/0022-show-the-rollup-only-to-maintainers.md`
- Modify: `go-app/main.go` (`appVersion`, history)

**Interfaces:** none new.

- [ ] **Step 1: Extend the secrets template**

In `go-app/set-api-keys.sh.template`, insert before `# DEV keeps the service on the development database`:
```bash
# ---- maintainer login for the rollup page (ADR-0022) ----------------------
# Without the first four, /rollup answers 503; everything else runs as usual.

# GitHub OAuth App "arc42 status (local)", authorization callback URL
#   http://localhost:8043/auth/callback
export GITHUB_OAUTH_CLIENT_ID=<oauth-app-client-id>
export GITHUB_OAUTH_CLIENT_SECRET=<oauth-app-client-secret>

# Signs the login cookies. Generate with:  openssl rand -base64 32
# Changing it signs every maintainer out.
export SESSION_KEY=<openssl-rand-base64-32>

# Where this service is reachable; the OAuth callback URL is built from it.
export PUBLIC_BASE_URL="http://localhost:8043"

# Plausible shared link for rollup.arc42.com, without &embed=... .
# Secret: whoever holds it can read the rollup dashboard.
export PLAUSIBLE_ROLLUP_SHARE_URL=<plausible-share-link>

```

- [ ] **Step 2: Report the login variables in `make doctor`**

In `Makefile`, directly after the block that prints `[ok]   %s present` / `[fail] %s missing` for `$(SECRETS)`, insert:
```make
	@if [ -f $(SECRETS) ]; then \
		for var in GITHUB_OAUTH_CLIENT_ID GITHUB_OAUTH_CLIENT_SECRET SESSION_KEY PUBLIC_BASE_URL PLAUSIBLE_ROLLUP_SHARE_URL; do \
			if grep -Eq "^export $$var=\"?[^\"<]+" $(SECRETS); then \
				printf "  [ok]   %s set\n" "$$var"; \
			else \
				printf "  [warn] %s not set — maintainer login (/rollup) incomplete, see README\n" "$$var"; \
			fi; \
		done; \
	fi
```
Recipe lines must be indented with a tab.

- [ ] **Step 3: Verify doctor**

Run: `make doctor`
Expected: five lines `[ok]` or `[warn]` for the login variables; the rest unchanged. A copy of the template without real values shows `[warn]` for the four placeholders and `[ok]` for `PUBLIC_BASE_URL`.

- [ ] **Step 4: README section**

In `README.md`, insert before `### Database & Schema Management (Atlas)`:
```markdown
### Maintainer login (rollup page)

`/rollup` on the service is for people with push access to
`arc42/status.arc42.org-site` only (ADR-0022). To use it locally:

1. Create a GitHub OAuth App (GitHub → Settings → Developer settings → OAuth Apps),
   homepage `http://localhost:4270`, authorization callback URL
   `http://localhost:8043/auth/callback`.
2. Put its client ID and a new client secret into `go-app/set-api-keys.sh`, together with
   `SESSION_KEY` (`openssl rand -base64 32`), `PUBLIC_BASE_URL=http://localhost:8043` and
   `PLAUSIBLE_ROLLUP_SHARE_URL` (see `set-api-keys.sh.template`).
3. `make backend`, then open http://localhost:8043/rollup.

Production uses a second OAuth App with callback `https://arc42-stats.fly.dev/auth/callback`;
its values are fly secrets (`make fly-secrets` lists them). Without the login variables
`/rollup` answers 503. Locally the site's web fonts may not load on the page, because the
Jekyll dev server sends no CORS headers for them.

```

- [ ] **Step 5: Write ADR-0022**

`documentation/adrs/0022-show-the-rollup-only-to-maintainers.md`:
```markdown
# 22. show the rollup only to maintainers

Date: 2026-09-15

## Status

Accepted. Supersedes the public page `/rollup/` and the table's "Unique across sites" row
of ADR-0021.

## Context

ADR-0021 published the rollup: a footer row with its unique visitors under the traffic
table, and a page `/rollup/` with the comparison, an explanation and the embedded shared
dashboard. The owner decided that these figures are for the people who maintain the arc42
sites, not for the public.

This repository is public, and `docs/` is served statically by GitHub Pages. Whatever a
Jekyll page contains, including a Plausible share link with its `auth=` token, can be read
in the repository and its history. A login in front of static content protects nothing.

## Decision

- The Go service renders the rollup page itself, at `/rollup`, behind a GitHub login.
- Only a GitHub user with push access to `arc42/status.arc42.org-site` gets in. At login
  the service reads `permissions.push` of `GET /repos/arc42/status.arc42.org-site` with the
  user's token, obtained through a GitHub OAuth App without scopes (PKCE and `state`).
- A successful login sets an HMAC-signed, HttpOnly session cookie holding the login and an
  8-hour expiry. Neither the token nor the session is stored.
- The public page, the footer row and the public `/rollup` fragment are removed. The
  collector keeps querying `rollup.arc42.com`.
- The rollup's Plausible share link is rotated and kept in the secret
  `PLAUSIBLE_ROLLUP_SHARE_URL`.
- Rejected: oauth2-proxy and Cloudflare Access. Both check organisation or team membership,
  neither checks push access to one repository, and both add infrastructure for one page.

Design: `documentation/specs/2026-09-15-rollup-login-design.md`.

## Consequences

- Someone whose push access is removed can read the page until their session ends, at most
  8 hours later. Rotating `SESSION_KEY` ends every session at once.
- Two GitHub OAuth Apps (production, local) and five environment variables have to be
  maintained (ADR-0018).
- Without the login variables `/rollup` answers 503; the rest of the service is unaffected.
- The page is served from `arc42-stats.fly.dev`, so its stylesheet and its links into the
  site are absolute (`env.SiteBaseURL`). Locally the site's web fonts may not load on it.
- The old share link stays in the git history, dead once it is revoked in Plausible.
```

- [ ] **Step 6: Bump the version**

In `go-app/main.go`, replace `const appVersion = "1.4.0"` with `const appVersion = "1.5.0"`, and insert above the `// 1.4.0:` history line:
```go
// 1.5.0: the rollup is for maintainers only (ADR-0022). /rollup is a complete
//        page behind a GitHub login (push access to arc42/status.arc42.org-site);
//        the public /rollup/ page, the table's "Unique across sites" row and the
//        public fragment are gone.
```

- [ ] **Step 7: Build and test**

Run: `make build && make test`
Expected: builds; all tests `ok`.

- [ ] **Step 8: Commit**

```bash
git add go-app/set-api-keys.sh.template Makefile README.md documentation/adrs/0022-show-the-rollup-only-to-maintainers.md go-app/main.go
git commit -m "maintainer login: local setup, ADR-0022, version 1.5.0

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 10: Verify end to end, then roll out

Steps marked **(maintainer)** need GitHub, Plausible or fly.io accounts and are done by the maintainer, not by an agent.

**Files:** `go-app/set-api-keys.sh` (local, gitignored) only.

- [ ] **Step 1 (maintainer): Create the local OAuth App** and fill the five variables into `go-app/set-api-keys.sh`, as in the README section.

- [ ] **Step 2: Full login locally**

Run `make backend` (terminal 1) and `make site` (terminal 2). Open http://localhost:8043/rollup.
Expected: redirect to GitHub; after authorising, back on `/rollup`, page renders with "Signed in as <login>", the figures table and the dashboard.

- [ ] **Step 3: Headers**

Run: `curl -sI http://localhost:8043/rollup`
Expected: `302`, `Location: /auth/login`, `Cache-Control: private, no-store`, a `Content-Security-Policy`, no `Access-Control-Allow-Origin`.

- [ ] **Step 4: Without push access**

Sign in with a GitHub account that has no push access to `arc42/status.arc42.org-site` (private window).
Expected: 403 page "Signed in as <login>, but without push access to arc42/status.arc42.org-site."

- [ ] **Step 5: Logout**

Click "Sign out". Expected: back on http://localhost:4270/; opening http://localhost:8043/rollup asks for login again.

- [ ] **Step 6: Public site**

Open http://localhost:4270. Expected: traffic table with eight sites and one "Totals" footer row, no "Unique across sites". http://localhost:4270/rollup/ is 404.

- [ ] **Step 7: Merge**

After review: merge `rollup-login` into `main` (use superpowers:finishing-a-development-branch).

- [ ] **Step 8 (maintainer): Production OAuth App**, callback `https://arc42-stats.fly.dev/auth/callback`.

- [ ] **Step 9 (maintainer): New Plausible shared link** for `rollup.arc42.com`. Keep the old one for now.

- [ ] **Step 10 (maintainer): Secrets**

```bash
cd go-app && flyctl secrets set \
  GITHUB_OAUTH_CLIENT_ID=… GITHUB_OAUTH_CLIENT_SECRET=… \
  SESSION_KEY="$(openssl rand -base64 32)" \
  PUBLIC_BASE_URL=https://arc42-stats.fly.dev \
  PLAUSIBLE_ROLLUP_SHARE_URL=…
```

- [ ] **Step 11 (maintainer): Deploy service, then site, then revoke — back to back**

1. `make fly-deploy` — from now on the footer row is gone and `/rollup` requires login.
2. `git push` of `main` — GitHub Pages removes `/rollup/`.
3. In Plausible, delete the old shared link (`auth=_Uc9YHaNOglp6i0arDxiE`).

- [ ] **Step 12: Production checks**

Repeat Steps 2–6 against `https://arc42-stats.fly.dev/rollup` and `https://status.arc42.org`. Also open the old share link: expected to be rejected by Plausible.
