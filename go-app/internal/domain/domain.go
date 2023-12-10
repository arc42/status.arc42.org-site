package domain

import (
	"arc42-status/internal/badge"
	"arc42-status/internal/github"
	"arc42-status/internal/plausible"
	"arc42-status/internal/types"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

var AppVersion string

// ArcStats collects all data
var ArcStats types.Arc42Statistics

func SetAppVersion(appVersion string) {
	AppVersion = appVersion
	log.Debug().Msg("App version set to " + appVersion)
}

func setServerMetaInfo(a42s *types.Arc42Statistics) {
	a42s.AppVersion = AppVersion

	location, _ := time.LoadLocation("Europe/Berlin")

	// Get the current time in Bielefeld, the town that presumably does not exist
	bielefeldTime := time.Now().In(location)

	a42s.LastUpdated = bielefeldTime
	a42s.LastUpdatedString = bielefeldTime.Format("2. January 2006, 15:04:03h")
}

// LoadStats4AllSites retrieves the statistics for all sites from plausible.io and GitHub repositories.
func LoadStats4AllSites() types.Arc42Statistics {

	// the WaitGroup synchronises the parallel goroutines
	var wg sync.WaitGroup

	var a42s = types.Arc42Statistics{}

	var Stats4Sites = make([]types.SiteStats, len(types.Arc42sites))
	var Stats4Repos = make([]types.RepoStats, len(types.Arc42sites))

	// 1.) set meta info
	setServerMetaInfo(&a42s)

	// retrieve usage statistics (visitors and pageviews)
	for index, site := range types.Arc42sites {
		wg.Add(1)

		go getUsageStatisticsForSite(site, &Stats4Sites[index], &wg)
	}

	// retrieve repo statistics
	// currently:  number of open bugs and issues from GitHub
	for index, site := range types.Arc42sites {
		wg.Add(1)

		go getRepoStatisticsForSite(site+"-site", &Stats4Repos[index], &wg)
	}

	wg.Wait()

	// get results from Goroutines
	log.Debug().Msgf("transferring results in LoadStats4Site")
	for index := range types.Arc42sites {
		a42s.Stats4Site[index] = Stats4Sites[index]
		a42s.Stats4Site[index].NrOfOpenIssues = Stats4Repos[index].NrOfOpenIssues
		a42s.Stats4Site[index].NrOfOpenBugs = Stats4Repos[index].NrOfOpenBugs
		a42s.Stats4Site[index].Repo = Stats4Repos[index].Repo
	}

	// now calculate totals
	a42s.Totals = calculateTotals(Stats4Sites)

	// create Issue- and Bug Links
	for index, _ := range types.Arc42sites {
		badge.SetIssuesAndBugBadgeURLsForSite(&a42s.Stats4Site[index])
	}

	return a42s
}

func calculateTotals(stats []types.SiteStats) types.TotalsForAllSites {
	var totals types.TotalsForAllSites

	for index := range types.Arc42sites {
		totals.SumOfVisitors7d += stats[index].Visitors7dNr
		totals.SumOfPageviews7d += stats[index].Pageviews7dNr
		totals.SumOfVisitors30d += stats[index].Visitors30dNr
		totals.SumOfPageviews30d += stats[index].Pageviews30dNr
		totals.SumOfVisitors12m += stats[index].Visitors12mNr
		totals.SumOfPageviews12m += stats[index].Pageviews12mNr
	}
	log.Debug().Msgf("Total visits and pageviews (V/PV, 7d, 30d, 12m)= %d/%d, %d/%d, %d/%d", totals.SumOfVisitors7d, totals.SumOfPageviews7d, totals.SumOfVisitors30d, totals.SumOfPageviews30d, totals.SumOfVisitors12m, totals.SumOfPageviews12m)

	return totals
}

// getUsageStatisticsForSite retrieves the statistics for a single site from plausible.io.
// This func is called as Goroutine.
func getUsageStatisticsForSite(site string, thisSiteStats *types.SiteStats, wg *sync.WaitGroup) {
	defer wg.Done()

	// to avoid repeating the expression, introduce local var
	thisSiteStats.Site = site

	// get statistic data from plausible.io
	plausible.StatsForSite(site, thisSiteStats)

}

func getRepoStatisticsForSite(site string, thisRepoStats *types.RepoStats, wg *sync.WaitGroup) {
	defer wg.Done()

	thisRepoStats.Site = site
	thisRepoStats.Repo = github.GithubArc42URL + site

	github.StatsForRepo(site, thisRepoStats)

}
