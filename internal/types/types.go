package types

import (
	"time"
)

var Arc42sites = [7]string{
	"arc42.org",
	"arc42.de",
	"docs.arc42.org",
	"faq.arc42.org",
	"canvas.arc42.org",
	"quality.arc42.org",
	"status.arc42.org",
}

// SiteStats contains viewer and issue statistics for a single arc42 site or subdomain.
// To facilitate template processing, site and GitHub repository name plus some other
// values are contained here too.
type SiteStats struct {
	Site           string // site name
	Repo           string // the URL of the GitHub repository
	NrOfOpenBugs   int    // the number of open bugs in that repo
	NrOfOpenIssues int    // number of open issues
	IssueBadgeURL  string // URL of the shields.io issues badge
	BugBadgeURL    string // URL of the shields.io bugs issue
	Visitors7d     string
	Pageviews7d    string
	Visitors30d    string
	Pageviews30d   string
	Visitors12m    string
	Pageviews12m   string

	//SumOfAllCounters TotalsForAllSites
}

// TotalsForAllSites contains the sum of all the distinct statistics,
// currently for 7d, 30d and 12m.
// If certain values are "n/a" (when the external API sends errors),
// we let these values count 0.
type TotalsForAllSites struct {
	SumOfVisitors7d   int
	SumOfPageviews7d  int
	SumOfVisitors30d  int
	SumOfPageviews30d int
	SumOfVisitors12m  int
	SumOfPageviews12m int
}

// Arc42Statistics collects information about the sites and subdomains
type Arc42Statistics struct {
	AppVersion string

	// LastUpdated contains the time.Time when the stats have
	// been updated. Can help to avoid flooding plausible.io with requests.
	LastUpdated       time.Time
	LastUpdatedString string // as we cannot directly use Golang functions from templates

	// HowLongDidItTake stores the time it took to collect
	// this data (from both plausible and GitHub)
	HowLongDidItTake string

	// FlyRegion stores the fly.io region code
	FlyRegion string
	// WhereDoesItRun contains the name of the location corresponding to FlyRegion
	WhereDoesItRun string

	// Stats4Site contains the statistics per site or subdomain
	Stats4Site [len(Arc42sites)]SiteStats

	// Totals contains the sum of all the statistics over all sites
	Totals TotalsForAllSites
}

// VisitorsAndPageViews is a temporarily-used struct.
// Note the 'string' type: most often it will be a number,
// but in case of errors it should be NotAvailable
type VisitorsAndPageViews struct {
	Visitors   string
	VisitorNr  int
	Pageviews  string
	PageViewNr int
}
