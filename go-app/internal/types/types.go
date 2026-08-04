package types

import (
	"time"
)

var Arc42sites = [10]string{
	"arc42.org",
	"arc42.de",
	"docs.arc42.org",
	"faq.arc42.org",
	"canvas.arc42.org",
	"quality.arc42.org",
	"status.arc42.org",
	"pdfminion.arc42.org",
	"trainings.arc42.org",
	"meta.arc42.org",
}

// NotAvailable is what a metric reads when it could not be measured at all -
// either the external API refused to answer, or the site is not measured in
// the first place. It is deliberately not "0": a site nobody visited and a
// site nobody counts are different facts.
const NotAvailable = "n/a"

// RepoItem is one open issue or pull request as a dashboard tile lists it.
// Unlabelled marks an item no maintainer has classified, however old it is.
type RepoItem struct {
	Title      string
	URL        string
	AgeString  string // human-readable, e.g. "3 days", "5 weeks"
	IsPR       bool
	Unlabelled bool
}

// ClosedItem is a recently closed (or merged) issue or pull request. It is the
// counterweight to the open list: a tile showing only what is open reads like a
// site where nothing ever happens.
type ClosedItem struct {
	Title     string
	URL       string
	IsPR      bool
	ClosedAgo string // human-readable phrase, e.g. "3 days ago", "today"
}

// TilesData is what the dashboard template renders: the sites already in the
// order the grid reads them (see domain.TilesInAttentionOrder).
type TilesData struct {
	Tiles             []SiteStatsType
	LastUpdatedString string
}

// SiteStatsType contains visitor and pageviews statistics for a single arc42 site or subdomain.
type SiteStatsType struct {
	Site string // site name

	// HasTraffic tells "measured zero" apart from "not measured at all".
	// meta.arc42.org has no Plausible site, so its numbers are NotAvailable
	// rather than 0, and templates must not present them as a reading.
	HasTraffic bool

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

	// dashboard tile data
	IsHub          bool         // arc42.org and arc42.de are the hubs; the rest are satellites
	OpenItems      []RepoItem   // newest open issues and PRs, capped at github.MaxOpenShown
	RecentlyClosed []ClosedItem // most recently closed, capped at github.MaxClosedShown
	NrUntriaged    int          // how many open items nobody has classified
}

// MoreOpen is how many open issues and PRs exist beyond the ones the tile
// lists. Templates cannot do arithmetic, so it is computed here.
func (s SiteStatsType) MoreOpen() int {
	if n := s.NrOfOpenIssues + s.NrOfOpenPRs - len(s.OpenItems); n > 0 {
		return n
	}
	return 0
}

// RepoStatsType contains information about the repository underlying the site
type RepoStatsType struct {
	Site           string // site name
	Repo           string // the URL of the GitHub repository
	NrOfOpenBugs   int    // the number of open bugs in that repo
	NrOfOpenIssues int    // number of open issues
	NrOfPRs        int    // number of open pull-requests

	OpenItems      []RepoItem   // the newest open issues and PRs, for the tile list
	RecentlyClosed []ClosedItem // the most recently closed issues and PRs
	NrUntriaged    int          // how many open items nobody has classified
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
	// been updated.
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
	Stats4Site [len(Arc42sites)]SiteStatsType

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
