package env

import "testing"

func TestSiteBaseURL(t *testing.T) {
	cases := map[string]string{
		"PROD": "https://status.arc42.org",
		"DEV":  "http://localhost:4046",
		"TEST": "http://localhost:4046",
	}
	for environment, want := range cases {
		if got := SiteBaseURL(environment); got != want {
			t.Errorf("SiteBaseURL(%q) = %q, want %q", environment, got, want)
		}
	}
}
