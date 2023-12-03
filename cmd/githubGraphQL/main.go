package main

import "site-usage-statistics/internal/github"

func main() {

	github.StatsForRepo("faq.arc42.org-site")
	github.StatsForRepo("arc42.org-site")
}
