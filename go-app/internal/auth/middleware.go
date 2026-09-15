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
