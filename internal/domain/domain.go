package domain

import (
	"github.com/rs/zerolog/log"
	"site-usage-statistics/internal/badge"
	"site-usage-statistics/internal/github"
	"site-usage-statistics/internal/plausible"
	"site-usage-statistics/internal/types"
	"sync"
	"time"
)

var AppVersion string

// ArcStats collects all data
var ArcStats types.Arc42Statistics

var mutex sync.Mutex

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

// LoadStats4AllSites calls the plausible wrapper package to retrieve site statistics.
func LoadStats4AllSites() types.Arc42Statistics {

	// the WaitGroup synchronises the parallel goroutines
	var wg sync.WaitGroup

	var a42s = types.Arc42Statistics{}

	var Stats4Sites = make([]types.SiteStats, len(types.Arc42sites))
	var Stats4Repos = make([]types.RepoStats, len(types.Arc42sites))

	// var IssuesAndBugs4Sites = make([]types.)
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
	}

	// now calculate totals
	a42s.Totals = calculateTotals(Stats4Sites)

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

	// TODO: mutex.Lock might break performance optimization - rethink!!
	mutex.Lock()
	defer mutex.Unlock()

	// to avoid repeating the expression, introduce local var
	thisSiteStats.Site = site

	// get statistic data from plausible.io
	plausible.StatsForSite(site, thisSiteStats)

}

func getRepoStatisticsForSite(site string, thisRepoStats *types.RepoStats, wg *sync.WaitGroup) {
	defer wg.Done()

	mutex.Lock()
	defer mutex.Unlock()

	thisRepoStats.Site = site
	thisRepoStats.Repo = github.GithubArc42URL + site + "-site"

	github.StatsForRepo(site, thisRepoStats)

}

// bugBadgeURL returns a shields.io bug badge URL,
// if the bug-count is >= 0. Otherwise, NO bug badge
// shall be shown.
func bugBadgeURL(site string, nrOfBugs int) string {

	// shields.io bug URLS look like that:https://img.shields.io/github/issues-search/arc42/quality.arc42.org-site?query=label%3Abug%20is%3Aopen&label=bugs&color=red

	if nrOfBugs > 0 {
		return badge.ShieldsGithubBugsURLPrefix + site + "-site" + badge.ShieldsBugSuffix
	} else {
		return ""
	}
}

// setIssuesAndBugBadgeURLsForSite sets some constants for use within the templates
// (to avoid overly long string constants within these templates)
//
// if the number of bugs==0, then this URL remains empty, so no badge will be shown
// if the number of issues==0, then a special "hurray" badge shall be shown.

func setIssuesAndBugBadgeURLsForSite(stats *types.RepoStats) {

	// shields.io issues URLS look like that: https://img.shields.io/github/issues-raw/arc42/arc42.org-site
	stats.IssueBadgeURL = badge.ShieldsGithubIssuesURL + stats.Site + "-site"

	stats.BugBadgeURL = bugBadgeURL(stats.Site, stats.NrOfOpenBugs)
}
