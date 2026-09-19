package types

import (
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// RollupSiteID is the Plausible site every rollup member reports into besides
// its own (meta.arc42.org ADR-0005). No host serves it: it exists to be named in
// each member's data-domain, and to be asked about here.
const RollupSiteID = "rollup.arc42.com"

// RollupWindow compares the rollup with its members for one horizon.
//
// The comparison is the point. A member's own dashboard counts a person who
// read arc42.org and then the docs once per site; the rollup counts them once.
// The difference between the two is the number of people who used more than
// one arc42 site - but only when both sides counted the same traffic.
type RollupWindow struct {
	Label string // "7 days", "30 days", "12 months"

	MemberVisitors  string // summed over the members' own dashboards
	UniqueVisitors  string // the rollup's own count
	MultiSite       string // MemberVisitors - UniqueVisitors, or NotAvailable
	MemberPageViews string
	RollupPageViews string

	// Complete is false when a member joined the rollup inside this window.
	// Its own dashboard then counts traffic the rollup never received, so
	// MultiSite would be an artefact of the join date, not a reading.
	// Joiners names the members responsible.
	Complete bool
	Joiners  []string
}

// RollupMember is one property reporting into the rollup, with the day it
// started to.
type RollupMember struct {
	Key   string
	Since string
}

// RollupStats is everything the maintainers-only /rollup page renders (ADR-0022).
type RollupStats struct {
	// Unique holds the rollup's own figures, fetched exactly like a site's.
	Unique SiteStatsType

	Windows []RollupWindow // 7 days, 30 days, 12 months

	Members []RollupMember // reporting, with their join date
	Pending []string       // snippet in the repository, not deployed yet
	Outside []string       // measured, but not reporting into the rollup
}

// RollupPageData is what the maintainers-only /rollup page renders (ADR-0022).
type RollupPageData struct {
	Rollup              RollupStats
	RegistrationOrigins []RegistrationOrigins
	LastUpdatedString   string

	Login       string // GitHub login of the signed-in maintainer
	SiteBaseURL string // env.SiteBaseURL: the page is served from the service's host
	ShareURL    string // Plausible shared link for rollup.arc42.com; "" when not configured
}

// BuildRollup compares the rollup's own figures with the sum of its members'
// dashboards, for the three horizons the table shows.
//
// It never invents a difference. MultiSite stays NotAvailable when the rollup
// or any member could not be measured, when a member joined inside the window,
// or when the rollup counts more visitors than its members together - which
// cannot happen with a correct roster, so it means the roster is wrong (a
// member deployed but still marked pending), not that nobody crossed sites.
func BuildRollup(unique SiteStatsType, sites []SiteStatsType, props []Property, now time.Time) RollupStats {
	r := RollupStats{Unique: unique}

	byKey := make(map[string]SiteStatsType, len(sites))
	for _, s := range sites {
		byKey[s.Site] = s
	}

	var members []Property
	for _, p := range props {
		switch {
		case p.RollupSince != "":
			members = append(members, p)
			r.Members = append(r.Members, RollupMember{Key: p.Key, Since: p.RollupSince})
		case p.RollupPending:
			r.Pending = append(r.Pending, p.Key)
		case p.HasTraffic:
			r.Outside = append(r.Outside, p.Key)
		}
	}

	type figures struct {
		visitors    string
		visitorsNr  int
		pageViews   string
		pageViewsNr int
	}
	horizons := []struct {
		label string
		start time.Time
		pick  func(SiteStatsType) figures
	}{
		{"7 days", now.AddDate(0, 0, -7), func(s SiteStatsType) figures {
			return figures{s.Visitors7d, s.Visitors7dNr, s.PageViews7d, s.PageViews7dNr}
		}},
		{"30 days", now.AddDate(0, 0, -30), func(s SiteStatsType) figures {
			return figures{s.Visitors30d, s.Visitors30dNr, s.PageViews30d, s.PageViews30dNr}
		}},
		{"12 months", now.AddDate(-1, 0, 0), func(s SiteStatsType) figures {
			return figures{s.Visitors12m, s.Visitors12mNr, s.PageViews12m, s.PageViews12mNr}
		}},
	}

	p := message.NewPrinter(language.German)

	for _, h := range horizons {
		w := RollupWindow{Label: h.label, Complete: true}

		membersMeasured := true
		sumVisitors, sumPageViews := 0, 0
		for _, m := range members {
			since, err := time.Parse("2006-01-02", m.RollupSince)
			if err != nil || since.After(h.start) {
				w.Complete = false
				w.Joiners = append(w.Joiners, m.Key)
			}

			f := h.pick(byKey[m.Key])
			if !isMeasured(f.visitors) || !isMeasured(f.pageViews) {
				membersMeasured = false
			}
			sumVisitors += f.visitorsNr
			sumPageViews += f.pageViewsNr
		}

		u := h.pick(unique)
		uniqueMeasured := isMeasured(u.visitors)

		w.MemberVisitors = formatRollupNr(p, membersMeasured, sumVisitors)
		w.MemberPageViews = formatRollupNr(p, membersMeasured, sumPageViews)
		w.UniqueVisitors = formatRollupNr(p, uniqueMeasured, u.visitorsNr)
		w.RollupPageViews = formatRollupNr(p, isMeasured(u.pageViews), u.pageViewsNr)

		w.MultiSite = NotAvailable
		if membersMeasured && uniqueMeasured && w.Complete && sumVisitors >= u.visitorsNr {
			w.MultiSite = p.Sprintf("%d", sumVisitors-u.visitorsNr)
		}

		r.Windows = append(r.Windows, w)
	}

	return r
}

// isMeasured treats the empty string like NotAvailable: a member with no
// collected row at all has not been measured either.
func isMeasured(s string) bool {
	return s != "" && s != NotAvailable
}

func formatRollupNr(p *message.Printer, measured bool, n int) string {
	if !measured {
		return NotAvailable
	}
	return p.Sprintf("%d", n)
}
