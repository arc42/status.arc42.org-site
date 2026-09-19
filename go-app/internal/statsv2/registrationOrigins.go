package statsv2

import (
	"os"
	"sync"

	"arc42-status/internal/types"

	"github.com/rs/zerolog/log"
)

type originCutSpec struct {
	dimension string
	title     string
	note      string
}

// registrationTarget is one course site's registration page and the
// dashboard that can say where its visits began.
type registrationTarget struct {
	heading string
	site    string
	page    string // Plausible strips the query string, so one path covers every ?kurs=... link
	window  string

	siteID    string
	dateRange any
	inRollup  bool
	joinedOn  string
	cuts      []originCutSpec
}

// trainingsTarget: the English courses. Their question is answered in the
// shared rollup: in the trainings site's own dashboard a visit arriving from
// the docs looks like a fresh visit with a referrer, not one visit that began
// there (ADR-0023). joinedOn is the day trainings started reporting into the
// rollup; the window is all time in the rollup, which begins then.
func trainingsTarget(joinedOn string) registrationTarget {
	return registrationTarget{
		heading:   "English courses: trainings.arc42.org",
		site:      "trainings.arc42.org",
		page:      "/registration/",
		window:    "all time in the rollup",
		siteID:    "rollup.arc42.com",
		dateRange: "all",
		inRollup:  true,
		joinedOn:  joinedOn,
		cuts: []originCutSpec{
			{"visit:entry_page_hostname", "Which arc42 site they came in through", ""},
			{"visit:entry_page", "Which page they came in through",
				"Paths carry no host name here: \"/\" is the front page of whichever member site the visit started on."},
			{"visit:source", "Where they came from before arc42", ""},
		},
	}
}

// germanTarget: the German courses, which register on arc42.de. arc42.de
// reports only to its own dashboard (meta.arc42.org ADR-0005), so its block
// asks that dashboard. Every visit there enters on arc42.de, so there is no
// entry-hostname cut; an arrival from another arc42 site shows up as a source
// instead. That dashboard holds years of history, so the window is twelve
// months rather than all time (ADR-0024).
func germanTarget() registrationTarget {
	return registrationTarget{
		heading:   "German courses: arc42.de",
		site:      "arc42.de",
		page:      "/anmeldung/",
		window:    "the last 12 months",
		siteID:    "arc42.de",
		dateRange: "12mo",
		cuts: []originCutSpec{
			{"visit:entry_page", "Which arc42.de page they came in through", ""},
			{"visit:source", "Where they came from before arc42.de",
				"arc42.de is not in the rollup, so a reader who comes over from another arc42 site starts a new visit here, and that site is listed as the source."},
		},
	}
}

// RegistrationOriginsFor answers, for every course site: of the visits that
// opened the registration page, where did the visit begin? trainingsJoinedOn
// is the day trainings.arc42.org started reporting into the rollup. With no
// API key it returns nil, and the page omits the section.
func RegistrationOriginsFor(trainingsJoinedOn string) []types.RegistrationOrigins {
	return registrationOriginsAll(StatsV2Endpoint, os.Getenv("PLAUSIBLE_API_KEY"), trainingsJoinedOn)
}

func registrationOriginsAll(endpoint, token, trainingsJoinedOn string) []types.RegistrationOrigins {
	// With no token there is nothing to ask - and nothing to leak. Send no
	// request at all, so the page can omit the section.
	if token == "" {
		return nil
	}

	targets := []registrationTarget{trainingsTarget(trainingsJoinedOn), germanTarget()}

	// The blocks are independent, so they are asked concurrently: the German
	// block must not lengthen the collection pass it runs in. Each goroutine
	// writes only its own slot.
	out := make([]types.RegistrationOrigins, len(targets))
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t registrationTarget) {
			defer wg.Done()
			out[i] = registrationOriginsFrom(endpoint, token, t)
		}(i, t)
	}
	wg.Wait()
	return out
}

func registrationOriginsFrom(endpoint, token string, t registrationTarget) types.RegistrationOrigins {
	out := types.RegistrationOrigins{
		Heading:  t.heading,
		Site:     t.site,
		Page:     t.page,
		Window:   t.window,
		InRollup: t.inRollup,
		JoinedOn: t.joinedOn,
	}

	// Selects whole visits that contained a registration page view. An
	// event-level filter would select the page views themselves, whose entry
	// page is meaningless - see the spec.
	filter := []any{[]any{"has_done", []any{"is", "event:page", []string{t.page}}}}

	// anyOK tracks whether at least one cut actually returned data. Without
	// it, failed cuts would leave TotalVisits at its zero value and
	// SmallSample would read that zero as "small" - a zero standing in for
	// unknown, which the spec forbids.
	anyOK := false

	for _, spec := range t.cuts {
		cut := types.OriginCut{Title: spec.title, Note: spec.note}

		rows, err := RunV2Query(endpoint, token, V2Query{
			SiteID:     t.siteID,
			Metrics:    []string{"visitors", "visits"},
			DateRange:  t.dateRange,
			Dimensions: []string{spec.dimension},
			Filters:    filter,
			OrderBy:    []any{[]any{"visitors", "desc"}},
			Pagination: &V2Pagination{Limit: 25},
		})
		if err != nil {
			log.Warn().Msgf("registration origins (%s, %s): %v", t.site, spec.dimension, err)
			cut.Failed = true
			cut.FailureReason = err.Error()
			out.Cuts = append(out.Cuts, cut)
			continue
		}

		anyOK = true

		visits := 0
		for _, r := range rows {
			label := ""
			if len(r.Dimensions) > 0 {
				label = r.Dimensions[0]
			}
			visitors, v := 0, 0
			if len(r.Metrics) > 0 {
				visitors = r.Metrics[0]
			}
			if len(r.Metrics) > 1 {
				v = r.Metrics[1]
			}
			visits += v
			cut.Rows = append(cut.Rows, types.OriginRow{Label: label, Visitors: visitors, Visits: v})
		}
		// Every cut counts the same visits, so the total is one cut's worth,
		// never the sum of the cuts.
		if visits > out.TotalVisits {
			out.TotalVisits = visits
		}
		out.Cuts = append(out.Cuts, cut)
	}

	// SmallSample only ever qualifies a total that came from real data. When
	// every cut failed, TotalVisits is 0 for lack of any answer, not because
	// the traffic was small - so it must not read as "too small to
	// generalise from".
	out.SmallSample = anyOK && out.TotalVisits < types.SmallSampleVisits
	return out
}
