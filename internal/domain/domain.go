package domain

import (
	"site-usage-statistics/internal/badge"
	"site-usage-statistics/internal/github"
	"site-usage-statistics/internal/plausible"
	"site-usage-statistics/internal/types"
	"time"
)

var AppVersion string

var ArcStats types.Arc42Statistics
var SumOfStats types.TotalsForAllSites

func SetAppVersion(appVersion string) {
	AppVersion = appVersion

}

func setServerMetaInfo(a42s *types.Arc42Statistics) {
	a42s.AppVersion = AppVersion

	location, _ := time.LoadLocation("Europe/Berlin")

	// Get the current time in Bielefeld, the town that presumably does not exist
	bielefeldTime := time.Now().In(location)

	a42s.LastUpdated = bielefeldTime
	a42s.LastUpdatedString = bielefeldTime.Format("2. January 2006, 15:04:03h")
}

// LoadStats4AllSites calls the various wrapper APIs to retrieve all site and repo statistics.
// 1.) set some meta info: App-version plus current time
// 2.) for all sites:
// 2.1) get data from plausible.io
// 2.2) get issues and bug counts from GitHub.com
// 2.3) set the URLs for the issue and bug badges

func LoadStats4AllSites() types.Arc42Statistics {

	a42s := types.Arc42Statistics{}

	// 1.) set meta info
	setServerMetaInfo(&a42s)

	// 2.) for all sites...
	// this could be done in Goroutines to improve performance
	for index, site := range types.Arc42sites {

		// to avoid repeating the expression, introduce local var
		thisSite := &a42s.Stats4Site[index]

		// previously this was: a42s.Stats4Site[index].Site = site
		thisSite.Site = site
		// all arc42 website repos follow this naming convention, e.g. arc42.org-site
		thisSite.Repo = github.GithubArc42URL + site + "-site"

		// 2.1 set the statistic data from plausible.io
		plausible.StatsForSite(site, &a42s.Stats4Site[index], &a42s.Totals)

		// 2.2 query the number of open bugs and issues from GitHub
		//a42s.Stats4Site[index].NrOfOpenIssues, a42s.Stats4Site[index].NrOfOpenBugs = github.IssuesAndBugsCountForSite(site)
		thisSite.NrOfOpenIssues, thisSite.NrOfOpenBugs = github.IssuesAndBugsCountForSite(site + "-site")

		// set some URLs so the templates get smaller
		setIssuesAndBugBadgeURLsForSite(thisSite)

	}
	return a42s
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

func setIssuesAndBugBadgeURLsForSite(stats *types.SiteStats) {

	// shields.io issues URLS look like that: https://img.shields.io/github/issues-raw/arc42/arc42.org-site
	stats.IssueBadgeURL = badge.ShieldsGithubIssuesURL + stats.Site + "-site"

	stats.BugBadgeURL = bugBadgeURL(stats.Site, stats.NrOfOpenBugs)
}
