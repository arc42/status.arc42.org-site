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
	"strconv"
	"time"
)

const AppVersion = "0.1.7"
const PortNr = ":8043"

const GithubArc42URL = "https://github.com/arc42/"
const ShieldsGithubIssuesURL = "https://img.shields.io/github/issues-raw/arc42/"
const ShieldsGithubBugsURLPrefix = "https://img.shields.io/github/issues-search/arc42/"
const ShieldsBugSuffix = "?query=label%3Abug%20is%3Aopen&label=bugs&color=red"

// HomeIP is needed to deploy on fly.io
const HomeIP = "0.0.0.0"

var ArcStats types.Arc42Statistics
var SumOfStats types.SumOfAllSites

const TemplatesDir = "./web"

const HtmlTableTmpl = "arc42statistics.gohtml"
const PingTmpl = "ping.gohtml"

// embed templates into compiled binary, so we don't need to read from file system
// embeds the templates folder into variable embeddedTemplatesFolder
// === DON'T REMOVE THE COMMENT BELOW
//
//go:embed web/*.gohtml
var embeddedTemplatesFolder embed.FS

// sendCORSHeaders sets specific headers
// * calls from the "official" URL status.arc42.org are allowed
// * calls from localhost or "null" are also allowed
func sendCORSHeaders(w *http.ResponseWriter, r *http.Request) {

	// TODO: why do we use * here?

	var origin string
	origin = r.Host
	fmt.Printf("received request from host: %s\n", origin)

	// TODO: don't always allow origin, restrict to known hosts
	//(*w).Header().Set("Access-Control-Allow-Origin", origin)

	//w.Header().Set("Access-Control-Allow-Origin", "https://status.arc42.org")
	//w.Header().Set("Access-Control-Allow-Origin", "http://0.0.0.0:4000")

	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Headers", "Authorization, hx-target, hx-current-url, hx-request, hx-trigger")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

}

// executeTemplate handles the common stuff needed to process templates
func executeTemplate(w http.ResponseWriter, templatePath string, data any) {

	tpl, err := template.ParseFS(embeddedTemplatesFolder, templatePath)
	if err != nil {
		log.Printf("parsing template: %v", err)
		http.Error(w, "There was an error parsing the template {#err}.", http.StatusInternalServerError)
		return
	}
	err = tpl.Execute(w, data)
	if err != nil {
		log.Printf("executing template: %v", err)
		http.Error(w, "There was an error executing the template {#err}.", http.StatusInternalServerError)
		return
	}
}

// statsHTMLTableHandler returns the usage statistics as html table
// 1. start timer
// 2. updates ArcStats
// 3. sets required http headers needed for CORS
// 4. renders the output via HtmlTableTmpl
func statsHTMLTableHandler(w http.ResponseWriter, r *http.Request) {

	// 1. set timer
	var startOfProcessing = time.Now()

	// 2. update ArcStats
	ArcStats = loadStats4AllSites()

	// remember how long it took to update statistics
	ArcStats.HowLongDidItTake = strconv.FormatInt(time.Since(startOfProcessing).Milliseconds(), 10)

	// 3. handle the CORS stuff
	sendCORSHeaders(&w, r)

	// 4. finally, render the template
	executeTemplate(w, filepath.Join(TemplatesDir, HtmlTableTmpl), ArcStats)
}

// pingHandler returns a message and the time
func pingHandler(w http.ResponseWriter, r *http.Request) {

	// need to set specific headers, depending on request origin
	sendCORSHeaders(&w, r)

	var Host string = r.Host
	var RequestURI string = r.RequestURI

	fmt.Printf("Host = %s\n", Host)
	fmt.Printf("RequestURI = %s\n", RequestURI)
	executeTemplate(w, filepath.Join(TemplatesDir, PingTmpl), r)
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

	// shields.io bug URLS look like that:https://img.shields.io/github/issues-search/arc42/quality.arc42.org-site?query=label%3Abug%20is%3Aopen&label=bugs&color=red
	stats.BugBadgeURL = ShieldsGithubBugsURLPrefix + stats.Site + "-site" + ShieldsBugSuffix
}

// loadStats4AllSites calls the plausible.io API to retrieve all statistics
// and sets several site constants (URLs)
func loadStats4AllSites() types.Arc42Statistics {

	fmt.Printf("loading statistics...\n")

	location, _ := time.LoadLocation("Europe/Berlin")

	// Get the current time in Bielefeld, the town that presumably does not exist
	bielefeldTime := time.Now().In(location)

	a42s := types.Arc42Statistics{
		AppVersion:        AppVersion,
		LastUpdated:       bielefeldTime,
		LastUpdatedString: bielefeldTime.Format("2. January 2006, 15:04:03h"),
	}

	for index, site := range types.Arc42sites {
		a42s.Stats4Site[index].Site = site

		// query the number of open bugs from GitHub
		a42s.Stats4Site[index].NrOfOpenBugs = 1

		// TODO: let StatsForSite update the Stats4Site and the Sums struct
		// set the statistic data from plausible.io
		plausible.StatsForSite(site, &a42s.Stats4Site[index], &a42s.Sums)

		// set some URLs so the templates get smaller
		setURLsForSite(&a42s.Stats4Site[index])
	}
	return a42s
}

// printServerDetails displays a few details about this program,
// mainly to give admins some idea what version is currently running
// and where in the fly.io cloud the service is deployed.
func printServerDetails() {

	fmt.Printf("Starting server version %s on Port %s at %s\n", AppVersion, getPort(), time.Now().Format("2. January 2006, 15:04h"))

	// assumes we're running this program within the fly.io cloud.
	// There, the env variable FLY_REGION should be set.
	// If this variable is empty, we assume we're running locally
	region := os.Getenv("FLY_REGION")

	if region != "" {
		fmt.Printf("Running in fly.io region %s\n", region)
	}
}

func main() {

	mux := http.NewServeMux()

	// define some routes
	mux.HandleFunc("/statsTable", statsHTMLTableHandler)
	mux.HandleFunc("/statistics", statsHTMLTableHandler)
	mux.HandleFunc("/stats", statsHTMLTableHandler)
	mux.HandleFunc("/ping", pingHandler)

	printServerDetails()

	// TODO why are we setting HomeIP?
	log.Fatal(http.ListenAndServe(HomeIP+getPort(), mux))

}
