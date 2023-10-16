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

const AppVersion = "0.0.4d"
const PortNr = ":8043"

// HomeIP is needed to deploy on fly.io
const HomeIP = "0.0.0.0"

var Arc42sites = [6]string{
	"arc42.org",
	"arc42.de",
	"docs.arc42.org",
	"faq.arc42.org",
	"canvas.arc42.org",
	"quality.arc42.org",
}

type Arc42Statistics struct {
	AppVersion string
	Timestamp  string
	Stats4Site [len(Arc42sites)]types.SiteStats
}

var ArcStats Arc42Statistics

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

	w.Header().Set("Access-Control-Allow-Origin", "https://status.arc42.org")
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

// loadStats4AllSites calls the plausible.io API to retrieve all statistics
// than
func loadStats4AllSites() Arc42Statistics {
	a42s := Arc42Statistics{
		AppVersion: AppVersion,
		Timestamp:  time.Now().Format("2. January 2006, 15:04h")}

	for index, site := range Arc42sites {
		a42s.Stats4Site[index] = plausible.StatsForSite(site)
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
