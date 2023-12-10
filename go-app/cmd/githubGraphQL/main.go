package main

import (
	"site-usage-statistics/internal/github"
	"site-usage-statistics/internal/types"
)

func main() {

	var stats4Repos = make([]types.RepoStats, len(types.Arc42sites))

	github.StatsForRepo("faq.arc42.org-site", &stats4Repos[0])
	github.StatsForRepo("arc42.org-site", &stats4Repos[1])
}
