package auth

import (
	"net/url"
	"strings"
)

// RedactedURL returns a URL safe to write to the request log. Paths under
// /auth/ carry the OAuth code and state as query parameters (e.g.
// /auth/callback?code=...&state=...), so their query string is stripped;
// every other path is returned unchanged.
func RedactedURL(u *url.URL) string {
	if strings.HasPrefix(u.Path, "/auth/") {
		return u.Path
	}
	return u.String()
}
