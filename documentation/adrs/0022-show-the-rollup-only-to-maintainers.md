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
