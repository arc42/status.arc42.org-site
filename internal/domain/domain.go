package domain

import (
	"fmt"
	badges "site-usage-statistics/badge"
	"site-usage-statistics/internal/github"
	"site-usage-statistics/internal/plausible"
	"site-usage-statistics/internal/types"
	"time"
)

var AppVersion string

var ArcStats types.Arc42Statistics
var SumOfStats types.SumOfAllSites

func SetAppVersion(appVersion string) {
	AppVersion = appVersion

}

// LoadStats4AllSites calls the plausible.io API to retrieve all statistics
// and sets several site constants (URLs)
func LoadStats4AllSites() types.Arc42Statistics {

	fmt.Printf("loading statistics...\n")

	location, _ := time.LoadLocation("Europe/Berlin")

	// Get the current time in Bielefeld, the town that presumably does not exist
	bielefeldTime := time.Now().In(location)

	a42s := types.Arc42Statistics{
		AppVersion:        AppVersion,
		LastUpdated:       bielefeldTime,
		LastUpdatedString: bielefeldTime.Format("2. January 2006, 15:04:03h"),
	}

	for index, site := range types.Arc42sites {
		a42s.Stats4Site[index].Site = site

		// query the number of open bugs from GitHub
		a42s.Stats4Site[index].NrOfOpenBugs = 1

		// TODO: let StatsForSite update the Stats4Site and the Sums struct
		// set the statistic data from plausible.io
		plausible.StatsForSite(site, &a42s.Stats4Site[index], &a42s.Totals)

		// set some URLs so the templates get smaller
		setURLsForSite(&a42s.Stats4Site[index])
	}
	return a42s
}

// setURLsForSite sets some constants for use within the templates
// (to avoid overly long string constants within these templates)
func setURLsForSite(stats *types.SiteStats) {

	// all arc42 website repos follow this naming convention, e.g. arc42.org-site
	stats.Repo = github.GithubArc42URL + stats.Site + "-site"

	// shields.io issues URLS look like that: https://img.shields.io/github/issues-raw/arc42/arc42.org-site
	stats.IssueBadgeURL = badges.ShieldsGithubIssuesURL + stats.Site + "-site"

	// shields.io bug URLS look like that:https://img.shields.io/github/issues-search/arc42/quality.arc42.org-site?query=label%3Abug%20is%3Aopen&label=bugs&color=red
	stats.BugBadgeURL = badges.ShieldsGithubBugsURLPrefix + stats.Site + "-site" + badges.ShieldsBugSuffix
}
