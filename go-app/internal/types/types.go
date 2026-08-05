package types

import (
	"time"
)

// Property is one thing this dashboard watches. Usually that is a site with a
// repository behind it; sometimes - as with the arc42 template - it is a
// repository with no site of its own.
//
// This list is the single declaration of what the family consists of and of how
// each member is reported. Every earlier version of this file spread the same
// facts over three places: a []string of hosts, a map of sites without
// Plausible, and the templates' own assumptions. Facts written down twice are
// facts that can disagree.
type Property struct {
	// Key identifies the property everywhere: it is the tile's data-site
	// value, the /site/<key>/ subpage segment, and the siteDetail query
	// parameter. For sites it is the host; for repo-only properties it is
	// the repository name.
	Key string

	// Host is the site's hostname, empty for a repo-only property. Empty Host
	// is what "this has no website" is written as; templates ask for it
	// rather than guessing from the key.
	Host string

	// Repo is the repository name under github.com/arc42/.
	Repo string

	// HasTraffic says a Plausible site exists and is worth asking about.
	// False means the property is not measured at all - which is a different
	// fact from "measured, and the number is zero", and must never be
	// rendered as 0.
	HasTraffic bool

	// InTable says the property gets a row in the traffic table. A property
	// can be measured and still stay out of the table: the table exists to
	// be read down a column, and rows that are structurally incomparable
	// (a CLI's landing page, a course-date feed) make that reading worse
	// rather than more complete. Their numbers live on their subpage.
	InTable bool

	// IsHub marks the two hub twins. BRAND.md (ADR-0001) defines arc42.org
	// and arc42.de as sharing the brand navy; every other property owns one
	// signature hue.
	IsHub bool
}

// Arc42properties is the family, in the order the collector walks it.
var Arc42properties = [11]Property{
	{Key: "arc42.org", Host: "arc42.org", Repo: "arc42.org-site", HasTraffic: true, InTable: true, IsHub: true},
	{Key: "arc42.de", Host: "arc42.de", Repo: "arc42.de-site", HasTraffic: true, InTable: true, IsHub: true},
	{Key: "docs.arc42.org", Host: "docs.arc42.org", Repo: "docs.arc42.org-site", HasTraffic: true, InTable: true},
	{Key: "faq.arc42.org", Host: "faq.arc42.org", Repo: "faq.arc42.org-site", HasTraffic: true, InTable: true},
	{Key: "canvas.arc42.org", Host: "canvas.arc42.org", Repo: "canvas.arc42.org-site", HasTraffic: true, InTable: true},
	{Key: "quality.arc42.org", Host: "quality.arc42.org", Repo: "quality.arc42.org-site", HasTraffic: true, InTable: true},
	{Key: "status.arc42.org", Host: "status.arc42.org", Repo: "status.arc42.org-site", HasTraffic: true, InTable: true},

	// Measured, but deliberately out of the traffic table (owner decision,
	// 2026-08-05). A tool's landing page and a course-date feed do not
	// compare with the documentation sites in the same column; their numbers
	// are reported on their own subpages instead.
	{Key: "pdfminion.arc42.org", Host: "pdfminion.arc42.org", Repo: "PDFminion", HasTraffic: true},
	{Key: "trainings.arc42.org", Host: "trainings.arc42.org", Repo: "trainings.arc42.org-site", HasTraffic: true},

	// No Plausible site at all: the brand and decision home is read by
	// maintainers, not by an audience, so it was never registered. Asking
	// Plausible about it would produce one API error per collection run that
	// says nothing, because nothing is broken.
	{Key: "meta.arc42.org", Host: "meta.arc42.org", Repo: "meta.arc42.org"},

	// The template itself: a repository with no site of its own. It is the
	// artefact the whole family exists to distribute, so it belongs on the
	// dashboard even though it has no host and no traffic to report.
	{Key: "arc42-template", Repo: "arc42-template"},
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

// SiteDetailData is what one per-site subpage fragment renders: everything
// known about a single property, uncapped - the lists the tile had to cut short.
type SiteDetailData struct {
	Site              SiteStatsType
	LastUpdatedString string
}

// SiteStatsType contains visitor and pageviews statistics for a single arc42 site or subdomain.
type SiteStatsType struct {
	Site string // the property's Key: a hostname, or a repository name

	// Host is the site's hostname, empty for a repo-only property. Templates
	// test it to decide whether "visit the site" is a link that can exist.
	Host string

	// HasTraffic tells "measured zero" apart from "not measured at all".
	// meta.arc42.org has no Plausible site, so its numbers are NotAvailable
	// rather than 0, and templates must not present them as a reading.
	HasTraffic bool

	// InTable says this property gets a row in the traffic table.
	InTable bool

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
	OpenItems      []RepoItem   // every open issue and PR the collector saw, newest first
	RecentlyClosed []ClosedItem // most recently closed, capped at github.MaxClosedStored
	NrUntriaged    int          // how many open items nobody has classified
}

// TileOpenShown and TileClosedShown are how much of each list a dashboard tile
// carries. The full lists are on the property's subpage; a tile that reprints
// them stops being a dashboard and becomes ten issue trackers side by side.
// They live here rather than in the github package because they are a property
// of the tile, not of the collection.
const (
	TileOpenShown   = 3
	TileClosedShown = 2
)

// TopOpen is the slice of open items a tile shows.
func (s SiteStatsType) TopOpen() []RepoItem {
	if len(s.OpenItems) > TileOpenShown {
		return s.OpenItems[:TileOpenShown]
	}
	return s.OpenItems
}

// TopClosed is the slice of recently closed items a tile shows.
func (s SiteStatsType) TopClosed() []ClosedItem {
	if len(s.RecentlyClosed) > TileClosedShown {
		return s.RecentlyClosed[:TileClosedShown]
	}
	return s.RecentlyClosed
}

// MoreOpen is how many open issues and PRs exist beyond the ones the tile
// lists. Templates cannot do arithmetic, so it is computed here.
//
// It counts from the reported totals, not from the collected list: the totals
// come from GitHub's own TotalCount and are right even when the item query
// failed or hit its page ceiling.
func (s SiteStatsType) MoreOpen() int {
	if n := s.NrOfOpenIssues + s.NrOfOpenPRs - len(s.TopOpen()); n > 0 {
		return n
	}
	return 0
}

// OpenIssues and OpenPRs split the open list the way the subpage lists it:
// issues and pull requests under headings of their own. The tile shows them
// merged and newest-first, because there it is one short "what is new" list;
// a full page needs the split to stay readable.
func (s SiteStatsType) OpenIssues() []RepoItem {
	return filterItems(s.OpenItems, false)
}

func (s SiteStatsType) OpenPRs() []RepoItem {
	return filterItems(s.OpenItems, true)
}

func filterItems(items []RepoItem, wantPR bool) []RepoItem {
	out := make([]RepoItem, 0, len(items))
	for _, item := range items {
		if item.IsPR == wantPR {
			out = append(out, item)
		}
	}
	return out
}

// HiddenOpen is how many open items exist beyond the ones even the subpage
// lists - the collector's page ceiling made visible rather than silently
// swallowed. Normally zero for the arc42 repositories.
func (s SiteStatsType) HiddenOpen() int {
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
	Stats4Site [len(Arc42properties)]SiteStatsType

	// Totals: sum of all the statistics over all sites
	Totals TotalsForAllSites
}

// TableRows are the properties the traffic table shows - and, exactly, the ones
// the totals row sums. A totals row that counts rows the reader cannot see is a
// number nobody can check.
func (a Arc42Statistics) TableRows() []SiteStatsType {
	rows := make([]SiteStatsType, 0, len(a.Stats4Site))
	for _, s := range a.Stats4Site {
		if s.InTable {
			rows = append(rows, s)
		}
	}
	return rows
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
