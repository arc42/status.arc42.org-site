# Rollup page for maintainers only — design

Date: 2026-09-15
Status: approved; implemented on branch rollup-login (ADR-0022)
Replaces the public parts of ADR-0021 (page `/rollup/`, footer row "Unique across sites")

## Goal

The rollup figures and the rollup dashboard are visible only to people with push access
to `arc42/status.arc42.org-site`. Nothing about the rollup remains public.

## Decisions

| Question | Decision |
|---|---|
| What is protected | The whole rollup page: explanation, comparison figures, roster, embedded dashboard |
| Who may see it | A GitHub user whose `permissions.push` on `arc42/status.arc42.org-site` is true |
| Public footer row "Unique across sites" | Removed |
| Mechanism | Login built into the Go service: GitHub OAuth App, signed session cookie (option A) |
| Rejected | oauth2-proxy (checks org/team, not repo push; second process), Cloudflare Access (extra vendor and domain, no repo-push rule) |

## Non-goals

- No redirect or stub at the old public URL `https://status.arc42.org/rollup/`; it becomes a 404.
- No roles beyond "may see the rollup". No user database, no audit table.
- The other sites' Plausible share links on public pages stay as they are.
- No journeys page (ADR-0021's Stats API v2 step stays separate).

## Why the page has to move to the service

The repository is public and `docs/` is served statically by GitHub Pages. Anything in a
Jekyll page, including the Plausible share link with its `auth=` token, is readable in the
repository and its history. A login in front of static content protects nothing. So the
page is rendered by the Go service on fly.io after the permission check, and the share
link is rotated and kept in a secret.

## Architecture

```
browser ──GET /rollup──▶ arc42-stats.fly.dev
                          │
                          ├─ auth.RequirePush ── no/invalid session ──▶ 302 /auth/login
                          │                                              │
                          │                     302 github.com/login/oauth/authorize
                          │                                              │
                          │   /auth/callback ◀────────── code, state ────┘
                          │     ├─ exchange code (+PKCE verifier) for token
                          │     ├─ GET api.github.com/user                       → login
                          │     ├─ GET api.github.com/repos/arc42/status.arc42.org-site → permissions.push
                          │     └─ push: set rollup_session, 302 /rollup   no push: 403
                          │
                          └─ session valid ──▶ api.rollupPageHandler ──▶ rollupPage.gohtml
                                                  (same cached collection run as every fragment)
```

### New package `internal/auth`

One responsibility: establish who a visitor is and whether they may see protected pages.
It knows nothing about statistics.

| File | Content |
|---|---|
| `config.go` | `Config{ClientID, ClientSecret, SessionKey, PublicBaseURL, SiteBaseURL, SecureCookies, AuthorizeURL, TokenURL, APIBaseURL}`; `FromEnv(environment string) Config`; `(Config) Problem() error` returns why login is not usable (missing value, key shorter than 32 bytes), nil when usable. The GitHub URLs default to github.com and are overridden only by tests. |
| `session.go` | `sign(value, key)`, `verify(cookieValue, key)`; session value `login\|expiryUnix`; cookie value `base64url(value) + "." + base64url(HMAC-SHA256(value))`; constant-time comparison (`hmac.Equal`). |
| `github.go` | `checkPush(ctx, token) (login string, push bool, err error)`: two GETs with `Authorization: Bearer`, `Accept: application/vnd.github+json`, `X-GitHub-Api-Version: 2022-11-28`, 10 s timeout. Gate repository is the constant `GateRepo = "arc42/status.arc42.org-site"`. |
| `handlers.go` | `(*Auth) Login`, `Callback`, `Logout` |
| `middleware.go` | `(*Auth) RequirePush(http.Handler) http.Handler`; `LoginFrom(ctx) string` |
| `templates/authMessage.gohtml` | One small page for every outcome that is not the rollup (cancelled, expired, forbidden, unavailable), embedded with `embed.FS` |
| `*_test.go` | See Testing |

`auth.New(Config) *Auth` builds the `oauth2.Config` (`golang.org/x/oauth2`, already a
dependency; endpoint from `Config.AuthorizeURL` and `Config.TokenURL` so tests can point it
at a fake GitHub, credentials sent in the request body, no scopes,
redirect `PublicBaseURL + "/auth/callback"`).

### Changes in `internal/api`

- `StartAPIServer` registers:
  - `/rollup` → `a.RequirePush(rollupPageHandler)`, GET only
  - `/auth/login`, `/auth/callback` (GET), `/auth/logout` (POST)
- `rollupHandler` (fragment, CORS `*`) is replaced by `rollupPageHandler`, which renders
  `rollupPage.gohtml`. It does not call `SetCORSHeaders`.
- `rollupPage.gohtml` is a complete HTML document: the content of today's
  `docs/_pages/rollup.md` (band, explanation, "Reading the rollup", "Traps",
  "Three questions") plus today's `rollup.gohtml` table and roster, the embedded
  dashboard, "Signed in as *login*" and a logout form. `rollup.gohtml` is deleted.
- `types.RollupPageData` gains `Login`, `SiteBaseURL`, `ShareURL`.
- `arc42statistics.gohtml` loses the `.stats-rollup` row and the caption sentence about it.
  `domain.ArcStats.Rollup` stays: the collector still queries `rollup.arc42.com`, the
  table just stops rendering it.

### Links and assets from the service's host

The page is served from `arc42-stats.fly.dev`, so every link into the site is absolute:

| Environment | `SiteBaseURL` |
|---|---|
| PROD | `https://status.arc42.org` |
| DEV, TEST | `http://localhost:4046` |

Used for the stylesheet (`/assets/css/main.css`, which imports `arc42-status-style.css`;
its fonts load from `/assets/fonts/` on the same host), `/site/<key>/` roster links, and
"back to the dashboard". Defined next to `env.GetEnv()` as `env.SiteBaseURL(environment)`.

## Login flow

1. `GET /rollup` without a valid `rollup_session` → 302 `/auth/login`.
2. `GET /auth/login`:
   - generate `state` (32 random bytes, base64url) and a PKCE verifier (`oauth2.GenerateVerifier`)
   - set `rollup_oauth` = signed `state|verifier`; Path `/auth`, Max-Age 600, HttpOnly, SameSite=Lax, Secure unless DEV/TEST
   - 302 to `AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))`
3. `GET /auth/callback?code=…&state=…`:
   - `error=access_denied` → "Login cancelled" page (200), clear `rollup_oauth`
   - `rollup_oauth` missing, invalid, or its state ≠ query `state` → 400 "Login expired, please start again"
   - exchange code with `oauth2.VerifierOption(verifier)`; failure → 502
   - `checkPush`; GitHub error → 502
   - push false → 403 "Signed in as *login*, but without push access to arc42/status.arc42.org-site"
   - push true → set `rollup_session` = signed `login|now+8h`; Path `/`, Max-Age 28800, HttpOnly, SameSite=Lax, Secure unless DEV/TEST; clear `rollup_oauth`; 302 `/rollup`
   - The token is used only inside this request: never stored, never logged.
4. `POST /auth/logout` → clear `rollup_session`, 302 `SiteBaseURL + "/"`.

The target after login is always `/rollup`. No return URL is accepted, so there is no open
redirect.

## Failure cases

| Situation | Status | Page / action |
|---|---|---|
| Login not configured (`Config.Problem() != nil`) | 503 for `/rollup`, `/auth/login`, `/auth/callback` | "Login not configured"; the rest of the service runs. `/auth/logout` is unaffected: it still clears the cookie and redirects (harmless, since there can be no valid session to clear) |
| Visitor cancels on GitHub | 200 | "Login cancelled", link to try again |
| `state` missing, wrong, or older than 10 minutes | 400 | "Login expired, please start again" |
| Code exchange or GitHub API fails, or times out | 502 | "GitHub could not be reached", no cookie, error logged |
| Signed in without push access | 403 | Names the login and the gate repository |
| Session expired, tampered, or signed with an old key | — | Treated as no session: 302 `/auth/login` |
| Method other than GET on `/auth/login`, `/auth/callback`; other than POST on `/auth/logout` | 405 | — |
| Method other than GET on `/rollup` | 405 with a valid session; without one, `RequirePush` runs first and any method is treated like a GET: 302 `/auth/login` | — |

## Security

- Session and state cookies are HMAC-SHA256 signed with `SESSION_KEY` (≥ 32 bytes after
  base64 decoding); verification is constant-time.
- Push access is checked at login only. Removing someone's access takes effect at their
  next login, at most 8 hours later. Rotating `SESSION_KEY` ends every session at once.
- Responses of `/rollup` and `/auth/*`:
  - no `Access-Control-Allow-*` headers
  - `Cache-Control: private, no-store`
  - `Content-Security-Policy: default-src 'self'; style-src 'self' <SiteBaseURL>; img-src 'self' <SiteBaseURL> data:; font-src <SiteBaseURL> data:; script-src https://plausible.io; frame-src https://plausible.io; frame-ancestors 'none'; form-action 'self' <SiteBaseURL>` (the logout form's redirect to the site counts as a form target)
  - `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin`
- Because of `style-src`, the dashboard iframe's inline `style` attribute moves into a
  class in `arc42-status-style.css`.
- Logging: login attempt outcome with the GitHub login (`auth: <login> granted|forbidden`),
  failures with the error. Never the code, token, or cookie values.
- The rollup share link comes from `PLAUSIBLE_ROLLUP_SHARE_URL`. The link committed so far
  (`auth=_Uc9YHaNOglp6i0arDxiE`) is revoked in Plausible; the history then holds a dead link.

## Configuration

Environment variables (ADR-0018): fly secrets in PROD, `set-api-keys.sh` locally.

| Variable | PROD | DEV |
|---|---|---|
| `GITHUB_OAUTH_CLIENT_ID` | production OAuth App | local OAuth App |
| `GITHUB_OAUTH_CLIENT_SECRET` | production OAuth App | local OAuth App |
| `SESSION_KEY` | `openssl rand -base64 32` | `openssl rand -base64 32` |
| `PUBLIC_BASE_URL` | `https://arc42-stats.fly.dev` | `http://localhost:8043` |
| `PLAUSIBLE_ROLLUP_SHARE_URL` | new rollup share link | same link |

Two GitHub OAuth Apps, created by the maintainer (preferably owned by the arc42 org, so
they do not depend on one personal account):

| App | Homepage | Authorization callback URL |
|---|---|---|
| arc42 status (production) | `https://status.arc42.org` | `https://arc42-stats.fly.dev/auth/callback` |
| arc42 status (local) | `http://localhost:4046` | `http://localhost:8043/auth/callback` |

`make doctor` reports each missing variable as `[warn]`, not `[fail]`.

## Testing

All tests run offline. GitHub is an `httptest.Server` whose URL is put into
`Config.AuthorizeURL`, `TokenURL`, `APIBaseURL`. Written test-first.

`internal/auth`:

- session: round trip; rejected when login altered, expiry altered, expired, signed with a
  different key, malformed (no dot, bad base64)
- config: `Problem()` for each missing variable and for a short key; nil when complete
- middleware: no cookie → 302 `/auth/login`; valid cookie → handler runs and
  `LoginFrom` returns the login; unusable config → 503
- login: sets `rollup_oauth`, redirect carries `state`, `code_challenge`,
  `code_challenge_method=S256`, the configured `redirect_uri`, and no scope
- callback against fake GitHub:
  - push true → `rollup_session` set, 302 `/rollup`
  - push false → 403, no session cookie
  - wrong state, missing state cookie → 400
  - `error=access_denied` → cancelled page
  - token endpoint 500, API 500, API slower than the timeout → 502
  - a `redirect`/`next` query parameter is ignored
- logout: cookie cleared (Max-Age < 0), 302 to site; GET → 405
- headers: `/rollup` and `/auth/*` carry no `Access-Control-Allow-Origin` and carry
  `Cache-Control: private, no-store` and the CSP

`internal/api` / `cmd/rendercheck`:

- rendercheck renders `rollupPage.gohtml` with fixture data and checks that stylesheet and
  `/site/` links start with `SiteBaseURL`
- rendercheck's table arithmetic reports one footer row (totals only)

Manual, before deploying:

1. `make backend` with the local OAuth App; `/rollup` → GitHub → back → page renders
2. A GitHub account without push access → 403 page naming it
3. Log out → `/rollup` asks for login again
4. `make site`: home table has no "Unique across sites" row; `/rollup/` is 404

## Documentation and cleanup

- New `documentation/adrs/0022-show-the-rollup-only-to-maintainers.md`: context (public
  repository, static site), decision (this design, option A), consequences (8-hour window,
  rotated share link, OAuth Apps to maintain, two new secrets). ADR-0021's status gets
  "Public page and footer row superseded by ADR-0022".
- Delete `docs/_pages/rollup.md`, `go-app/internal/api/rollup.gohtml`.
- `docs/_pages/home.md` skeleton comment: two header rows, eight sites (trainings is in the
  table again), the totals row — still 11 rows; the rollup row is no longer mentioned.
- `set-api-keys.sh.template`: the five variables with a comment each.
- `main.go`: version 1.5.0, history line.
- README: short "Maintainer login (rollup)" section pointing at the OAuth App setup.

## Rollout

1. Create both OAuth Apps; put the local values into `set-api-keys.sh`.
2. Run the manual checks 1–3 of the Testing section locally, with the local OAuth App,
   before anything is merged.
3. In Plausible, create a new shared link for `rollup.arc42.com`.
4. `flyctl secrets set GITHUB_OAUTH_CLIENT_ID=… GITHUB_OAUTH_CLIENT_SECRET=… SESSION_KEY=… PUBLIC_BASE_URL=https://arc42-stats.fly.dev PLAUSIBLE_ROLLUP_SHARE_URL=…`
5. Merge `rollup-login` into `main`. This one push deploys the service (`fly.yml`) and the
   site (`pages.yml`) together: the footer row, the public `/rollup/` page and the public
   `/rollup` fragment all disappear in this same step.
6. In Plausible, delete the old shared link right after.
7. Run the manual checks against production.

Merging is deploying: there is no separate "push the deploy" step afterwards, and no window
during which the old public page stays reachable. This makes the order of steps 4 and 5
load-bearing: merging before the fly secrets are set does not leave the old public page
running by accident, but it does mean the new, maintainer-only `/rollup` answers 503 for
maintainers too until the secrets are set — the service fails closed rather than open.
