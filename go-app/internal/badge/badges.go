package badge

// A badge is a graphical representation, e.g. for the current issue- or bug count of a GitHub repository.

// to improve energy- and carbon efficiency, we try to use LOCAL images to the largest possible extend:
// We therefore have pre-generated such images for issue- and bug-counts from 0 to 20.
// Only if the bug- or issue count exceeds this number we will use the fallback of remote badges from shields.io, in other cases take the local equivalents.

import (
	"arc42-status/internal/types"
	"github.com/rs/zerolog/log"
	"strconv"
)

// after ADR-0006, we don't need the following URLs any more:
// const ShieldsGithubIssuesURL = "https://img.shields.io/github/issues-raw/arc42/"
// const ShieldsGithubBugsURLPrefix = "https://img.shields.io/github/issues-search/arc42/"
// const ShieldsBugSuffix = "?query=label%3Abug%20is%3Aopen&label=bugs&color=red"

const IssueName = "issue"
const BugName = "bug"

const IssueBadgeFileNameSuffix = "-issues.svg"
const BugBadgeFileNameSuffix = "-bugs.svg"

// LocalBadgeLocation is the constant for the file path of local badge images.
// we use these local versions to save remote-requests to shields.io
const LocalBadgeLocation = "/images/badges/"

const LocalBadgeSvgThreshold = 20

// SVGFileNameForKindOf creates the filename for the downloaded issue-svg files.
// These are required both for the downloading process AND for creating the URLs in the final output HTML
func SVGFileNameForKindOf(kindOf string, count int) string {
	switch kindOf {
	case IssueName:
		return strconv.Itoa(count) + IssueBadgeFileNameSuffix
	case BugName:
		return strconv.Itoa(count) + BugBadgeFileNameSuffix
	default:
		log.Error().Msgf("error creating filename for count %d and kindOf %s", count, kindOf)
		return "_error-" + strconv.Itoa(count)
	}
}

// badgeURL returns a badge URL, which always refers to a local image
// as these have been pre-generated (see ADR-0006)
// if the nrOfIssuesOrBugs is >= 0, create a link to a badge, otherwise NO bug badge shall be shown.
func badgeURL(kindOf string, nrOfIssuesOrBugs int) string {

	// corner case: zero bugs -> no badge shall be shown
	if nrOfIssuesOrBugs == 0 && kindOf == BugName {
		return ""
	}

	// corner case: more issues/bugs than LocalBadgeSvgThreshold
	badgeValue := nrOfIssuesOrBugs
	if nrOfIssuesOrBugs > LocalBadgeSvgThreshold {
		badgeValue = LocalBadgeSvgThreshold + 1
	}

	return LocalBadgeLocation + SVGFileNameForKindOf(kindOf, badgeValue)

}

// SetIssuesAndBugBadgeURLsForSite sets links for use within the templates
// (to avoid overly long string constants within these templates)
//
// if the number of bugs==0, then this URL remains empty, so no badge will be shown
func SetIssuesAndBugBadgeURLsForSite(stats *types.SiteStats) {

	stats.IssueBadgeURL = badgeURL(IssueName, stats.NrOfOpenIssues)
	stats.BugBadgeURL = badgeURL(BugName, stats.NrOfOpenBugs)
}
