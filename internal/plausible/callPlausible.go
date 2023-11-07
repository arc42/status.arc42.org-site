package plausible

// thanx and credits to Andre
// https://github.com/andrerfcsantos/go-plausible
// wrapping https://plausible.io/docs
// ==============================================

import (
	"github.com/andrerfcsantos/go-plausible/plausible"
	"log"
	"site-usage-statistics/internal/types"
	"strconv"
)

// APIKEY is Gernot's personal key for https://plausible.io.
// one day I will find a better way to keep this key secret,
// but for now it's good enough to have in the private Github repo.
// TODO: handle secret in appropriate way
const APIKEY = "1-eu-hRPPmR6MJ28oGkc8cye3I5dgBUCE4jWvoSXtMj8zN2kmuwUaABcE2gO0MST"

var plausibleClient = plausible.NewClient(APIKEY)

// StatsForSite collects all relevant statistics for a given site
// (currently 7D, 30D and 12M), and updates the Sums accordingly
func StatsForSite(thisSite string, stats *types.SiteStats, sums *types.SumOfAllSites) {

	// Get a handler to perform queries for a given site
	siteHandler := plausibleClient.Site(thisSite)

	// Get the different metrics
	var stats7D types.VisitorsAndViews = SiteMetrics(siteHandler, plausible.Last7Days())
	stats.Visitors7d = stats7D.Visitors
	stats.Pageviews7d = stats7D.Pageviews
	sums.SumOfVisitors7d += stats7D.VisitorNr
	sums.SumOfPageviews7d += stats7D.PageViewNr

	var stats30D = SiteMetrics(siteHandler, plausible.Last30Days())
	stats.Visitors30d = stats30D.Visitors
	stats.Pageviews30d = stats30D.Pageviews
	sums.SumOfVisitors30d += stats30D.VisitorNr
	sums.SumOfPageviews30d += stats30D.PageViewNr

	var stats12M = SiteMetrics(siteHandler, plausible.Last12Months())
	stats.Visitors12m = stats12M.Visitors
	stats.Pageviews12m = stats12M.Pageviews
	sums.SumOfVisitors12m += stats12M.VisitorNr
	sums.SumOfPageviews12m += stats12M.PageViewNr

}

// SiteMetrics collects statics for given site and period from plausible.io API.
// Return either the numbers or "n/a" in case of API errors
// Updates the SumOfAllSites
func SiteMetrics(siteHandler *plausible.Site, period plausible.TimePeriod) types.VisitorsAndViews {

	var vAv types.VisitorsAndViews

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
		log.Println("Error performing query to plausible.io: %v", err)
		// in this case, we don't add anything to the Sums
		vAv.Pageviews = "n/a"
		vAv.PageViewNr = 0
		vAv.Visitors = "n/a"
		vAv.VisitorNr = 0
	} else {
		vAv.Pageviews = strconv.Itoa(result.Pageviews)
		vAv.PageViewNr = result.Pageviews
		vAv.Visitors = strconv.Itoa(result.Visitors)
		vAv.VisitorNr = result.Visitors
	}

	return vAv
}
