// github wraps the GitHub GraphQL API
package github

import (
	"github.com/shurcooL/githubv4"
	"site-usage-statistics/internal/types"
)

const GITHUB_GRAPHQL_API_KEY = "GITHUB_PAT"

// Define a query struct
type IssuesQuery struct {
	Repository struct {
		Issues struct {
			TotalCount githubv4.Int
		} `graphql:"issues(states:OPEN)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func BugCountForSite(thisSite string, stats *types.SiteStats) {

}
