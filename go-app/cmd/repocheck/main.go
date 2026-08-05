// Command repocheck asks the live GitHub GraphQL API the same questions the
// service asks, and prints the answers. It exists because the interesting
// failure mode of these queries is not an error but silence: the `search`
// connection returns an empty node set for fine-grained tokens without
// complaining, so "no results" has to be looked at, not assumed.
//
// Read-only. With no arguments it walks every arc42 site through
// domain.RepoNameForSite, which also proves the site-to-repository mapping
// against reality. With arguments it takes them as repository names:
//
//	source ./set-api-keys.sh >/dev/null && LOGLEVEL=ERROR go run ./cmd/repocheck status.arc42.org-site canvas.arc42.org-site
//
// Scratch tool: not part of the deployed service.
package main

import (
	"arc42-status/internal/github"
	"arc42-status/internal/types"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func main() {
	// this tool speaks on stdout; the log is only there to show API failures
	level, err := zerolog.ParseLevel(strings.ToLower(os.Getenv("LOGLEVEL")))
	if err != nil || os.Getenv("LOGLEVEL") == "" {
		level = zerolog.ErrorLevel
	}
	zerolog.SetGlobalLevel(level)

	repos := os.Args[1:]
	if len(repos) == 0 {
		for _, property := range types.Arc42properties {
			repos = append(repos, property.Repo)
		}
		fmt.Printf("no repositories given, checking all %d arc42 properties\n", len(types.Arc42properties))
	}

	for _, repo := range repos {
		var stats types.RepoStatsType
		github.StatsForRepo(repo, &stats)
		github.OpenItemsForRepo(repo, &stats)
		github.RecentlyClosedForRepo(repo, &stats)

		fmt.Printf("\n=== %s\n", repo)
		fmt.Printf("    counts: %d open issues, %d bugs, %d open PRs, %d untriaged\n",
			stats.NrOfOpenIssues, stats.NrOfOpenBugs, stats.NrOfPRs, stats.NrUntriaged)

		fmt.Printf("    open items collected: %d (a tile shows %d)\n", len(stats.OpenItems), types.TileOpenShown)
		for _, item := range stats.OpenItems {
			fmt.Printf("      %-5s %-9s %-11s %s\n", kind(item.IsPR), item.AgeString, unlabelled(item.Unlabelled), truncate(item.Title))
		}

		fmt.Printf("    recently closed collected: %d (cap %d, a tile shows %d)\n", len(stats.RecentlyClosed), github.MaxClosedStored, types.TileClosedShown)
		for _, item := range stats.RecentlyClosed {
			fmt.Printf("      %-5s closed %-14s %s\n", kind(item.IsPR), item.ClosedAgo, truncate(item.Title))
		}
	}
}

func kind(isPR bool) string {
	if isPR {
		return "PR"
	}
	return "issue"
}

func unlabelled(u bool) string {
	if u {
		return "unlabelled"
	}
	return ""
}

func truncate(title string) string {
	const max = 70
	runes := []rune(title)
	if len(runes) <= max {
		return title
	}
	return string(runes[:max-1]) + "…"
}
