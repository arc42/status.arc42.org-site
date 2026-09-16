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
