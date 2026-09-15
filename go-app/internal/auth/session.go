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
