package main

import "site-usage-statistics/internal/github"

func main() {

	github.IssuesAndBugsCountForSite("faq.arc42.org-site")
	github.IssuesAndBugsCountForSite("arc42.org-site")
}
