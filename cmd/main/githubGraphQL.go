package main

import (
	"context"
	"fmt"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"log"
	"os"
	"site-usage-statistics/internal/github"
)

func main() {
	// Set your GitHub API token here
	apiToken := os.Getenv(github.GITHUB_GRAPHQL_API_KEY)

	fmt.Printf("GitHub API token: %s\n", apiToken)

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	// Initialize GitHub GraphQL client
	client := githubv4.NewClient(httpClient)

	// Fill in these variables with the appropriate values
	variables := map[string]interface{}{
		"owner": githubv4.String("arc42"),              // Replace with the repository owner's name
		"repo":  githubv4.String("faq.arc42.org-site"), // Replace with the repository name
	}

	// Declare an instance of the query struct
	var query github.IssuesQuery

	// Perform the query
	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		log.Fatal(err)
	}

	// Output the result
	fmt.Printf("Number of open issues: %d\n", query.Repository.Issues.TotalCount)
}
