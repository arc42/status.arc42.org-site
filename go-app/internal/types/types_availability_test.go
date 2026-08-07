package types

import "testing"

func TestSiteAvailabilityToken(t *testing.T) {
	cases := []struct {
		name string
		av   SiteAvailability
		want string
	}{
		{"never measured", SiteAvailability{Measured: false, State: "up"}, "unmonitored"},
		{"stale overrides state", SiteAvailability{Measured: true, Stale: true, State: "up"}, "unknown"},
		{"fresh up", SiteAvailability{Measured: true, State: "up"}, "up"},
		{"fresh down", SiteAvailability{Measured: true, State: "down"}, "down"},
		{"fresh degraded", SiteAvailability{Measured: true, State: "degraded"}, "degraded"},
	}
	for _, c := range cases {
		if got := c.av.Token(); got != c.want {
			t.Errorf("%s: Token() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFamilyAvailabilityToken(t *testing.T) {
	cases := []struct {
		name string
		f    FamilyAvailability
		want string
	}{
		{"never measured", FamilyAvailability{}, "unmonitored"},
		{"stale", FamilyAvailability{Measured: true, Stale: true}, "unknown"},
		{"one down wins", FamilyAvailability{Measured: true, NrDown: 1, NrDegraded: 2}, "down"},
		{"degraded without down", FamilyAvailability{Measured: true, NrDegraded: 1}, "degraded"},
		{"all up", FamilyAvailability{Measured: true, AllUp: true}, "up"},
	}
	for _, c := range cases {
		if got := c.f.Token(); got != c.want {
			t.Errorf("%s: Token() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestMonitored(t *testing.T) {
	if Monitored(Property{Key: "arc42-template"}) {
		t.Error("hostless property must not be monitored")
	}
	if Monitored(Property{Key: "examples.arc42.org", Host: "examples.arc42.org", Planned: true}) {
		t.Error("planned property must not be monitored")
	}
	if Monitored(Property{Key: "meta.arc42.org", Host: "meta.arc42.org", NoProbe: true}) {
		t.Error("NoProbe property must not be monitored")
	}
	if !Monitored(Property{Key: "arc42.org", Host: "arc42.org"}) {
		t.Error("hosted, built property must be monitored")
	}
}

// TestExamplesIsMonitored pins down a specific case of TestMonitored: once
// examples.arc42.org-site existed on GitHub and the host resolved
// (2026-08-07), its Arc42properties entry lost Planned: true and picked up
// a Repo, which - via Monitored() - also puts it on the prober's list. That
// combination is easy to break silently (e.g. NoProbe added by habit while
// copying a neighbouring entry), so it is asserted directly rather than only
// via the registry-wide checks below.
func TestExamplesIsMonitored(t *testing.T) {
	for _, p := range Arc42properties {
		if p.Key != "examples.arc42.org" {
			continue
		}
		if !Monitored(p) {
			t.Error("examples.arc42.org must be monitored now that the site exists")
		}
		return
	}
	t.Fatal("examples.arc42.org is no longer in Arc42properties")
}

func TestEveryMonitoredPropertyDeclaresExpectedContent(t *testing.T) {
	for _, p := range Arc42properties {
		if Monitored(p) && p.ExpectedContent == "" {
			t.Errorf("%s is monitored but declares no ExpectedContent", p.Key)
		}
	}
}

func TestProbePathIsAbsolute(t *testing.T) {
	for _, p := range Arc42properties {
		if p.ProbePath != "" && p.ProbePath[0] != '/' {
			t.Errorf("%s: ProbePath %q must start with /", p.Key, p.ProbePath)
		}
	}
}
