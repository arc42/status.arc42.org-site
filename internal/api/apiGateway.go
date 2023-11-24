package api

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"site-usage-statistics/internal/domain"
	"site-usage-statistics/internal/fly"
	"strconv"
	"time"
)

const PortNr = ":8043"

// HomeIP is needed to deploy on fly.io
const homeIP = "0.0.0.0"

const TemplatesDir = ""
const HtmlTableTmpl = "arc42statistics.gohtml"
const PingTmpl = "ping.gohtml"

// embed templates into compiled binary, so we don't need to read from file system
// embeds the templates folder into variable embeddedTemplatesFolder
// === DON'T REMOVE THE COMMENT BELOW
//
//go:embed *.gohtml
var embeddedTemplatesFolder embed.FS

// statsHTMLTableHandler returns the usage statistics as html table
// 1. start timer
// 2. updates ArcStats
// 3. sets required http headers needed for CORS
// 4. renders the output via HtmlTableTmpl
func statsHTMLTableHandler(w http.ResponseWriter, r *http.Request) {

	// 1. set timer
	var startOfProcessing = time.Now()

	// 2. update ArcStats
	domain.ArcStats = domain.LoadStats4AllSites()

	// remember how long it took to update statistics
	domain.ArcStats.HowLongDidItTake = strconv.FormatInt(time.Since(startOfProcessing).Milliseconds(), 10)

	//find out  where this service is running
	domain.ArcStats.FlyRegion, domain.ArcStats.WhereDoesItRun = fly.RegionAndLocation()

	// 3. handle the CORS stuff
	SendCORSHeaders(&w, r)

	// 4. finally, render the template
	executeTemplate(w, filepath.Join(TemplatesDir, HtmlTableTmpl), domain.ArcStats)
}

// pingHandler returns a message and the time
func pingHandler(w http.ResponseWriter, r *http.Request) {

	// need to set specific headers, depending on request origin
	SendCORSHeaders(&w, r)

	var Host string = r.Host
	var RequestURI string = r.RequestURI

	fmt.Printf("Host = %s\n", Host)
	fmt.Printf("RequestURI = %s\n", RequestURI)
	executeTemplate(w, filepath.Join(TemplatesDir, PingTmpl), r)
}

// sendCORSHeaders sets specific headers
// * calls from the "official" URL status.arc42.org are allowed
// * calls from localhost or "null" are also allowed
func SendCORSHeaders(w *http.ResponseWriter, r *http.Request) {

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

func getPort() string {
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = PortNr
	}
	return httpPort
}

// executeTemplate handles the common stuff needed to process templates
func executeTemplate(w http.ResponseWriter, templatePath string, data any) {

	tpl, err := template.ParseFS(embeddedTemplatesFolder, templatePath)
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "There was an error parsing the template "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = tpl.Execute(w, data)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "There was an error executing the template "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func logRequestHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL, time.Since(start))
	})
}

// PrintServerDetails displays a few details about this program,
// mainly to give admins some idea what version is currently running
// and where in the fly.io cloud the service is deployed.
func PrintServerDetails(appVersion string) {

	fmt.Printf("Starting API server, version %s on Port %s at %s\n", appVersion, getPort(), time.Now().Format("2. January 2006, 15:04h"))

	// assumes we're running this program within the fly.io cloud.
	// There, the env variable FLY_REGION should be set.
	// If this variable is empty, we assume we're running locally
	region, location := fly.RegionAndLocation()
	fmt.Printf("Server region is %s/%s", region, location)
}

// StartAPIServer creates an http ServeMux with a few predefined routes.
func StartAPIServer() {

	mux := http.NewServeMux()

	// define some routes
	mux.HandleFunc("/statsTable", statsHTMLTableHandler)
	mux.HandleFunc("/statistics", statsHTMLTableHandler)
	mux.HandleFunc("/stats", statsHTMLTableHandler)
	mux.HandleFunc("/ping", pingHandler)

	// wrap ServeMux with logging
	loggedMux := logRequestHandler(mux)

	// TODO why are we setting HomeIP?
	err := http.ListenAndServe(homeIP+getPort(), loggedMux)

	if err != nil {
		log.Fatalf("API server failed to start: %v", err)
	}

}
