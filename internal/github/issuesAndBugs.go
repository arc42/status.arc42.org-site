// github wraps the GitHub GraphQL API
package github

import (
	"github.com/rs/zerolog/log"
	"github.com/shurcooL/githubv4"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"os"
)

const GithubArc42URL = "https://github.com/arc42/"

const GITHUB_GRAPHQL_API_KEY_NAME = "GITHUB_API_KEY"

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
		} `graphql:"bugs: issues(states:OPEN, labels:[\"BUG\"])"`
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

func IssuesAndBugsCountForSite(thisSite string) (nrOfIssues int, nrOfBugs int) {

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

	nrOfBugs = int(query.Repository.Bugs.TotalCount)
	nrOfIssues = int(query.Repository.Issues.TotalCount)

	log.Debug().Msgf("Number of open issues on %s: %d\n", thisSite, nrOfIssues)
	log.Debug().Msgf("Number of open bugs on %s: %d\n", thisSite, nrOfBugs)

	// this kind of return takes the named result parameters and returns those...
	return
}
