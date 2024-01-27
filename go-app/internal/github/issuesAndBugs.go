// github wraps the GitHub GraphQL API
package github

import (
	"arc42-status/internal/types"
	"github.com/rs/zerolog/log"
	"github.com/shurcooL/githubv4"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"os"
	"time"
)

const GithubArc42URL = "https://github.com/arc42/"

const GITHUB_GRAPHQL_API_KEY_NAME = "GITHUB_API_KEY"

// GitHubQueryInterval determines how many minutes to minimally wait prior to calling the external API again
// currently set to 1 minute
const GitHubQueryInterval = time.Minute

// gitHubLastTimeCalled contains the time we called the public GitHub API the last time.
// Initially, it is set to Jan 1st 2004 - the approximate date arc42 was created.
var gitHubLastTimeCalled = time.Date(2004, time.January, 1, 0, 0, 0, 0, time.UTC)

// Define the query structs,
// using JSON GraphQL "struct-tags":
// for an explanation, see here: https://www.digitalocean.com/community/tutorials/how-to-use-struct-tags-in-go

type BugsIssuesQuery struct {
	Repository struct {
		Issues struct {
			TotalCount githubv4.Int
		} `graphql:"issues(states:OPEN)"`
		Bugs struct {
			TotalCount githubv4.Int
		} `graphql:"bugs: issues(states:OPEN, labels:[\"BUG\", \"BUGS\"])"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func initGitHubGraphQLClient() *githubv4.Client {
	// Set your GitHub API token here
	apiToken := os.Getenv(GITHUB_GRAPHQL_API_KEY_NAME)

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	// Initialize GitHub GraphQL client
	return githubv4.NewClient(httpClient)

}

func StatsForRepo(thisSite string, stats *types.RepoStats) {

	// Initialize GitHub GraphQL client
	client := initGitHubGraphQLClient()

	// Declare an instance of the query struct
	var query BugsIssuesQuery

	// Fill in these variables with the appropriate values
	variables := map[string]interface{}{
		"owner": githubv4.String("arc42"),
		"repo":  githubv4.String(thisSite),
	}

	// Perform the query
	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		log.Error().Msgf(err.Error(), query)
	}

	stats.NrOfOpenBugs = int(query.Repository.Bugs.TotalCount)
	stats.NrOfOpenIssues = int(query.Repository.Issues.TotalCount)

	// reset timer
	gitHubLastTimeCalled = time.Now()

	log.Debug().Msgf("%s has %d open issues and %d bugs", thisSite, stats.NrOfOpenIssues, stats.NrOfOpenBugs)

}
