package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"arc42-status/internal/auth"
	"arc42-status/internal/database"
	"arc42-status/internal/domain"
	"arc42-status/internal/env"
	"arc42-status/internal/fly"
	"arc42-status/internal/probe"
	"arc42-status/internal/types"
	"embed"
)

const PortNr = ":8043"

// HomeIP is needed to deploy on fly.io
const homeIP = "0.0.0.0"

const TemplatesDir = ""
const HtmlTableTmpl = "arc42statistics.gohtml"
const PingTmpl = "ping.gohtml"
const TilesTmpl = "tiles.gohtml"
const SiteDetailTmpl = "siteDetail.gohtml"
const SiteTrafficTmpl = "siteTraffic.gohtml"
const SiteAvailabilityTmpl = "siteAvailability.gohtml"
const RollupPageTmpl = "rollupPage.gohtml"

func init() {
	log.Debug().Msg("apiGateway initialized ")
}

// embed templates into compiled binary, so we don't need to read from file system
// embeds the templates folder into variable embeddedTemplatesFolder
// === KEEP THE COMMENT BELOW
//
//go:embed *.gohtml
var embeddedTemplatesFolder embed.FS

// statsHTMLTableHandler returns the usage statistics as html table
// 1. sets required http headers needed for CORS
// 2a for the preflight OPTIONS request, return the CORS header and OK.
// otherwise:
// 2b. start timer
// 3. update ArcStats
// 4. render the output via HtmlTableTmpl
func statsHTMLTableHandler(w http.ResponseWriter, r *http.Request) {

	log.Debug().Msg("received statsTable request")

	// handle the CORS stuff
	SetCORSHeaders(&w, r)

	//2a. Check if it's an OPTIONS request (preflight)
	if r.Method == "OPTIONS" {
		// No further action beyond setting headers is required for the preflight request
		w.WriteHeader(http.StatusOK)
		return
	} else {

		// 2b. set timer
		var startOfProcessing = time.Now()

		// 3. get ArcStats (hopefully from cache)
		domain.ArcStats = domain.Stats4AllSites()

		// remember how long it took to update statistics
		domain.ArcStats.HowLongDidItTake = strconv.FormatInt(time.Since(startOfProcessing).Milliseconds(), 10)

		// find out where this service is running
		domain.ArcStats.FlyRegion, domain.ArcStats.WhereDoesItRun = fly.RegionAndLocation()
		domain.ArcStats.DeploymentID = fly.DeploymentID()

		// 4. store request params in database
		// TODO: include real IP address
		go database.SaveInvocationParams(r.Host, r.RequestURI)
		// 5. finally, render the template
		executeTemplate(w, filepath.Join(TemplatesDir, HtmlTableTmpl), domain.ArcStats)
	}
}

// tilesHandler returns the dashboard tiles: one per property, ordered so the
// eye lands on work (hubs first, then by untriaged count).
// It reuses the same cached statistics as the table, so asking for both costs
// one collection run, not two.
func tilesHandler(w http.ResponseWriter, r *http.Request) {

	log.Debug().Msg("received tiles request")

	SetCORSHeaders(&w, r)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	domain.ArcStats = domain.Stats4AllSites()

	go database.SaveInvocationParams(r.Host, r.RequestURI)

	executeTemplate(w, filepath.Join(TemplatesDir, TilesTmpl), types.TilesData{
		Tiles:             domain.TilesInDisplayOrder(domain.ArcStats),
		LastUpdatedString: domain.ArcStats.LastUpdatedString,
	})
}

// siteDetailHandler returns the repository detail for one property: the full
// open lists a tile can only summarise, and what closed recently. Asked for as
// /siteDetail?site=<key> by the page at /site/<key>/.
func siteDetailHandler(w http.ResponseWriter, r *http.Request) {
	servePropertyFragment(w, r, SiteDetailTmpl)
}

// siteTrafficHandler returns the six Plausible figures for one property, for
// the Traffic section of the same page. It is a second request rather than one
// combined fragment so that each block lands under the heading it belongs to,
// and so that each can fail on its own where the reader is looking.
func siteTrafficHandler(w http.ResponseWriter, r *http.Request) {
	servePropertyFragment(w, r, SiteTrafficTmpl)
}

// siteAvailabilityHandler returns the Availability section for one
// property: current state, the 30-day strip, the three windows, and
// recent incidents. Same fragment-per-heading contract as siteDetail and
// siteTraffic.
func siteAvailabilityHandler(w http.ResponseWriter, r *http.Request) {
	servePropertyFragment(w, r, SiteAvailabilityTmpl)
}

// rollupPageHandler renders the maintainers-only rollup page (ADR-0022): the
// rollup's unique counts beside the sum of its members' own dashboards, which
// properties report into it since when, and the embedded dashboard.
//
// It runs behind auth.RequirePush, which has set the private headers and put
// the visitor's GitHub login into the request context. Unlike every fragment
// it sets no CORS headers: no other origin may read this page. It reads the
// same cached collection run as every other handler.
func rollupPageHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domain.ArcStats = domain.Stats4AllSites()

	go database.SaveInvocationParams(r.Host, r.RequestURI)

	executeTemplate(w, filepath.Join(TemplatesDir, RollupPageTmpl), types.RollupPageData{
		Rollup:            domain.ArcStats.Rollup,
		LastUpdatedString: domain.ArcStats.LastUpdatedString,
		Login:             auth.LoginFrom(r.Context()),
		SiteBaseURL:       env.SiteBaseURL(env.GetEnv()),
		ShareURL:          strings.TrimSpace(os.Getenv("PLAUSIBLE_ROLLUP_SHARE_URL")),
	})
}

// servePropertyFragment renders one template for one property.
//
// It serves the same cached statistics as the table and the tiles, so a visitor
// opening a subpage costs no extra call to GitHub or Plausible - and the two
// fragments a subpage asks for cost one collection run between them, not two.
func servePropertyFragment(w http.ResponseWriter, r *http.Request, templateName string) {

	log.Debug().Msgf("received %s request", templateName)

	SetCORSHeaders(&w, r)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	key := r.URL.Query().Get("site")

	// The key is checked against the declared property list before anything
	// else happens. An unknown key is a 404 and never reaches a template, so
	// no caller-supplied string is ever rendered back into the page.
	if _, known := domain.PropertyByKey(key); !known {
		log.Warn().Msgf("%s asked for unknown property %q", templateName, key)
		http.Error(w, "unknown arc42 property", http.StatusNotFound)
		return
	}

	domain.ArcStats = domain.Stats4AllSites()

	stats, found := domain.StatsForKey(domain.ArcStats, key)
	if !found {
		// A declared property with no collected statistics means the
		// collection run has not produced this entry - a service bug, not a
		// bad request, and worth a different status code than the 404 above.
		log.Error().Msgf("no collected statistics for declared property %q", key)
		http.Error(w, "no statistics collected for this property", http.StatusInternalServerError)
		return
	}

	go database.SaveInvocationParams(r.Host, r.RequestURI)

	executeTemplate(w, filepath.Join(TemplatesDir, templateName), types.SiteDetailData{
		Site:              stats,
		LastUpdatedString: domain.ArcStats.LastUpdatedString,
	})
}

// pingHandler returns a message and the time
func pingHandler(w http.ResponseWriter, r *http.Request) {

	// need to set specific headers, depending on request origin
	SetCORSHeaders(&w, r)

	var Host string = r.Host
	var RequestURI string = r.RequestURI

	log.Debug().Msgf("Host = %s\n", Host)
	log.Debug().Msgf("RequestURI = %s\n", RequestURI)
	executeTemplate(w, filepath.Join(TemplatesDir, PingTmpl), r)
}

// SetCORSHeaders sets specific headers
// * calls from the "official" URL status.arc42.org are allowed
// * calls from localhost or "null" are also allowed
func SetCORSHeaders(w *http.ResponseWriter, r *http.Request) {

	// TODO: why do we use * here?

	var origin = r.Host

	log.Debug().Msgf("received request from host: %s", origin)

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
		log.Error().Msgf("Error parsing template: %v", err)
		http.Error(w, "There was an error parsing the template "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = tpl.Execute(w, data)
	if err != nil {
		log.Error().Msgf("Error executing template: %v", err)
		http.Error(w, "There was an error executing the template "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func logRequestHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Info().Msgf("%s %s %v", r.Method, auth.RedactedURL(r.URL), time.Since(start))
	})
}

// LogServerDetails displays a few details about this program,
// mainly to give admins some idea what version is currently running
// and where in the fly.io cloud the service is deployed.
func LogServerDetails(appVersion string) {

	log.Info().Msgf("Starting API server, version %s on Port %s at %s", appVersion, getPort(), time.Now().Format("2. January 2006, 15:04h"))

	// assumes we're running this program within the fly.io cloud.
	// There, the env variable FLY_REGION should be set.
	// If this variable is empty, we assume we're running locally
	region, location := fly.RegionAndLocation()
	log.Info().Msgf("Server region is%s %s", region, location)
}

func probeHandler(w http.ResponseWriter, r *http.Request) {
	SetCORSHeaders(&w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	secretKey := strings.TrimSpace(os.Getenv("PROBE_SECRET_KEY"))
	if secretKey != "" {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		queryKey := strings.TrimSpace(r.URL.Query().Get("key"))
		expectedBearer := "Bearer " + secretKey

		if authHeader != expectedBearer && authHeader != secretKey && queryKey != secretKey {
			log.Warn().Msg("probe API request unauthorized")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	log.Info().Msg("received external probe request")
	db := database.GetDB()
	start := time.Now()

	recorded, total, err := probe.RunAll(db, "cron-job.org")
	if err != nil {
		log.Error().Err(err).Msg("probe execution failed")
		http.Error(w, fmt.Sprintf("Probe failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Warm the Plausible & GitHub statistics cache in the background so visitors get instant (< 10ms) responses
	go domain.Stats4AllSites()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"recorded_sites": recorded,
		"total_sites":    total,
		"duration_ms":    time.Since(start).Milliseconds(),
	}); err != nil {
		// Status and headers are already on the wire, so there is no error
		// response left to send -- the probe itself succeeded and has been
		// recorded either way. Log it so a caller reporting a truncated body
		// has something to match against.
		log.Error().Err(err).Msg("encoding probe response failed")
	}
}

// StartAPIServer creates http ServeMux with a few predefined routes.
func StartAPIServer() {

	mux := http.NewServeMux()

	// define some routes
	mux.HandleFunc("/statsTable", statsHTMLTableHandler)
	mux.HandleFunc("/statistics", statsHTMLTableHandler)
	mux.HandleFunc("/stats", statsHTMLTableHandler)
	mux.HandleFunc("/tiles", tilesHandler)
	mux.HandleFunc("/siteDetail", siteDetailHandler)
	mux.HandleFunc("/siteTraffic", siteTrafficHandler)
	mux.HandleFunc("/siteAvailability", siteAvailabilityHandler)

	// the maintainers-only rollup page and its GitHub login (ADR-0022)
	authCfg := auth.FromEnv(env.GetEnv())
	if err := authCfg.Problem(); err != nil {
		log.Warn().Msgf("maintainer login unavailable, /rollup answers 503: %v", err)
	}
	gate := auth.New(authCfg)
	mux.Handle("/rollup", gate.RequirePush(http.HandlerFunc(rollupPageHandler)))
	mux.HandleFunc("/auth/login", gate.Login)
	mux.HandleFunc("/auth/callback", gate.Callback)
	mux.HandleFunc("/auth/logout", gate.Logout)

	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/api/probe", probeHandler)

	// wrap ServeMux with logging
	loggedMux := logRequestHandler(mux)

	// TODO why are we setting HomeIP?
	err := http.ListenAndServe(homeIP+getPort(), loggedMux)

	if err != nil {
		log.Fatal().Msgf("API server failed to start: %v", err)
	}

}
