package types

type SiteStats struct {
	Site         string
	Visitors7d   string
	Pageviews7d  string
	Visitors30d  string
	Pageviews30d string
	Visitors12m  string
	Pageviews12m string
}

// note the 'string' type: most often it will be a number,
// but in case of errors it should be NotAvailable
type VisitorsAndViews struct {
	Visitors  string
	Pageviews string
}
