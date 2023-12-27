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

// SiteStats contains visitor and pageviews statistics for a single arc42 site or subdomain.
type SiteStats struct {
	Site           string // site name
	Visitors7d     string
	Visitors7dNr   int
	PageViews7d    string
	PageViews7dNr  int
	Visitors30d    string
	Visitors30dNr  int
	PageViews30d   string
	PageViews30dNr int
	Visitors12m    string
	Visitors12mNr  int
	PageViews12m   string
	PageViews12mNr int

	// these are needed for the template to execute properly
	Repo           string // the URL of the GitHub repository
	NrOfOpenBugs   int    // the number of open bugs in that repo
	NrOfOpenIssues int    // number of open issues
	NrOfOpenPRs    int
}

// RepoStats contains information about the repository underlying the site
type RepoStats struct {
	Site           string // site name
	Repo           string // the URL of the GitHub repository
	NrOfOpenBugs   int    // the number of open bugs in that repo
	NrOfOpenIssues int    // number of open issues
	NrOfPRs        int    // number of open pull-requests

}

// TotalsForAllSites contains the sum of all the distinct statistics,
// currently for 7d, 30d and 12m.
// If certain values are "n/a" (when the external API sends errors),
// we let these values count 0.
type TotalsForAllSites struct {
	SumOfVisitors7dNr   int
	SumOfVisitors7d     string
	SumOfPageViews7dNr  int
	SumOfPageViews7d    string
	SumOfVisitors30dNr  int
	SumOfVisitors30d    string
	SumOfPageViews30dNr int
	SumOfPageViews30d   string
	SumOfVisitors12mNr  int
	SumOfVisitors12m    string
	SumOfPageViews12mNr int
	SumOfPageViews12m   string
	TotalNrOfIssues     int
	TotalNrOfBugs       int
	TotalNrOfPRs        int
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
	// it also contains Repo stats, like issues and bugs
	Stats4Site [len(Arc42sites)]SiteStats

	// Totals: sum of all the statistics over all sites
	Totals TotalsForAllSites
}

// VisitorsAndPageViews is a temporary struct.
// Note the 'string' type: most often it will be a number,
// but in case of errors it should be NotAvailable
type VisitorsAndPageViews struct {
	Visitors   string
	VisitorNr  int
	PageViews  string
	PageViewNr int
}

// IssuesAndBugs is a struct used during the (concurrent) calls to GitHub.
type IssuesAndBugs struct {
	NrOfIssues int
	NrOfBugs   int
}
