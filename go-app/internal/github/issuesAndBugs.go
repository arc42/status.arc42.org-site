// github wraps the GitHub GraphQL API
package github

import (
	"arc42-status/internal/types"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/shurcooL/githubv4"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"os"
	"sort"
	"time"
)

const GithubArc42URL = "https://github.com/arc42/"

const GITHUB_GRAPHQL_API_KEY_NAME = "GITHUB_API_KEY"

// GitHubQueryInterval determines how many minutes to minimally wait prior to calling the external API again
// currently set to 1 minute
const GitHubQueryInterval = time.Minute

// gitHubLastTimeCalled contains the time we called the public GitHub API the last time.
// Initially, it is set to Jan 1st 2004 - the approximate date arc42 was created.
//
// Written at the end of every query but not yet read by anything: the guard
// that compares it against GitHubQueryInterval is the subject of the open
// #68-rate-limiter branch. Kept rather than deleted so that branch still has
// its groundwork; the nolint goes when the guard lands and reads it.
//
//nolint:unused // groundwork for the #68 rate limiter, see above
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
		} `graphql:"bugs: issues(states:OPEN, labels:[\"bug\", \"bugs\", \"BUG\", \"BUGS\"])"`
		PullRequests struct {
			TotalCount githubv4.Int
		} `graphql:"pullRequests(states:OPEN)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// UntriagedWindow is how recently an issue or PR must have arrived to count as
// untriaged on age alone. Anything older still counts when it has no label:
// an unlabelled issue is one no maintainer has classified.
const UntriagedWindow = 30 * 24 * time.Hour

// MaxClosedStored caps how many recently closed items the collector keeps.
// Three is what a per-site subpage lists; the tile shows the first two of them
// (types.TileClosedShown). Closed work is context, not a call to action, and a
// longer list turns a status page into a changelog.
//
// There is no matching cap for open items: the collector keeps every one it
// sees, because the subpage lists them all. What a tile shows is decided by the
// tile (types.TileOpenShown), not by what was collected - the two used to be
// the same number, and that is why the subpages had nothing to add.
const MaxClosedStored = 3

// Each of the two closed connections still asks for 6, deliberately more than
// MaxClosedStored: the merge below re-sorts by the time an item actually
// closed, and asking each side for only three would let a repository with four
// recent merges push a more recently closed issue out of the result before the
// sort ever saw it.

// openNode is one open issue or pull request with just enough detail to list it
// and to decide whether anybody has triaged it.
type openNode struct {
	Title     githubv4.String
	URL       githubv4.URI
	CreatedAt githubv4.DateTime
	Labels    struct {
		TotalCount githubv4.Int
	} `graphql:"labels(first: 1)"`
}

// openItemsQuery walks the repository's own issue and pull-request connections.
//
// It deliberately does NOT use the GraphQL `search` connection, which would be
// the tidier way to get issues and PRs in one list: search returns an empty
// node set for fine-grained personal access tokens (`github_pat_…`) — with no
// error, just no results — and that is the kind of token this service uses.
// The repository connections work with both token types. The same holds for
// closedItemsQuery below.
type openItemsQuery struct {
	Repository struct {
		Issues struct {
			Nodes []openNode
		} `graphql:"issues(states: OPEN, first: 50, orderBy: {field: CREATED_AT, direction: DESC})"`
		PullRequests struct {
			Nodes []openNode
		} `graphql:"pullRequests(states: OPEN, first: 50, orderBy: {field: CREATED_AT, direction: DESC})"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// closedNode is one closed issue or pull request. ClosedAt is set for both
// closed and merged pull requests, which is why it - and not MergedAt - is what
// the list sorts on.
type closedNode struct {
	Title    githubv4.String
	URL      githubv4.URI
	ClosedAt githubv4.DateTime
}

// closedItemsQuery asks each connection for its most recently touched closed
// items. GitHub cannot order by CLOSED_AT, so it orders by UPDATED_AT and the
// merge below re-sorts by ClosedAt: a stale item that got a late comment would
// otherwise jump the queue. Pull requests need [CLOSED, MERGED] because in
// GitHub's vocabulary a merged PR is not a closed one, while to a maintainer
// reading the tile it plainly is.
type closedItemsQuery struct {
	Repository struct {
		Issues struct {
			Nodes []closedNode
		} `graphql:"issues(states: CLOSED, orderBy: {field: UPDATED_AT, direction: DESC}, first: 6)"`
		PullRequests struct {
			Nodes []closedNode
		} `graphql:"pullRequests(states: [CLOSED, MERGED], orderBy: {field: UPDATED_AT, direction: DESC}, first: 6)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// humanAge renders a duration the way a maintainer would say it out loud.
func humanAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < 24*time.Hour:
		return "today"
	case d < 48*time.Hour:
		return "1 day"
	case d < 14*24*time.Hour:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	case d < 60*24*time.Hour:
		return fmt.Sprintf("%d weeks", int(d.Hours()/24/7))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%d months", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%d years", int(d.Hours()/24/365))
	}
}

// humanAgo is humanAge as a phrase pointing backwards: "today", "1 day ago",
// "3 weeks ago". Kept separate because "today ago" is not English.
func humanAgo(t time.Time) string {
	age := humanAge(t)
	if age == "today" {
		return "today"
	}
	return age + " ago"
}

// OpenItemsForRepo lists every open issue and pull request the query returns,
// newest first across both, and counts in passing the ones nobody has
// classified: opened within UntriagedWindow, or carrying no label at all.
//
// One query serves both, because both need the same nodes: the list is what the
// tile shows, the count is what decides whether the tile calls for attention.
//
// Note the ceiling: each connection returns at most 50 open items, so a repo
// with more than 50 open issues (or 50 open PRs) could hide an old unlabelled
// one beyond that page. The arc42 repos are far below this today; the largest
// has 19 open issues.
func OpenItemsForRepo(repoName string, stats *types.RepoStatsType) {
	client := initGitHubGraphQLClient()
	if client == nil {
		log.Error().Msgf("GitHub client initialization failed for repo %s - API key not available", repoName)
		return
	}

	var query openItemsQuery
	variables := map[string]interface{}{
		"owner": githubv4.String("arc42"),
		"repo":  githubv4.String(repoName),
	}

	if err := client.Query(context.Background(), &query, variables); err != nil {
		log.Error().Msgf("GitHub open-items query failed for repo %s: %v", repoName, err)
		// leave OpenItems empty; the tile renders its "nothing open" state
		return
	}

	cutoff := time.Now().Add(-UntriagedWindow)

	// collect issues and PRs together, newest first across both
	type candidate struct {
		node openNode
		isPR bool
	}
	candidates := make([]candidate, 0,
		len(query.Repository.Issues.Nodes)+len(query.Repository.PullRequests.Nodes))
	for _, n := range query.Repository.Issues.Nodes {
		candidates = append(candidates, candidate{n, false})
	}
	for _, n := range query.Repository.PullRequests.Nodes {
		candidates = append(candidates, candidate{n, true})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].node.CreatedAt.Time.After(candidates[j].node.CreatedAt.Time)
	})

	for _, c := range candidates {
		unlabelled := int(c.node.Labels.TotalCount) == 0
		created := c.node.CreatedAt.Time

		untriaged := unlabelled || created.After(cutoff)
		if untriaged {
			stats.NrUntriaged++
		}

		stats.OpenItems = append(stats.OpenItems, types.RepoItem{
			Title:      string(c.node.Title),
			URL:        c.node.URL.String(),
			AgeString:  humanAge(created),
			IsPR:       c.isPR,
			Untriaged:  untriaged,
			Unlabelled: unlabelled,
		})
	}

	log.Debug().Msgf("%s: %d open items, %d untriaged", repoName, len(candidates), stats.NrUntriaged)
}

// RecentlyClosedForRepo lists the most recently closed issues and pull requests
// (capped at MaxClosedStored), merged from the two connections and re-sorted by
// the time they actually closed.
func RecentlyClosedForRepo(repoName string, stats *types.RepoStatsType) {
	client := initGitHubGraphQLClient()
	if client == nil {
		log.Error().Msgf("GitHub client initialization failed for repo %s - API key not available", repoName)
		return
	}

	var query closedItemsQuery
	variables := map[string]interface{}{
		"owner": githubv4.String("arc42"),
		"repo":  githubv4.String(repoName),
	}

	if err := client.Query(context.Background(), &query, variables); err != nil {
		log.Error().Msgf("GitHub closed-items query failed for repo %s: %v", repoName, err)
		// leave RecentlyClosed empty; the tile simply omits the section
		return
	}

	type candidate struct {
		node closedNode
		isPR bool
	}
	candidates := make([]candidate, 0,
		len(query.Repository.Issues.Nodes)+len(query.Repository.PullRequests.Nodes))
	for _, n := range query.Repository.Issues.Nodes {
		candidates = append(candidates, candidate{n, false})
	}
	for _, n := range query.Repository.PullRequests.Nodes {
		candidates = append(candidates, candidate{n, true})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].node.ClosedAt.Time.After(candidates[j].node.ClosedAt.Time)
	})

	for _, c := range candidates {
		if len(stats.RecentlyClosed) >= MaxClosedStored {
			break
		}
		stats.RecentlyClosed = append(stats.RecentlyClosed, types.ClosedItem{
			Title:     string(c.node.Title),
			URL:       c.node.URL.String(),
			IsPR:      c.isPR,
			ClosedAgo: humanAgo(c.node.ClosedAt.Time),
		})
	}

	log.Debug().Msgf("%s: %d recently closed items listed", repoName, len(stats.RecentlyClosed))
}

func initGitHubGraphQLClient() *githubv4.Client {
	// Set your GitHub API token here
	apiToken := os.Getenv(GITHUB_GRAPHQL_API_KEY_NAME)

	if apiToken == "" {
		log.Error().Msgf("GitHub API key not set. You need to set the '%s' environment variable to access GitHub repositories.", GITHUB_GRAPHQL_API_KEY_NAME)
		return nil
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	// Initialize GitHub GraphQL client
	return githubv4.NewClient(httpClient)

}

func StatsForRepo(thisSite string, stats *types.RepoStatsType) {

	// Initialize GitHub GraphQL client
	client := initGitHubGraphQLClient()
	if client == nil {
		log.Error().Msgf("GitHub client initialization failed for repo %s - API key not available", thisSite)
		return
	}

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
		log.Error().Msgf("GitHub API call failed for repo %s: %v", thisSite, err)
		// When API call fails, keep the existing values (which are 0 by default)
		// This prevents overwriting with invalid/empty data
		return
	}

	stats.NrOfOpenBugs = int(query.Repository.Bugs.TotalCount)
	stats.NrOfOpenIssues = int(query.Repository.Issues.TotalCount)
	stats.NrOfPRs = int(query.Repository.PullRequests.TotalCount)

	// reset timer
	gitHubLastTimeCalled = time.Now()

	log.Debug().Msgf("%s has %d open issues, %d bugs, and %d PRs", thisSite, stats.NrOfOpenIssues, stats.NrOfOpenBugs, stats.NrOfPRs)

}
