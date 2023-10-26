package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"site-usage-statistics/internal/plausible"
	"site-usage-statistics/internal/types"
	"time"
)

const AppVersion = "0.0.5b"
const PortNr = ":8043"

const GithubArc42URL = "https://github.com/arc42/"
const ShieldsGithubIssuesURL = "https://img.shields.io/github/issues-raw/arc42/"
const ShieldsGithubBugsURLPrefix = "https://img.shields.io/github/issues-search/arc42/"
const ShieldsBugSuffix = "?query=label%3Abug%20is%3Aopen&label=bugs&color=red"

// HomeIP is needed to deploy on fly.io
const HomeIP = "0.0.0.0"

var ArcStats types.Arc42Statistics

const TemplatesDir = "./web"

const HtmlTableTmpl = "arc42statistics.gohtml"
const PingTmpl = "ping.gohtml"

// embed templates into compiled binary, so we don't need to read from file system
//
//go:embed web/*.gohtml
var embeddedTplsFolder embed.FS // embeds the templates folder into variable embeddedTplsFolder

// enableCORS sets specific headers
// * calls from the "official" URL status.arc42.org are allowed
// * calls from localhost or "null" are also allowed
func enableCORS(w *http.ResponseWriter, r *http.Request) {

	var origin string
	origin = r.Host
	fmt.Printf("received request from host: %s\n", origin)

	// TODO: don't always allow origin, restrict to known hosts
	(*w).Header().Set("Access-Control-Allow-Origin", origin)
}

// executeTemplate handles the common stuff needed to process templates
func executeTemplate(w http.ResponseWriter, templatePath string) {

	tpl, err := template.ParseFS(embeddedTplsFolder, templatePath)
	if err != nil {
		log.Printf("parsing template: %v", err)
		http.Error(w, "There was an error parsing the template {#err}.", http.StatusInternalServerError)
		return
	}
	err = tpl.Execute(w, ArcStats)
	if err != nil {
		log.Printf("executing template: %v", err)
		http.Error(w, "There was an error executing the template {#err}.", http.StatusInternalServerError)
		return
	}
}

// statsHTMLTableHandler returns the usage statistics as html table
func statsHTMLTableHandler(w http.ResponseWriter, r *http.Request) {

	//	w.Header().Set("Access-Control-Allow-Origin", "https://status.arc42.org")
	w.Header().Set("Access-Control-Allow-Origin", "http://0.0.0.0:4000")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, hx-target, hx-current-url, hx-request, hx-trigger")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	executeTemplate(w, filepath.Join(TemplatesDir, HtmlTableTmpl))
}

// pingHandler returns a message and the time
func pingHandler(w http.ResponseWriter, r *http.Request) {
	// need to set specific headers, depending on request origin
	enableCORS(&w, r)

	executeTemplate(w, filepath.Join(TemplatesDir, PingTmpl))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "https://status.arc42.org")

	executeTemplate(w, filepath.Join(TemplatesDir, "home.gohtml"))
}

func getPort() string {
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = PortNr
	}
	return httpPort
}

// setURLsForSite sets some constants for use within the templates
// (to avoid overly long string constants within these templates)
func setURLsForSite(stats *types.SiteStats) {

	// all arc42 website repos follow this naming convention, e.g. arc42.org-site
	stats.Repo = GithubArc42URL + stats.Site + "-site"

	// shields.io issues URLS look like that: https://img.shields.io/github/issues-raw/arc42/arc42.org-site
	stats.IssueBadgeURL = ShieldsGithubIssuesURL + stats.Site + "-site"

	// shields.io bug URLS loook like that:https://img.shields.io/github/issues-search/arc42/quality.arc42.org-site?query=label%3Abug%20is%3Aopen&label=bugs&color=red
	stats.BugBadgeURL = ShieldsGithubBugsURLPrefix + stats.Site + "-site" + ShieldsBugSuffix
}

// loadStats4AllSites calls the plausible.io API to retrieve all statistics
// and sets several site constants (URLs)
func loadStats4AllSites() types.Arc42Statistics {

	location, _ := time.LoadLocation("Europe/Berlin")

	// Get the current time in Bielefeld, the town that presumably does not exist
	bielefeldTime := time.Now().In(location).Format("2. January 2006, 15:04h")

	a42s := types.Arc42Statistics{
		AppVersion: AppVersion,
		Timestamp:  bielefeldTime + " (@Cologne)",
	}

	for index, site := range types.Arc42sites {
		a42s.Stats4Site[index].Site = site

		// set the statistic data from plausible.io
		plausible.StatsForSite(site, &a42s.Stats4Site[index])

		// set some URLs so the templates get smaller
		setURLsForSite(&a42s.Stats4Site[index])
	}
	return a42s
}

func main() {

	ArcStats = loadStats4AllSites()

	realPortNr := getPort()
	mux := http.NewServeMux()

	// define some routes
	mux.HandleFunc("/statsTable", statsHTMLTableHandler)
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/", homeHandler)

	fmt.Printf("Starting server version %s on Port %s\n", AppVersion, realPortNr)

	log.Fatal(http.ListenAndServe(HomeIP+realPortNr, mux))

}
