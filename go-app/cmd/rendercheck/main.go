// Command rendercheck renders the statistics table and the dashboard tiles with
// fabricated, deliberately awkward data, so both templates can be verified
// without the external APIs. It then checks the table's column arithmetic in the
// rendered HTML - header colspans against body and footer cell counts - and
// fails loudly when they disagree.
//
// Scratch tool: not part of the deployed service. It imports domain, which
// transitively imports plausible, whose init() insists on an API key, so run it
// as:
//
//	source ./set-api-keys.sh >/dev/null && go run ./cmd/rendercheck [outdir]
//
// No API call is made; only the key check runs.
package main

import (
	"arc42-status/internal/domain"
	"arc42-status/internal/types"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

// longTitle is the kind of issue title that breaks narrow layouts.
const longTitle = "A deliberately long issue title that has to wrap inside a narrow tile without pushing anything sideways, and which nobody would ever shorten because the whole sentence is apparently load-bearing"

func main() {
	outDir := filepath.Join(os.TempDir(), "rendercheck")
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Printf("cannot create output directory %s: %v\n", outDir, err)
		os.Exit(1)
	}

	rows := fixtureRows()

	stats := types.Arc42Statistics{
		AppVersion:        "1.2.0",
		LastUpdated:       time.Now(),
		LastUpdatedString: "4. August 2026, 16:26:04h",
		HowLongDidItTake:  "605",
		FlyRegion:         "ams",
		WhereDoesItRun:    "Amsterdam, Netherlands",
		Totals: types.TotalsForAllSites{
			SumOfVisitors7d: "5.705", SumOfPageViews7d: "16.911",
			SumOfVisitors30d: "23.141", SumOfPageViews30d: "63.688",
			SumOfVisitors12m: "292.201", SumOfPageViews12m: "829.733",
			TotalNrOfIssues: 55, TotalNrOfBugs: 7, TotalNrOfPRs: 7,
		},
	}
	copy(stats.Stats4Site[:], rows)

	tablePath := filepath.Join(outDir, "table.html")
	tableHTML := render("internal/api/arc42statistics.gohtml", tablePath, stats)

	// tiles, with work waiting
	var withWork types.Arc42Statistics
	copy(withWork.Stats4Site[:], rows)
	tilesPath := filepath.Join(outDir, "tiles.html")
	render("internal/api/tiles.gohtml", tilesPath, types.TilesData{
		Tiles:             domain.TilesInAttentionOrder(withWork),
		LastUpdatedString: stats.LastUpdatedString,
	})

	// tiles, all clear: nothing untriaged anywhere
	clearRows := make([]types.SiteStatsType, 0, len(rows))
	for _, r := range rows {
		r.NrUntriaged = 0
		clearRows = append(clearRows, r)
	}
	var allClear types.Arc42Statistics
	copy(allClear.Stats4Site[:], clearRows)
	clearPath := filepath.Join(outDir, "tiles-allclear.html")
	render("internal/api/tiles.gohtml", clearPath, types.TilesData{
		Tiles:             domain.TilesInAttentionOrder(allClear),
		LastUpdatedString: stats.LastUpdatedString,
	})

	// the per-site detail fragment, for the property with the longest lists
	detailPath := filepath.Join(outDir, "siteDetail.html")
	detail, found := domain.StatsForKey(withWork, "arc42-template")
	if !found {
		fmt.Println("fixture for arc42-template is missing")
		os.Exit(1)
	}
	render("internal/api/siteDetail.gohtml", detailPath, types.SiteDetailData{
		Site:              detail,
		LastUpdatedString: stats.LastUpdatedString,
	})

	// the traffic fragment, for a property that IS measured -- the template
	// repository is not, and its "no Plausible site" branch would never show
	// the six figures the layout has to cope with
	trafficPath := filepath.Join(outDir, "siteTraffic.html")
	traffic, found := domain.StatsForKey(withWork, "status.arc42.org")
	if !found {
		fmt.Println("fixture for status.arc42.org is missing")
		os.Exit(1)
	}
	render("internal/api/siteTraffic.gohtml", trafficPath, types.SiteDetailData{
		Site:              traffic,
		LastUpdatedString: stats.LastUpdatedString,
	})

	fmt.Printf("rendered %s\n", detailPath)
	fmt.Printf("rendered %s\n", trafficPath)
	fmt.Printf("rendered %s\n", tablePath)
	fmt.Printf("rendered %s\n", tilesPath)
	fmt.Printf("rendered %s\n", clearPath)

	if !checkTableColumns(tableHTML) {
		os.Exit(1)
	}
}

// fixtureRows are the ten properties, each carrying a different way of being
// awkward: no traffic measurement at all, every metric unavailable, an empty
// repository, more open items than a tile lists, six-digit counts, a hostname
// nobody sized a column for.
func fixtureRows() []types.SiteStatsType {
	return []types.SiteStatsType{
		// hub, healthy numbers, a full open list including a title that will not fit
		{Site: "arc42.org", Host: "arc42.org", HasTraffic: true, InTable: true, IsHub: true,
			Visitors7d: "2.269", PageViews7d: "5.274", Visitors30d: "10.079", PageViews30d: "22.873",
			Visitors12m: "121.581", PageViews12m: "288.375", Repo: "https://github.com/arc42/arc42.org-site",
			NrOfOpenIssues: 4, NrOfOpenBugs: 1, NrOfOpenPRs: 2, NrUntriaged: 3,
			OpenItems: []types.RepoItem{
				{Title: "Move jQuery and DataTables.js from CDN to local directory", URL: "https://example.invalid/1", AgeString: "1 years", IsPR: true, Unlabelled: true},
				{Title: "add (private) entry page with statistics for more sites", URL: "https://example.invalid/2", AgeString: "1 years", Unlabelled: true},
				{Title: longTitle, URL: "https://example.invalid/3", AgeString: "3 days"},
				{Title: "Tiny", URL: "https://example.invalid/4", AgeString: "today"},
				{Title: "build(deps): bump nokogiri from 1.16.5 to 1.16.7", URL: "https://example.invalid/5", AgeString: "5 weeks", IsPR: true},
			},
			RecentlyClosed: []types.ClosedItem{
				{Title: longTitle, URL: "https://example.invalid/c1", ClosedAgo: "today"},
				{Title: "Fix broken anchor in the method chapter", URL: "https://example.invalid/c2", IsPR: true, ClosedAgo: "1 day ago"},
				{Title: "Update imprint", URL: "https://example.invalid/c3", ClosedAgo: "3 weeks ago"},
			}},

		// hub twin: Plausible answered with errors, and the repository is empty --
		// no open items, no closed items, no counts. The "nothing at all" case.
		{Site: "arc42.de", Host: "arc42.de", HasTraffic: true, InTable: true, IsHub: true,
			Visitors7d: types.NotAvailable, PageViews7d: types.NotAvailable,
			Visitors30d: types.NotAvailable, PageViews30d: types.NotAvailable,
			Visitors12m: types.NotAvailable, PageViews12m: types.NotAvailable,
			Repo: "https://github.com/arc42/arc42.de-site"},

		// an implausibly long hostname, and more open items than a tile may list
		{Site: "a-very-long-subdomain-name.arc42.org", Host: "a-very-long-subdomain-name.arc42.org", HasTraffic: true, InTable: true,
			Visitors7d: "1", PageViews7d: "2", Visitors30d: "3", PageViews30d: "4",
			Visitors12m: "5", PageViews12m: "6", Repo: "https://github.com/arc42/docs.arc42.org-site",
			NrOfOpenIssues: 21, NrOfOpenBugs: 3, NrOfOpenPRs: 6, NrUntriaged: 12,
			OpenItems: []types.RepoItem{
				{Title: "one", URL: "https://example.invalid/6", AgeString: "today", Unlabelled: true},
				{Title: "two", URL: "https://example.invalid/7", AgeString: "1 day", IsPR: true},
				{Title: "three", URL: "https://example.invalid/8", AgeString: "3 days"},
				{Title: "four", URL: "https://example.invalid/9", AgeString: "2 weeks", Unlabelled: true},
				{Title: "five", URL: "https://example.invalid/10", AgeString: "4 months", IsPR: true},
				{Title: "six -- beyond the cap, the template must still cope", URL: "https://example.invalid/11", AgeString: "2 years"},
				{Title: "seven -- likewise", URL: "https://example.invalid/12", AgeString: "3 years"},
			},
			RecentlyClosed: []types.ClosedItem{
				{Title: "closed a while back", URL: "https://example.invalid/c4", ClosedAgo: "11 months ago"},
			}},

		{Site: "faq.arc42.org", Host: "faq.arc42.org", HasTraffic: true, InTable: true,
			Visitors7d: "103", PageViews7d: "638", Visitors30d: "375", PageViews30d: "1.456",
			Visitors12m: "5.414", PageViews12m: "16.935", Repo: "https://github.com/arc42/faq.arc42.org-site",
			NrOfOpenIssues: 4, NrOfOpenPRs: 0, NrUntriaged: 1,
			OpenItems: []types.RepoItem{
				{Title: "Add a question about arc42 and C4", URL: "https://example.invalid/13", AgeString: "6 days", Unlabelled: true},
			},
			RecentlyClosed: []types.ClosedItem{
				{Title: "Typo in question 42", URL: "https://example.invalid/c5", ClosedAgo: "2 days ago"},
				{Title: "bump jekyll", URL: "https://example.invalid/c6", IsPR: true, ClosedAgo: "1 week ago"},
			}},

		{Site: "canvas.arc42.org", Host: "canvas.arc42.org", HasTraffic: true, InTable: true,
			Visitors7d: "157", PageViews7d: "332", Visitors30d: "739", PageViews30d: "1.627",
			Visitors12m: "11.906", PageViews12m: "29.189", Repo: "https://github.com/arc42/canvas.arc42.org-site",
			NrOfOpenIssues: 1, NrOfOpenBugs: 1, NrOfOpenPRs: 1, NrUntriaged: 0,
			OpenItems: []types.RepoItem{
				{Title: "Canvas print layout drops the last column", URL: "https://example.invalid/14", AgeString: "8 months"},
				{Title: "Update dependencies", URL: "https://example.invalid/15", AgeString: "7 months", IsPR: true},
			}},

		{Site: "quality.arc42.org", Host: "quality.arc42.org", HasTraffic: true, InTable: true,
			Visitors7d: "1.033", PageViews7d: "2.537", Visitors30d: "3.812", PageViews30d: "10.475",
			Visitors12m: "41.055", PageViews12m: "127.973", Repo: "https://github.com/arc42/quality.arc42.org-site",
			NrOfOpenIssues: 12, NrOfOpenPRs: 1, NrUntriaged: 2,
			OpenItems: []types.RepoItem{
				{Title: "Quality model: add a source for ISO 25010:2023", URL: "https://example.invalid/16", AgeString: "12 days", Unlabelled: true},
			}},

		// six-digit counts everywhere, to stress column widths
		{Site: "status.arc42.org", Host: "status.arc42.org", HasTraffic: true, InTable: true,
			Visitors7d: "999.999", PageViews7d: "999.999", Visitors30d: "999.999", PageViews30d: "999.999",
			Visitors12m: "999.999", PageViews12m: "999.999", Repo: "https://github.com/arc42/status.arc42.org-site",
			NrOfOpenIssues: 19, NrOfOpenBugs: 1, NrOfOpenPRs: 1, NrUntriaged: 5,
			OpenItems: []types.RepoItem{
				{Title: "Availability monitoring with a GitHub-Actions prober (ADR-0019)", URL: "https://example.invalid/17", AgeString: "today", Unlabelled: true},
				{Title: "Per-site subpages", URL: "https://example.invalid/18", AgeString: "today", Unlabelled: true},
			},
			RecentlyClosed: []types.ClosedItem{
				{Title: "Dashboard tiles: what needs me, above what the numbers say", URL: "https://example.invalid/c7", IsPR: true, ClosedAgo: "today"},
			}},

		{Site: "pdfminion.arc42.org", Host: "pdfminion.arc42.org", HasTraffic: true,
			Visitors7d: "5", PageViews7d: "7", Visitors30d: "9", PageViews30d: "11",
			Visitors12m: "60", PageViews12m: "76", Repo: "https://github.com/arc42/PDFminion",
			NrOfOpenIssues: 11, NrOfOpenBugs: 2, NrOfOpenPRs: 1, NrUntriaged: 0,
			OpenItems: []types.RepoItem{
				{Title: "Support --config with relative paths", URL: "https://example.invalid/19", AgeString: "5 months"},
			},
			RecentlyClosed: []types.ClosedItem{
				{Title: "Release v2.0.1", URL: "https://example.invalid/c8", IsPR: true, ClosedAgo: "2 months ago"},
			}},

		// brand-new property: measured, but everything is genuinely zero
		{Site: "trainings.arc42.org", Host: "trainings.arc42.org", HasTraffic: true,
			Visitors7d: "0", PageViews7d: "0", Visitors30d: "0", PageViews30d: "0",
			Visitors12m: "0", PageViews12m: "0", Repo: "https://github.com/arc42/trainings.arc42.org-site"},

		// no Plausible site at all: every metric unavailable, never zero
		{Site: "meta.arc42.org", Host: "meta.arc42.org", HasTraffic: false,
			Visitors7d: types.NotAvailable, PageViews7d: types.NotAvailable,
			Visitors30d: types.NotAvailable, PageViews30d: types.NotAvailable,
			Visitors12m: types.NotAvailable, PageViews12m: types.NotAvailable,
			Repo: "https://github.com/arc42/meta.arc42.org", NrOfOpenIssues: 2, NrOfOpenPRs: 0, NrUntriaged: 2,
			OpenItems: []types.RepoItem{
				{Title: "BRAND.md: register trainings.arc42.org", URL: "https://example.invalid/20", AgeString: "today", Unlabelled: true},
				{Title: "ADR for the colour token interface", URL: "https://example.invalid/21", AgeString: "2 days", Unlabelled: true},
			}},

		// a repository with no site of its own: no host, no traffic, no table
		// row -- and the longest open list of the family
		{Site: "arc42-template", Repo: "https://github.com/arc42/arc42-template",
			Visitors7d: types.NotAvailable, PageViews7d: types.NotAvailable,
			Visitors30d: types.NotAvailable, PageViews30d: types.NotAvailable,
			Visitors12m: types.NotAvailable, PageViews12m: types.NotAvailable,
			NrOfOpenIssues: 22, NrOfOpenBugs: 4, NrOfOpenPRs: 3, NrUntriaged: 6,
			OpenItems: []types.RepoItem{
				{Title: "Golden master for the asciidoc export drifts on Windows line endings", URL: "https://example.invalid/22", AgeString: "today", Unlabelled: true},
				{Title: "Add a Spanish translation of chapter 8", URL: "https://example.invalid/23", AgeString: "4 days"},
				{Title: "build(deps): bump asciidoctor-pdf", URL: "https://example.invalid/24", AgeString: "1 day", IsPR: true},
				{Title: "docx template: heading numbering restarts at chapter 5", URL: "https://example.invalid/25", AgeString: "3 weeks", Unlabelled: true},
				{Title: "Markdown flavour: GitHub vs CommonMark tables", URL: "https://example.invalid/26", AgeString: "5 months"},
				{Title: "Drop the obsolete .odt variant", URL: "https://example.invalid/27", AgeString: "2 years", IsPR: true},
			},
			RecentlyClosed: []types.ClosedItem{
				{Title: "Release 8.2 of the template", URL: "https://example.invalid/c9", IsPR: true, ClosedAgo: "1 week ago"},
				{Title: "Typo in chapter 4 of the German docx", URL: "https://example.invalid/c10", ClosedAgo: "2 weeks ago"},
				{Title: "Add Ukrainian translation", URL: "https://example.invalid/c11", IsPR: true, ClosedAgo: "1 months ago"},
				{Title: "Broken link in the LaTeX variant", URL: "https://example.invalid/c12", ClosedAgo: "2 months ago"},
			}},
	}
}

func render(tplPath, outPath string, data any) string {
	tpl := template.Must(template.ParseFiles(tplPath))
	out, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer func() { _ = out.Close() }()
	if err := tpl.Execute(out, data); err != nil {
		panic(err)
	}
	rendered, err := os.ReadFile(outPath)
	if err != nil {
		panic(err)
	}
	return string(rendered)
}

// ---------------------------------------------------------------------------
// column arithmetic
// ---------------------------------------------------------------------------

// Go's RE2 has no backreferences, so each table section gets its own pattern
// rather than one pattern closing on \1.
var (
	sectionRes = map[string]*regexp.Regexp{
		"thead": regexp.MustCompile(`(?is)<thead\b[^>]*>(.*?)</thead>`),
		"tbody": regexp.MustCompile(`(?is)<tbody\b[^>]*>(.*?)</tbody>`),
		"tfoot": regexp.MustCompile(`(?is)<tfoot\b[^>]*>(.*?)</tfoot>`),
	}
	sectionOrder = []string{"thead", "tbody", "tfoot"}
	rowRe        = regexp.MustCompile(`(?is)<tr\b[^>]*>(.*?)</tr>`)
	cellRe       = regexp.MustCompile(`(?is)<t([hd])\b([^>]*)>`)
	spanRe       = regexp.MustCompile(`(?is)\b(colspan|rowspan)\s*=\s*"(\d+)"`)
)

// checkTableColumns lays the rendered table out the way a browser would -
// honouring both colspan and rowspan - and reports the width of every row.
// A table whose header promises more columns than its body delivers is broken
// for screen readers and for DataTables alike, and neither says so out loud.
func checkTableColumns(html string) bool {
	fmt.Println("\ncolumn arithmetic of the rendered statistics table")
	fmt.Println("--------------------------------------------------")

	occupied := make([]int, 64) // per column: how many further rows it blocks
	widths := map[string][]int{}

	// thead, tbody, tfoot in document order, so rowspans carry down correctly
	for _, name := range sectionOrder {
		for _, section := range sectionRes[name].FindAllStringSubmatch(html, -1) {
			for _, row := range rowRe.FindAllStringSubmatch(section[1], -1) {
				widths[name] = append(widths[name], rowWidth(row[1], occupied))
			}
		}
	}

	ok := true
	var expected int
	for _, name := range sectionOrder {
		for i, w := range widths[name] {
			if expected == 0 {
				expected = w
			}
			verdict := "ok"
			if w != expected {
				verdict, ok = "MISMATCH", false
			}
			fmt.Printf("  %-5s row %d: %2d columns   %s\n", name, i+1, w, verdict)
		}
	}
	fmt.Printf("  expected width: %d columns -- ", expected)
	if ok {
		fmt.Println("header, body and footer agree")
	} else {
		fmt.Println("ROWS DISAGREE")
	}
	return ok
}

// rowWidth places one row's cells into the running occupancy map and returns
// how many columns the row covers, cells carried down by rowspan included.
func rowWidth(row string, occupied []int) int {
	col := 0
	for _, cell := range cellRe.FindAllStringSubmatch(row, -1) {
		colspan, rowspan := 1, 1
		for _, span := range spanRe.FindAllStringSubmatch(cell[2], -1) {
			n, _ := strconv.Atoi(span[2])
			if span[1] == "colspan" {
				colspan = n
			} else {
				rowspan = n
			}
		}
		for col < len(occupied) && occupied[col] > 0 {
			col++
		}
		for k := 0; k < colspan && col+k < len(occupied); k++ {
			occupied[col+k] = rowspan
		}
		col += colspan
	}

	width := 0
	for i := range occupied {
		if occupied[i] > 0 {
			width++
			occupied[i]--
		}
	}
	return width
}
