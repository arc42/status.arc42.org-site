package auth

import (
	"net/url"
	"strings"
	"testing"
)

func TestRedactedURL(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "auth callback strips query",
			raw:  "/auth/callback?code=abc&state=xyz",
			want: "/auth/callback",
		},
		{
			name: "auth login without query is unchanged",
			raw:  "/auth/login",
			want: "/auth/login",
		},
		{
			name: "non-auth path keeps its query",
			raw:  "/siteDetail?site=docs.arc42.org",
			want: "/siteDetail?site=docs.arc42.org",
		},
		{
			name: "rollup path is unchanged",
			raw:  "/rollup",
			want: "/rollup",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u, err := url.Parse(c.raw)
			if err != nil {
				t.Fatalf("url.Parse(%q): %v", c.raw, err)
			}
			got := RedactedURL(u)
			if got != c.want {
				t.Errorf("RedactedURL(%q) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestRedactedURLNeverLeaksCodeOrState(t *testing.T) {
	u, err := url.Parse("/auth/callback?code=abc&state=xyz")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	got := RedactedURL(u)
	if strings.Contains(got, "abc") || strings.Contains(got, "xyz") {
		t.Errorf("RedactedURL(%q) = %q, leaks code or state", u, got)
	}
}
