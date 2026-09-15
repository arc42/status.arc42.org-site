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
