package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

const APPVERSION = "0.0.2"
const PORTNR = ":8000"

const TEMPLATENAME = "arc42statistics.gohtml"

var arc42sites = [6]string{
	"arc42.org",
	"arc42.de",
	"docs.arc42.org",
	"faq.arc42.org",
	"canvas.arc42.org",
	"quality.arc42.org",
}

type SiteStats struct {
	Site         string
	Visitors7d   string
	Pageviews7d  string
	Visitors30d  string
	Pageviews30d string
	Visitors12m  string
	Pageviews12m string
}

type Arc42Statistics struct {
	AppVersion string
	Timestamp  string
	Stats4Site [len(arc42sites)]SiteStats
}

var arcStats Arc42Statistics

func populateOneSite(siteName string) SiteStats {

	return SiteStats{
		Site:         siteName,
		Visitors7d:   strconv.Itoa(rand.Intn(10001)),
		Pageviews7d:  strconv.Itoa(rand.Intn(10001)),
		Visitors30d:  strconv.Itoa(rand.Intn(10001)),
		Pageviews30d: strconv.Itoa(rand.Intn(10001)),
		Visitors12m:  strconv.Itoa(rand.Intn(10001)),
		Pageviews12m: strconv.Itoa(rand.Intn(10001)),
	}
}

func executeTemplate(w http.ResponseWriter, filepath string) {
	tpl, err := template.ParseFiles(filepath)
	if err != nil {
		log.Printf("parsing template: %v", err)
		http.Error(w, "There was an error parsing the template.", http.StatusInternalServerError)
		return
	}
	err = tpl.Execute(w, arcStats)
	if err != nil {
		log.Printf("executing template: %v", err)
		http.Error(w, "There was an error executing the template.", http.StatusInternalServerError)
		return
	}
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	var tplPath = filepath.Join("templates", TEMPLATENAME)
	executeTemplate(w, tplPath)
}

func loadStats4AllSites() Arc42Statistics {
	a42s := Arc42Statistics{
		AppVersion: APPVERSION,
		Timestamp:  time.Now().Format("Mon Jan 2 15:04:05 MST 2006")}
	for index, site := range arc42sites {
		fmt.Print("index:", index)
		a42s.Stats4Site[index] = populateOneSite(site)
	}

	return a42s
}

func main() {

	arcStats = loadStats4AllSites()

	fmt.Print(arcStats)
	http.HandleFunc("/", statsHandler)

	log.Fatal(http.ListenAndServe(PORTNR, nil))
}
