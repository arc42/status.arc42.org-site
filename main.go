package main

import (
	"embed"
	"fmt"
	"github.com/rs/cors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"site-usage-statistics/internal/plausible"
	"site-usage-statistics/internal/types"
	"time"
)

const AppVersion = "0.0.2"
const PortNr = ":8042"
const HomeURL = "http://127.0.0.1"

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

func executeTemplate(w http.ResponseWriter, templatePath string) {
	//tpl, err := template.ParseFiles(filepath)

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
	executeTemplate(w, filepath.Join(TemplatesDir, HtmlTableTmpl))
}

// pingHandler returns the a message and the time
func pingHandler(w http.ResponseWriter, r *http.Request) {
	executeTemplate(w, filepath.Join(TemplatesDir, PingTmpl))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	executeTemplate(w, filepath.Join(TemplatesDir, "home.gohtml"))
}

func getPort() string {
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = PortNr
	}
	return httpPort
}

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

	// addressing CORS, see
	// https://github.com/rs/cors

	mux := http.NewServeMux()
	var realPortNr = getPort()

	// define some routes
	mux.HandleFunc("/statsTable", statsHTMLTableHandler)
	mux.HandleFunc("/ping", pingHandler)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://*", "https://*"},
		AllowedMethods: []string{http.MethodGet},
	})

	handler := cors.Default().Handler(mux)
	handler = c.Handler(handler)

	fmt.Println("Starting server on Port", realPortNr)

	log.Fatal(http.ListenAndServe(realPortNr, handler))

}
