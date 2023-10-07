package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

const APPVERSION = "0.0.2"
const PORTNR = ":8000"

const TEMPLATENAME = "arc42statistics.gohtml"

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
	Stats4Site SiteStats
}

var arcStats Arc42Statistics

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

func stats4Site(siteName string) SiteStats {
	return SiteStats{
		Site:         siteName,
		Visitors7d:   "v7d",
		Pageviews7d:  "pv7d",
		Visitors30d:  "v30d",
		Pageviews30d: "pv30d",
		Visitors12m:  "v12m",
		Pageviews12m: "pv12m",
	}
}

func main() {

	arcStats.AppVersion = APPVERSION
	arcStats.Timestamp = time.Now().Format("Mon Jan 2 15:04:05 MST 2006")
	arcStats.Stats4Site = stats4Site("docs.arc42.org")

	http.HandleFunc("/", statsHandler)

	log.Fatal(http.ListenAndServe(PORTNR, nil))
}
