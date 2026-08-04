// Command rendercheck renders arc42statistics.gohtml with fabricated,
// deliberately awkward data so the template can be verified without the
// external APIs. Scratch tool: not part of the deployed service.
package main

import (
	"arc42-status/internal/domain"
	"arc42-status/internal/types"
	"html/template"
	"os"
	"time"
)

func main() {
	stats := types.Arc42Statistics{
		AppVersion:        "1.1.0",
		LastUpdated:       time.Now(),
		LastUpdatedString: "4. August 2026, 16:26:04h",
		HowLongDidItTake:  "605",
		FlyRegion:         "ams",
		WhereDoesItRun:    "Amsterdam, Netherlands",
	}

	rows := []types.SiteStatsType{
		{Site: "arc42.org", Visitors7d: "2.269", PageViews7d: "5.274", Visitors30d: "10.079", PageViews30d: "22.873",
			Visitors12m: "121.581", PageViews12m: "288.375", Repo: "https://github.com/arc42/arc42.org-site",
			NrOfOpenIssues: 0, NrOfOpenBugs: 0, NrOfOpenPRs: 0},
		// every Plausible value unavailable -- the "n/a" path
		{Site: "arc42.de", Visitors7d: "n/a", PageViews7d: "n/a", Visitors30d: "n/a", PageViews30d: "n/a",
			Visitors12m: "n/a", PageViews12m: "n/a", Repo: "https://github.com/arc42/arc42.de-site",
			NrOfOpenIssues: 0, NrOfOpenBugs: 0, NrOfOpenPRs: 1},
		// an implausibly long hostname, to see what the column does
		{Site: "a-very-long-subdomain-name.arc42.org", Visitors7d: "1", PageViews7d: "2", Visitors30d: "3", PageViews30d: "4",
			Visitors12m: "5", PageViews12m: "6", Repo: "https://github.com/arc42/docs.arc42.org-site",
			NrOfOpenIssues: 4, NrOfOpenBugs: 3, NrOfOpenPRs: 0},
		{Site: "faq.arc42.org", Visitors7d: "103", PageViews7d: "638", Visitors30d: "375", PageViews30d: "1.456",
			Visitors12m: "5.414", PageViews12m: "16.935", Repo: "https://github.com/arc42/faq.arc42.org-site",
			NrOfOpenIssues: 4, NrOfOpenBugs: 0, NrOfOpenPRs: 0},
		{Site: "canvas.arc42.org", Visitors7d: "157", PageViews7d: "332", Visitors30d: "739", PageViews30d: "1.627",
			Visitors12m: "11.906", PageViews12m: "29.189", Repo: "https://github.com/arc42/canvas.arc42.org-site",
			NrOfOpenIssues: 5, NrOfOpenBugs: 1, NrOfOpenPRs: 3},
		{Site: "quality.arc42.org", Visitors7d: "1.033", PageViews7d: "2.537", Visitors30d: "3.812", PageViews30d: "10.475",
			Visitors12m: "41.055", PageViews12m: "127.973", Repo: "https://github.com/arc42/quality.arc42.org-site",
			NrOfOpenIssues: 12, NrOfOpenBugs: 0, NrOfOpenPRs: 1},
		// six-digit counts everywhere, to stress column widths
		{Site: "status.arc42.org", Visitors7d: "999.999", PageViews7d: "999.999", Visitors30d: "999.999", PageViews30d: "999.999",
			Visitors12m: "999.999", PageViews12m: "999.999", Repo: "https://github.com/arc42/status.arc42.org-site",
			NrOfOpenIssues: 19, NrOfOpenBugs: 1, NrOfOpenPRs: 1},
		{Site: "pdfminion.arc42.org", Visitors7d: "5", PageViews7d: "7", Visitors30d: "9", PageViews30d: "11",
			Visitors12m: "60", PageViews12m: "76", Repo: "https://github.com/arc42/PDFminion",
			NrOfOpenIssues: 11, NrOfOpenBugs: 2, NrOfOpenPRs: 1},
	}
	copy(stats.Stats4Site[:], rows)

	stats.Totals = types.TotalsForAllSites{
		SumOfVisitors7d: "5.705", SumOfPageViews7d: "16.911",
		SumOfVisitors30d: "23.141", SumOfPageViews30d: "63.688",
		SumOfVisitors12m: "292.201", SumOfPageViews12m: "829.733",
		TotalNrOfIssues: 55, TotalNrOfBugs: 7, TotalNrOfPRs: 7,
	}

	render("internal/api/arc42statistics.gohtml", os.Args[1], stats)

	// tiles, with untriaged work
	tiles := make([]types.SiteStatsType, 0, len(rows))
	for i, r := range rows {
		r.IsHub = r.Site == "arc42.org" || r.Site == "arc42.de"
		switch i {
		case 0:
			r.NrUntriaged = 4
			r.Untriaged = []types.UntriagedItem{
				{Title: "Move jQuery and DataTables.js from CDN to local directory", URL: "https://example.invalid/1", AgeString: "1 years", IsPR: true, Unlabelled: true},
				{Title: "add (private) entry page with statistics for more sites", URL: "https://example.invalid/2", AgeString: "1 years", Unlabelled: true},
				{Title: "A deliberately long issue title that has to wrap inside a narrow tile without pushing anything sideways", URL: "https://example.invalid/3", AgeString: "3 days"},
			}
		case 2:
			r.NrUntriaged = 1
			r.Untriaged = []types.UntriagedItem{
				{Title: "build(deps): bump json from 2.21.1 to 2.21.2", URL: "https://example.invalid/4", AgeString: "today", IsPR: true},
			}
		}
		tiles = append(tiles, r)
	}
	var ts types.Arc42Statistics
	copy(ts.Stats4Site[:], tiles)
	render("internal/api/tiles.gohtml", os.Args[2], types.TilesData{Tiles: domain.TilesInAttentionOrder(ts)})

	// tiles, all clear
	clear := make([]types.SiteStatsType, 0, len(rows))
	for _, r := range rows {
		r.IsHub = r.Site == "arc42.org" || r.Site == "arc42.de"
		r.NrUntriaged, r.Untriaged = 0, nil
		clear = append(clear, r)
	}
	var cs types.Arc42Statistics
	copy(cs.Stats4Site[:], clear)
	render("internal/api/tiles.gohtml", os.Args[3], types.TilesData{Tiles: domain.TilesInAttentionOrder(cs)})
}

func render(tplPath, outPath string, data any) {
	tpl := template.Must(template.ParseFiles(tplPath))
	out, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer func() { _ = out.Close() }()
	if err := tpl.Execute(out, data); err != nil {
		panic(err)
	}
}
