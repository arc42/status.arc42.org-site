package types

var Arc42sites = [7]string{
	"arc42.org",
	"arc42.de",
	"docs.arc42.org",
	"faq.arc42.org",
	"canvas.arc42.org",
	"quality.arc42.org",
	"status.arc42.org",
}

type SiteStats struct {
	Site          string // site name
	Repo          string // the URL of the Github repository
	IssueBadgeURL string // URL of the shields.io issues badge
	BugBadgeURL   string // URL of the shields.io bugs issue
	Visitors7d    string
	Pageviews7d   string
	Visitors30d   string
	Pageviews30d  string
	Visitors12m   string
	Pageviews12m  string
}

type Arc42Statistics struct {
	AppVersion string
	Timestamp  string
	Stats4Site [len(Arc42sites)]SiteStats
}

// VisitorsAndViews is a temporarily-used struct.
// Note the 'string' type: most often it will be a number,
// but in case of errors it should be NotAvailable
type VisitorsAndViews struct {
	Visitors  string
	Pageviews string
}
