package domain

import (
	"arc42-status/internal/github"
	"arc42-status/internal/plausible"
	"arc42-status/internal/types"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"sort"
	"sync"
	"time"
	"zgo.at/zcache/v2"
)

var AppVersion string

// ArcStats collects all data
var ArcStats types.Arc42Statistics

// cache expiration should be 5 or 10 minutes
// for testing, set expiration to a few seconds only
const cacheExpirationTime = time.Second * 100

// cacheStatsKey is the key under which the results are stored in the cache
const cacheStatsKey = "arc42Stats"

// create a cache with a default expiration time of 5 minutes, which
// purges expired items every 5 minutes
var cache = zcache.New[string, types.Arc42Statistics](cacheExpirationTime, cacheExpirationTime)

func SetAppVersion(appVersion string) {
	AppVersion = appVersion
	log.Debug().Msg("App version set to " + appVersion)
}

func GetAppVersion() string {
	return AppVersion
}

func setServerMetaInfo(a42s *types.Arc42Statistics) {
	a42s.AppVersion = GetAppVersion()

	location, _ := time.LoadLocation("Europe/Berlin")

	// Get the current time in Bielefeld, the town that presumably does not exist
	bielefeldTime := time.Now().In(location)

	a42s.LastUpdated = bielefeldTime
	a42s.LastUpdatedString = bielefeldTime.Format("2. January 2006, 15:04:03h")
}

// Stats4AllSites tries to return the value from the cache instead of calling
// the external APIs.
// If the value is expired, then new data is loaded.
// If it is still available, the existing value is returned
func Stats4AllSites() types.Arc42Statistics {

	var a42s, found = cache.Get(cacheStatsKey)

	// if not found, LoadStats4AllSites() again
	if !found {
		log.Info().Msg("cache miss, data expired")
		a42s = LoadStats4AllSites()
		cache.Set(cacheStatsKey, a42s)
	} else {
		log.Info().Msg("cache hit, data still valid")
		a42s.HowLongDidItTake = "0 msec (cached result)"
	}
	return a42s
}

// LoadStats4AllSites retrieves the statistics for all sites from plausible.io and GitHub repositories.
func LoadStats4AllSites() types.Arc42Statistics {

	// the WaitGroup synchronises the parallel goroutines
	var wg sync.WaitGroup

	var a42s = types.Arc42Statistics{}

	var Stats4Sites = make([]types.SiteStatsType, len(types.Arc42sites))
	var Stats4Repos = make([]types.RepoStatsType, len(types.Arc42sites))

	// 1.) set meta info
	setServerMetaInfo(&a42s)

	// retrieve usage statistics (visitors and pageviews)
	for index, site := range types.Arc42sites {
		wg.Add(1)

		go getUsageStatisticsForSite(site, &Stats4Sites[index], &wg)
	}

	// retrieve repo statistics
	// currently:  number of open bugs and issues from GitHub
	for index, site := range types.Arc42sites {
		wg.Add(1)

		go getRepoStatisticsForSite(site, &Stats4Repos[index], &wg)
	}

	wg.Wait()

	// get results from Goroutines
	log.Debug().Msgf("transferring results into LoadStats4Site")
	for index := range types.Arc42sites {
		a42s.Stats4Site[index] = Stats4Sites[index]
		a42s.Stats4Site[index].NrOfOpenIssues = Stats4Repos[index].NrOfOpenIssues
		a42s.Stats4Site[index].NrOfOpenBugs = Stats4Repos[index].NrOfOpenBugs
		a42s.Stats4Site[index].NrOfOpenPRs = Stats4Repos[index].NrOfPRs
		a42s.Stats4Site[index].Repo = Stats4Repos[index].Repo
		a42s.Stats4Site[index].OpenItems = Stats4Repos[index].OpenItems
		a42s.Stats4Site[index].RecentlyClosed = Stats4Repos[index].RecentlyClosed
		a42s.Stats4Site[index].NrUntriaged = Stats4Repos[index].NrUntriaged
		a42s.Stats4Site[index].IsHub = isHub(types.Arc42sites[index])

		log.Debug().Msgf("Repo %s has %d issues, %d bugs, and %d PRs", Stats4Repos[index].Repo, Stats4Repos[index].NrOfOpenIssues, Stats4Repos[index].NrOfOpenBugs, Stats4Repos[index].NrOfPRs)
	}

	// now calculate totals
	a42s.Totals = calculateTotals(a42s.Stats4Site)

	return a42s
}

func calculateTotals(stats [len(types.Arc42sites)]types.SiteStatsType) types.TotalsForAllSites {
	var totals types.TotalsForAllSites

	for index := range types.Arc42sites {
		totals.SumOfVisitors7dNr += stats[index].Visitors7dNr
		totals.SumOfPageViews7dNr += stats[index].PageViews7dNr
		totals.SumOfVisitors30dNr += stats[index].Visitors30dNr
		totals.SumOfPageViews30dNr += stats[index].PageViews30dNr
		totals.SumOfVisitors12mNr += stats[index].Visitors12mNr
		totals.SumOfPageViews12mNr += stats[index].PageViews12mNr
		totals.TotalNrOfIssues += stats[index].NrOfOpenIssues
		totals.TotalNrOfBugs += stats[index].NrOfOpenBugs
		totals.TotalNrOfPRs += stats[index].NrOfOpenPRs
	}

	// now convert numbers to strings-with-separators
	// e.g., 1234 -> 1.234
	p := message.NewPrinter(language.German)

	totals.SumOfVisitors7d = p.Sprintf("%d", totals.SumOfVisitors7dNr)
	totals.SumOfPageViews7d = p.Sprintf("%d", totals.SumOfPageViews7dNr)

	totals.SumOfVisitors30d = p.Sprintf("%d", totals.SumOfVisitors30dNr)
	totals.SumOfPageViews30d = p.Sprintf("%d", totals.SumOfPageViews30dNr)

	totals.SumOfVisitors12m = p.Sprintf("%d", totals.SumOfVisitors12mNr)
	totals.SumOfPageViews12m = p.Sprintf("%d", totals.SumOfPageViews12mNr)

	log.Debug().Msgf("Total visits and pageviews (V/PV, 7d, 30d, 12m)= %d/%d, %d/%d, %d/%d", totals.SumOfVisitors7dNr, totals.SumOfPageViews7dNr, totals.SumOfVisitors30dNr, totals.SumOfPageViews30dNr, totals.SumOfVisitors12mNr, totals.SumOfPageViews12mNr)
	log.Debug().Msgf("Total %d issues, %d bugs, and %d PRs", totals.TotalNrOfIssues, totals.TotalNrOfBugs, totals.TotalNrOfPRs)

	return totals
}

// sitesWithoutPlausible lists the properties that have no Plausible site at all.
// Asking Plausible about them would produce an API error on every single
// collection run - an error that says nothing, because nothing is broken: the
// site simply is not measured. meta.arc42.org is the brand and decision home,
// read by maintainers rather than by an audience, so it was never registered.
var sitesWithoutPlausible = map[string]bool{
	"meta.arc42.org": true,
}

// getUsageStatisticsForSite retrieves the statistics for a single site from plausible.io.
// This func is called as Goroutine.
func getUsageStatisticsForSite(site string, thisSiteStats *types.SiteStatsType, wg *sync.WaitGroup) {
	defer wg.Done()

	// to avoid repeating the expression, introduce local var
	thisSiteStats.Site = site

	if sitesWithoutPlausible[site] {
		// no site_id, no query, no numbers - and deliberately no error:
		// "not measured" is a fact about the site, not a failure of this run.
		plausible.MarkUnmeasured(thisSiteStats)
		log.Debug().Msgf("%s has no Plausible site, visitor numbers are %s", site, types.NotAvailable)
		return
	}

	thisSiteStats.HasTraffic = true

	// get statistic data from plausible.io
	plausible.StatsForSite(site, thisSiteStats)

}

// isHub reports whether a site is one of the two hubs. BRAND.md (ADR-0001)
// defines arc42.org and arc42.de as hub twins sharing the brand navy; every
// other property is a satellite owning one signature hue. The dashboard shows
// hubs first because they are the family's front door.
func isHub(siteName string) bool {
	return siteName == "arc42.org" || siteName == "arc42.de"
}

// TilesInAttentionOrder returns the sites arranged the way the dashboard reads
// them: hubs first, then satellites sorted by untriaged count descending and
// alphabetically within equal counts, so the eye lands on work and the order
// stays stable when nothing needs attention.
func TilesInAttentionOrder(a42s types.Arc42Statistics) []types.SiteStatsType {
	tiles := make([]types.SiteStatsType, 0, len(a42s.Stats4Site))
	tiles = append(tiles, a42s.Stats4Site[:]...)

	sort.SliceStable(tiles, func(i, j int) bool {
		if tiles[i].IsHub != tiles[j].IsHub {
			return tiles[i].IsHub
		}
		if tiles[i].NrUntriaged != tiles[j].NrUntriaged {
			return tiles[i].NrUntriaged > tiles[j].NrUntriaged
		}
		return tiles[i].Site < tiles[j].Site
	})

	return tiles
}

// repoNameExceptions lists the sites whose repository is not simply
// "<host>-site". This map, plus that default rule, is the ONLY place the
// site-to-repository relation is written down.
//
// It used to be written down twice - here and, inverted, in
// getRepoStatisticsForSite, which reconstructed the host from the repository
// name by trimming "-site". Two descriptions of one fact can disagree, and with
// meta.arc42.org they immediately would have: its repository carries no "-site"
// suffix, so trimming would have left the host unchanged by luck rather than by
// design, while PDFminion needed a hand-written special case on both sides.
// The inverse is now gone entirely: the host is carried through, never derived.
var repoNameExceptions = map[string]string{
	"pdfminion.arc42.org": "PDFminion",      // the tool predates its subdomain
	"meta.arc42.org":      "meta.arc42.org", // brand/decision home, no "-site" suffix
}

// RepoNameForSite maps a site host to the GitHub repository behind it.
// Most sites follow the pattern "<host>-site"; the exceptions are listed above.
func RepoNameForSite(siteName string) string {
	if repoName, isException := repoNameExceptions[siteName]; isException {
		return repoName
	}
	return siteName + "-site"
}

// getRepoStatisticsForSite collects everything the dashboard knows about the
// repository behind a site: the open counts, the newest open items, and what
// closed most recently. This func is called as Goroutine.
func getRepoStatisticsForSite(siteName string, thisRepoStats *types.RepoStatsType, wg *sync.WaitGroup) {
	defer wg.Done()

	repoName := RepoNameForSite(siteName)

	thisRepoStats.Site = siteName
	thisRepoStats.Repo = github.GithubArc42URL + repoName

	github.StatsForRepo(repoName, thisRepoStats)
	github.OpenItemsForRepo(repoName, thisRepoStats)
	github.RecentlyClosedForRepo(repoName, thisRepoStats)

}
