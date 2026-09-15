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
		log.Debug().Msgf("auth: %v", err)
		a.render(w, http.StatusServiceUnavailable, notConfigured())
		return true
	}
	return false
}
