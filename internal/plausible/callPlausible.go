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
// (currently 7D, 30D and 12M)
func StatsForSite(thisSite string, stats *types.SiteStats) {

	// Get a handler to perform queries for a given site
	siteHandler := plausibleClient.Site(thisSite)

	// Get the different metrics
	var stats7D types.VisitorsAndViews = SiteMetrics(siteHandler, plausible.Last7Days())
	stats.Visitors7d = stats7D.Visitors
	stats.Pageviews7d = stats7D.Pageviews

	var stats30D = SiteMetrics(siteHandler, plausible.Last30Days())
	stats.Visitors30d = stats30D.Visitors
	stats.Pageviews30d = stats30D.Pageviews

	var stats12M = SiteMetrics(siteHandler, plausible.Last12Months())
	stats.Visitors12m = stats12M.Visitors
	stats.Pageviews12m = stats12M.Pageviews

}

// SiteMetrics collects statics for given site and period from plausible.io API.
// return either the numbers or "n/a" in case of API errors
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
		vAv.Pageviews = "n/a"
		vAv.Visitors = "n/a"
	} else {
		vAv.Pageviews = strconv.Itoa(result.Pageviews)
		vAv.Visitors = strconv.Itoa(result.Visitors)
	}

	return vAv
}
