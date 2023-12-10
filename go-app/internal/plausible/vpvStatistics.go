// Package plausible interacts with the https://plausible.io web statistics service,
// that counts visitors and pageviews of the arc42 sites.
package plausible

// thanx and credits to Andre
// https://github.com/andrerfcsantos/go-plausible
// wrapping https://plausible.io/docs
// ==============================================

import (
	"arc42-status/internal/types"
	"github.com/andrerfcsantos/go-plausible/plausible"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
	"sync"
)

// plausibleClient wraps the plausible API.
// The required (secret) API key is set within the initialization.
var plausibleClient *plausible.Client = nil

// need mutex to ensure isolated access to shared variable
var mutex sync.Mutex

var APIKEY string

// init is called during system startup
func init() {
	APIKEY = os.Getenv("PLAUSIBLE_API_KEY")

	if APIKEY == "" {
		// no value, no API calls, no results.
		// we exit here, as we have no chance of recovery
		log.Error().Msgf("CRITICAL ERROR: required plausible API key not set.\n" +
			"You need to set the 'PLAUSIBLE_API_KEY' environment variable prior to launching this application.\n")

		os.Exit(13)
	} else {
		log.Debug().Msg("PLAUSIBLE_API_KEY set")
	}
}

// initPlausibleHandler gets the plausible API key
// and creates a handler (NewClient) to perform queries upon
func initPlausibleHandler() *plausible.Client {

	if plausibleClient == nil {

		// APIKEY is a personal key for https://plausible.io
		// It needs to be set via environment variable

		return plausible.NewClient(APIKEY)
	} else {
		return plausibleClient
	}
}

// StatsForSite collects all relevant statistics for a given site
// (currently 7D, 30D and 12M)
func StatsForSite(thisSite string, stats *types.SiteStats) {

	// init the required handler
	// the function ensures it's initialized only once.
	plausibleClient = initPlausibleHandler()

	// Get a handler to perform queries for a given site
	siteHandler := plausibleClient.Site(thisSite)

	// WaitGroup to handle concurrent Goroutines
	var wg sync.WaitGroup

	// Get the three different metrics:

	var stats7D types.VisitorsAndPageViews
	wg.Add(1)
	SiteMetricsConcurrent(siteHandler, plausible.Last7Days(), &stats7D, &wg)

	var stats30D types.VisitorsAndPageViews
	wg.Add(1)
	SiteMetricsConcurrent(siteHandler, plausible.Last30Days(), &stats30D, &wg)

	var stats12M types.VisitorsAndPageViews
	wg.Add(1)
	SiteMetricsConcurrent(siteHandler, plausible.Last12Months(), &stats12M, &wg)

	wg.Wait()

	// now process results
	stats.Visitors7d = stats7D.Visitors
	stats.Visitors7dNr = stats7D.VisitorNr
	stats.Pageviews7d = stats7D.Pageviews
	stats.Pageviews7dNr = stats7D.PageviewNr

	stats.Visitors30d = stats30D.Visitors
	stats.Visitors30dNr = stats30D.VisitorNr
	stats.Pageviews30d = stats30D.Pageviews
	stats.Pageviews30dNr = stats30D.PageviewNr

	stats.Visitors12m = stats12M.Visitors
	stats.Pageviews12m = stats12M.Pageviews
	stats.Visitors12mNr = stats12M.VisitorNr
	stats.Pageviews12mNr = stats12M.PageviewNr
}

// SiteMetricsConcurrent collects statics for given site and period from plausible.io API.
// Return either the numbers or "n/a" in case of API errors
func SiteMetricsConcurrent(siteHandler *plausible.Site, period plausible.TimePeriod, vApvs *types.VisitorsAndPageViews, wg *sync.WaitGroup) {

	defer wg.Done()

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
		vApvs.PageviewNr = 0
		vApvs.Visitors = "n/a"
		vApvs.VisitorNr = 0
	} else {
		log.Debug().Msgf("%s had %d visitors for period %s", siteHandler.ID(), result.Visitors, period.Period)
		vApvs.Pageviews = strconv.Itoa(result.Pageviews)
		vApvs.PageviewNr = result.Pageviews
		vApvs.Visitors = strconv.Itoa(result.Visitors)
		vApvs.VisitorNr = result.Visitors
	}

}
