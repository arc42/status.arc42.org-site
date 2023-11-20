// github wraps the GitHub GraphQL API
package github

import (
	"fmt"
	"github.com/shurcooL/githubv4"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"log"
	"os"
)

const GITHUB_GRAPHQL_API_KEY = "GITHUB_API_KEY"

// Define a query struct
type IssuesQuery struct {
	Repository struct {
		Issues struct {
			TotalCount githubv4.Int
		} `graphql:"issues(states:OPEN)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func BugCountForSite(thisSite string) {
	// Set your GitHub API token here
	apiToken := os.Getenv(GITHUB_GRAPHQL_API_KEY)

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	// Initialize GitHub GraphQL client
	client := githubv4.NewClient(httpClient)

	// Fill in these variables with the appropriate values
	variables := map[string]interface{}{
		"owner": githubv4.String("arc42"),  // Replace with the repository owner's name
		"repo":  githubv4.String(thisSite), // Replace with the repository name
	}

	// Declare an instance of the query struct
	var query IssuesQuery

	// Perform the query
	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		log.Fatal(err)
	}

	// Output the result
	fmt.Printf("Number of open issues: %d\n", query.Repository.Issues.TotalCount)

}
