package plausible

import (
	"os"

	"arc42-status/internal/types"

	"github.com/rs/zerolog/log"
)

// registrationPath is the page whose visits this report is about. Plausible
// strips the query string, so this single path covers every ?kurs=... link.
const registrationPath = "/registration/"

// rollupSiteID is the shared dashboard. The question can only be answered
// here: in the trainings site's own dashboard a visit arriving from the docs
// looks like a fresh visit with a referrer, not one visit that began there.
const rollupSiteID = "rollup.arc42.com"

type originCutSpec struct {
	dimension string
	title     string
	note      string
}

var originCuts = []originCutSpec{
	{"visit:entry_page_hostname", "Which arc42 site they came in through", ""},
	{"visit:entry_page", "Which page they came in through",
		"Paths carry no host name here: \"/\" is the front page of whichever member site the visit started on."},
	{"visit:source", "Where they came from before arc42", ""},
}

// RegistrationOriginsFor answers: of the visits that opened the registration
// page, where did the visit begin? joinedOn is the day trainings started
// reporting into the rollup; windows reaching back further under-report, and
// the page says so.
func RegistrationOriginsFor(joinedOn string) types.RegistrationOrigins {
	return registrationOriginsFrom(StatsV2Endpoint, os.Getenv("PLAUSIBLE_API_KEY"), joinedOn)
}

func registrationOriginsFrom(endpoint, token, joinedOn string) types.RegistrationOrigins {
	out := types.RegistrationOrigins{JoinedOn: joinedOn}

	// R3: with no token there is nothing to ask - and nothing to leak. Send
	// no request at all, and leave the section without cuts so the page can
	// omit it.
	if token == "" {
		return out
	}

	// Selects whole visits that contained a registration page view. An
	// event-level filter would select the page views themselves, whose entry
	// page is meaningless - see the spec.
	filter := []any{[]any{"has_done", []any{"is", "event:page", []string{registrationPath}}}}

	for _, spec := range originCuts {
		cut := types.OriginCut{Title: spec.title, Note: spec.note}

		rows, err := RunV2Query(endpoint, token, V2Query{
			SiteID:     rollupSiteID,
			Metrics:    []string{"visitors", "visits"},
			DateRange:  "all",
			Dimensions: []string{spec.dimension},
			Filters:    filter,
			OrderBy:    []any{[]any{"visitors", "desc"}},
			Pagination: &V2Pagination{Limit: 25},
		})
		if err != nil {
			log.Warn().Msgf("registration origins (%s): %v", spec.dimension, err)
			cut.Failed = true
			cut.FailureReason = err.Error()
			out.Cuts = append(out.Cuts, cut)
			continue
		}

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
		// never the sum of all three.
		if visits > out.TotalVisits {
			out.TotalVisits = visits
		}
		out.Cuts = append(out.Cuts, cut)
	}

	out.SmallSample = out.TotalVisits < types.SmallSampleVisits
	return out
}
