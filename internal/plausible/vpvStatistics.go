// Package plausible interacts with the https://plausible.io web statistics service,
// that counts visitors and pageviews of the arc42 sites.
package plausible

// thanx and credits to Andre
// https://github.com/andrerfcsantos/go-plausible
// wrapping https://plausible.io/docs
// ==============================================

import (
	"github.com/andrerfcsantos/go-plausible/plausible"
	"github.com/rs/zerolog/log"
	"os"
	"site-usage-statistics/internal/types"
	"strconv"
)

// plausibleClient wraps the plausible API.
// The required (secret) API key is set within the initialization.
var plausibleClient *plausible.Client = nil

// initPlausibleHandler gets the plausible API key
// and creates a handler (NewClient) to perform queries upon
func initPlausibleHandler() *plausible.Client {

	if plausibleClient == nil {

		// APIKEY is a personal key for https://plausible.io
		// It needs to be set via environment variable
		var APIKEY string = os.Getenv("PLAUSIBLE_API_KEY")

		if APIKEY == "" {
			// no value, no API calls, no results.
			// we exit here, as we have no chance of recovery
			log.Error().Msgf("CRITICAL ERROR: required plausible API key not set.\n" +
				"You need to set the 'PLAUSIBLE_API_KEY' environment variable prior to launching this application.\n")

			os.Exit(13)
		} else {
			log.Debug().Msg("PLAUSIBLE_API_KEY set")
		}
		return plausible.NewClient(APIKEY)
	} else {
		return plausibleClient
	}
}

// StatsForSite collects all relevant statistics for a given site
// (currently 7D, 30D and 12M), and updates the Sums accordingly
func StatsForSite(thisSite string, stats *types.SiteStats, totals *types.TotalsForAllSites) {

	// init the required handler
	// the function ensures it's initialized only once.
	plausibleClient = initPlausibleHandler()

	// Get a handler to perform queries for a given site
	siteHandler := plausibleClient.Site(thisSite)

	// Get the different metrics
	var stats7D types.VisitorsAndPageViews = SiteMetrics(siteHandler, plausible.Last7Days())
	stats.Visitors7d = stats7D.Visitors
	stats.Pageviews7d = stats7D.Pageviews
	totals.SumOfVisitors7d += stats7D.VisitorNr
	totals.SumOfPageviews7d += stats7D.PageViewNr

	var stats30D = SiteMetrics(siteHandler, plausible.Last30Days())
	stats.Visitors30d = stats30D.Visitors
	stats.Pageviews30d = stats30D.Pageviews
	totals.SumOfVisitors30d += stats30D.VisitorNr
	totals.SumOfPageviews30d += stats30D.PageViewNr

	var stats12M = SiteMetrics(siteHandler, plausible.Last12Months())
	stats.Visitors12m = stats12M.Visitors
	stats.Pageviews12m = stats12M.Pageviews
	totals.SumOfVisitors12m += stats12M.VisitorNr
	totals.SumOfPageviews12m += stats12M.PageViewNr

}

// SiteMetrics collects statics for given site and period from plausible.io API.
// Return either the numbers or "n/a" in case of API errors
// Updates the TotalsForAllSites
func SiteMetrics(siteHandler *plausible.Site, period plausible.TimePeriod) types.VisitorsAndPageViews {

	var vApvs types.VisitorsAndPageViews

	// Build query
	siteMetricsQuery := plausible.AggregateQuery{
		Period: period,
		Metrics: plausible.Metrics{
			plausible.Visitors,
			plausible.PageViews,
		},
	}

	// Execute query to plausible.io
	result, err := siteHandler.Aggregate(siteMetricsQuery)
	if err != nil {
		log.Error().Msgf("Error performing query to plausible.io: %v", err)
		// in this case, we don't add anything to the Sums
		vApvs.Pageviews = "n/a"
		vApvs.PageViewNr = 0
		vApvs.Visitors = "n/a"
		vApvs.VisitorNr = 0
	} else {
		vApvs.Pageviews = strconv.Itoa(result.Pageviews)
		vApvs.PageViewNr = result.Pageviews
		vApvs.Visitors = strconv.Itoa(result.Visitors)
		vApvs.VisitorNr = result.Visitors
	}

	return vApvs
}
