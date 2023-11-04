package github

import (
	"site-usage-statistics/internal/types"
)

func BugCountForSite(thisSite string, stats *types.SiteStats) {

}

// BugCount collects the number of issues labeled as bugs from github API.
// return either the numbers or "n/a" in case of API errors
