package types

import (
	"reflect"
	"strconv"
	"testing"
	"time"
)

var rollupNow = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

// measuredSite gives the same visitor and page-view figures to all three
// horizons, which is all BuildRollup's arithmetic needs.
func measuredSite(key string, visitors, pageViews int) SiteStatsType {
	v, pv := strconv.Itoa(visitors), strconv.Itoa(pageViews)
	return SiteStatsType{Site: key, HasTraffic: true,
		Visitors7d: v, Visitors7dNr: visitors, PageViews7d: pv, PageViews7dNr: pageViews,
		Visitors30d: v, Visitors30dNr: visitors, PageViews30d: pv, PageViews30dNr: pageViews,
		Visitors12m: v, Visitors12mNr: visitors, PageViews12m: pv, PageViews12mNr: pageViews,
	}
}

func unmeasuredSite(key string) SiteStatsType {
	s := SiteStatsType{Site: key}
	s.Visitors7d, s.PageViews7d = NotAvailable, NotAvailable
	s.Visitors30d, s.PageViews30d = NotAvailable, NotAvailable
	s.Visitors12m, s.PageViews12m = NotAvailable, NotAvailable
	return s
}

var rollupProps = []Property{
	{Key: "a.example", HasTraffic: true, RollupSince: "2026-07-30"},
	{Key: "b.example", HasTraffic: true, RollupSince: "2026-07-30"},
	{Key: "pending.example", HasTraffic: true, RollupPending: true},
	{Key: "outside.example", HasTraffic: true},
	{Key: "repo-only"},
}

func rollupSites() []SiteStatsType {
	return []SiteStatsType{
		measuredSite("a.example", 100, 400),
		measuredSite("b.example", 80, 200),
		measuredSite("pending.example", 1000, 5000), // must never be summed
		measuredSite("outside.example", 7000, 9000), // nor this
	}
}

func TestBuildRollupSumsMembersOnly(t *testing.T) {
	r := BuildRollup(measuredSite(RollupSiteID, 150, 600), rollupSites(), rollupProps, rollupNow)

	w := r.Windows[0]
	if w.Label != "7 days" || !w.Complete {
		t.Fatalf("7-day window: label %q, complete %v", w.Label, w.Complete)
	}
	if w.MemberVisitors != "180" || w.MemberPageViews != "600" {
		t.Errorf("member sums = %s / %s, want 180 / 600", w.MemberVisitors, w.MemberPageViews)
	}
	if w.UniqueVisitors != "150" || w.RollupPageViews != "600" {
		t.Errorf("rollup figures = %s / %s, want 150 / 600", w.UniqueVisitors, w.RollupPageViews)
	}
	if w.MultiSite != "30" {
		t.Errorf("MultiSite = %q, want 30", w.MultiSite)
	}
}

func TestBuildRollupClassifiesProperties(t *testing.T) {
	r := BuildRollup(measuredSite(RollupSiteID, 150, 600), rollupSites(), rollupProps, rollupNow)

	wantMembers := []RollupMember{{"a.example", "2026-07-30"}, {"b.example", "2026-07-30"}}
	if !reflect.DeepEqual(r.Members, wantMembers) {
		t.Errorf("Members = %v, want %v", r.Members, wantMembers)
	}
	if !reflect.DeepEqual(r.Pending, []string{"pending.example"}) {
		t.Errorf("Pending = %v", r.Pending)
	}
	// repo-only has no Plausible site: not "outside the rollup", just unmeasured
	if !reflect.DeepEqual(r.Outside, []string{"outside.example"}) {
		t.Errorf("Outside = %v", r.Outside)
	}
}

func TestBuildRollupWindowReachingBeforeTheJoin(t *testing.T) {
	r := BuildRollup(measuredSite(RollupSiteID, 150, 600), rollupSites(), rollupProps, rollupNow)

	if !r.Windows[1].Complete {
		t.Errorf("30-day window starts 2026-08-15, after both joins: should be complete")
	}

	w := r.Windows[2] // 12 months reaches back to 2025-09-14
	if w.Complete || w.MultiSite != NotAvailable {
		t.Errorf("12-month window: complete %v, MultiSite %q; want incomplete and n/a", w.Complete, w.MultiSite)
	}
	if !reflect.DeepEqual(w.Joiners, []string{"a.example", "b.example"}) {
		t.Errorf("Joiners = %v", w.Joiners)
	}
	// the measured figures themselves are still reported
	if w.MemberVisitors != "180" || w.UniqueVisitors != "150" {
		t.Errorf("figures = %s / %s, want 180 / 150", w.MemberVisitors, w.UniqueVisitors)
	}
}

func TestBuildRollupLateJoinerSpoilsShortWindows(t *testing.T) {
	props := append([]Property{{Key: "late.example", HasTraffic: true, RollupSince: "2026-09-10"}}, rollupProps...)
	sites := append(rollupSites(), measuredSite("late.example", 10, 20))

	r := BuildRollup(measuredSite(RollupSiteID, 150, 600), sites, props, rollupNow)

	for _, w := range r.Windows {
		if w.Complete || w.MultiSite != NotAvailable {
			t.Errorf("%s: complete %v, MultiSite %q; a member joined 4 days ago", w.Label, w.Complete, w.MultiSite)
		}
	}
	if got := r.Windows[0].Joiners; !reflect.DeepEqual(got, []string{"late.example"}) {
		t.Errorf("7-day Joiners = %v, want only late.example", got)
	}
}

func TestBuildRollupNeverInventsADifference(t *testing.T) {
	cases := []struct {
		name   string
		unique SiteStatsType
		sites  []SiteStatsType
	}{
		{"rollup unavailable", unmeasuredSite(RollupSiteID), rollupSites()},
		{"a member unavailable", measuredSite(RollupSiteID, 150, 600),
			[]SiteStatsType{measuredSite("a.example", 100, 400), unmeasuredSite("b.example")}},
		{"a member not collected at all", measuredSite(RollupSiteID, 150, 600),
			[]SiteStatsType{measuredSite("a.example", 100, 400)}},
		{"rollup counts more than its members", measuredSite(RollupSiteID, 999, 600), rollupSites()},
	}
	for _, c := range cases {
		r := BuildRollup(c.unique, c.sites, rollupProps, rollupNow)
		if got := r.Windows[0].MultiSite; got != NotAvailable {
			t.Errorf("%s: MultiSite = %q, want %q", c.name, got, NotAvailable)
		}
	}
}

func TestBuildRollupFormatsWithThousandsSeparators(t *testing.T) {
	sites := []SiteStatsType{measuredSite("a.example", 12000, 40000), measuredSite("b.example", 3456, 1)}
	r := BuildRollup(measuredSite(RollupSiteID, 14000, 40001), sites, rollupProps, rollupNow)

	w := r.Windows[0]
	if w.MemberVisitors != "15.456" || w.UniqueVisitors != "14.000" || w.MultiSite != "1.456" {
		t.Errorf("got %s / %s / %s, want 15.456 / 14.000 / 1.456", w.MemberVisitors, w.UniqueVisitors, w.MultiSite)
	}
}

// The declared family must stay readable by BuildRollup: a join date that does
// not parse would silently mark every window incomplete forever.
func TestDeclaredRollupRoster(t *testing.T) {
	for _, p := range Arc42properties {
		if p.RollupSince != "" {
			if _, err := time.Parse("2006-01-02", p.RollupSince); err != nil {
				t.Errorf("%s: RollupSince %q is not a 2006-01-02 date", p.Key, p.RollupSince)
			}
			if p.RollupPending {
				t.Errorf("%s: has a join date and is still marked pending", p.Key)
			}
		}
		if (p.RollupSince != "" || p.RollupPending) && !p.HasTraffic {
			t.Errorf("%s: in the rollup roster without a Plausible site of its own", p.Key)
		}
	}
}
