package domain

import (
	"arc42-status/internal/availability"
	"arc42-status/internal/database"
	"arc42-status/internal/github"
	"arc42-status/internal/plausible"
	"arc42-status/internal/types"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
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

	var Stats4Sites = make([]types.SiteStatsType, len(types.Arc42properties))
	var Stats4Repos = make([]types.RepoStatsType, len(types.Arc42properties))

	// 1.) set meta info
	setServerMetaInfo(&a42s)

	// retrieve usage statistics (visitors and pageviews)
	for index, property := range types.Arc42properties {
		wg.Add(1)

		go getUsageStatisticsForSite(property, &Stats4Sites[index], &wg)
	}

	// retrieve repo statistics
	// currently:  number of open bugs and issues from GitHub
	for index, property := range types.Arc42properties {
		wg.Add(1)

		go getRepoStatisticsForSite(property, &Stats4Repos[index], &wg)
	}

	wg.Wait()

	// get results from Goroutines
	log.Debug().Msgf("transferring results into LoadStats4Site")
	for index, property := range types.Arc42properties {
		a42s.Stats4Site[index] = Stats4Sites[index]
		a42s.Stats4Site[index].NrOfOpenIssues = Stats4Repos[index].NrOfOpenIssues
		a42s.Stats4Site[index].NrOfOpenBugs = Stats4Repos[index].NrOfOpenBugs
		a42s.Stats4Site[index].NrOfOpenPRs = Stats4Repos[index].NrOfPRs
		a42s.Stats4Site[index].Repo = Stats4Repos[index].Repo
		a42s.Stats4Site[index].OpenItems = Stats4Repos[index].OpenItems
		a42s.Stats4Site[index].RecentlyClosed = Stats4Repos[index].RecentlyClosed
		a42s.Stats4Site[index].NrUntriaged = Stats4Repos[index].NrUntriaged
		a42s.Stats4Site[index].IsHub = property.IsHub
		a42s.Stats4Site[index].Planned = property.Planned

		log.Debug().Msgf("Repo %s has %d issues, %d bugs, and %d PRs", Stats4Repos[index].Repo, Stats4Repos[index].NrOfOpenIssues, Stats4Repos[index].NrOfOpenBugs, Stats4Repos[index].NrOfPRs)
	}

	// availability is read from our own Turso tables, not from an
	// external API - one sequential read pass, cached with everything
	// else, so a page view costs one burst (ADR-0019).
	avail := availability.ForAllSites(database.GetDB(), time.Now().UTC())
	for index, property := range types.Arc42properties {
		if av, ok := avail[property.Key]; ok {
			a42s.Stats4Site[index].Availability = av
		}
	}
	a42s.Availability = availability.Family(avail)

	// now calculate totals
	a42s.Totals = calculateTotals(a42s.Stats4Site)

	return a42s
}

// calculateTotals sums the traffic over the rows the table actually shows, and
// the repository counts over every property.
//
// The two ranges differ on purpose. A totals row under a seven-row table that
// silently counts eleven properties is a number the reader cannot check by
// adding up the column - so traffic is summed over InTable rows only. The
// repository totals appear in no such column; they describe the whole family
// and are summed over all of it.
func calculateTotals(stats [len(types.Arc42properties)]types.SiteStatsType) types.TotalsForAllSites {
	var totals types.TotalsForAllSites

	for index := range stats {
		if stats[index].InTable {
			totals.SumOfVisitors7dNr += stats[index].Visitors7dNr
			totals.SumOfPageViews7dNr += stats[index].PageViews7dNr
			totals.SumOfVisitors30dNr += stats[index].Visitors30dNr
			totals.SumOfPageViews30dNr += stats[index].PageViews30dNr
			totals.SumOfVisitors12mNr += stats[index].Visitors12mNr
			totals.SumOfPageViews12mNr += stats[index].PageViews12mNr
		}
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

// getUsageStatisticsForSite retrieves the statistics for a single site from plausible.io.
// This func is called as Goroutine.
func getUsageStatisticsForSite(property types.Property, thisSiteStats *types.SiteStatsType, wg *sync.WaitGroup) {
	defer wg.Done()

	// to avoid repeating the expression, introduce local var
	thisSiteStats.Site = property.Key
	thisSiteStats.Host = property.Host
	thisSiteStats.InTable = property.InTable
	thisSiteStats.HasTraffic = property.HasTraffic
	thisSiteStats.Planned = property.Planned

	if !property.HasTraffic {
		// no site_id, no query, no numbers - and deliberately no error:
		// "not measured" is a fact about the site, not a failure of this run.
		plausible.MarkUnmeasured(thisSiteStats)
		log.Debug().Msgf("%s has no Plausible site, visitor numbers are %s", property.Key, types.NotAvailable)
		return
	}

	// get statistic data from plausible.io
	plausible.StatsForSite(property.Host, thisSiteStats)

}

// TilesInDisplayOrder returns the properties in the order the grid shows them,
// which is the order types.Arc42properties declares (see the row plan there).
//
// It used to be TilesInAttentionOrder and sorted: hubs first, then by untriaged
// count descending. That made the grid rearrange itself whenever somebody
// labelled an issue, so the page had no shape a reader could learn - and it
// buried the family's own ordering under a transient one. Attention is carried
// by the untriaged count in each tile instead, which is where it belongs.
func TilesInDisplayOrder(a42s types.Arc42Statistics) []types.SiteStatsType {
	tiles := make([]types.SiteStatsType, 0, len(a42s.Stats4Site))
	return append(tiles, a42s.Stats4Site[:]...)
}

// PropertyByKey looks up one property. The second return value is false for an
// unknown key, which is how the siteDetail endpoint rejects a made-up query
// parameter without ever putting it into a request to GitHub.
func PropertyByKey(key string) (types.Property, bool) {
	for _, property := range types.Arc42properties {
		if property.Key == key {
			return property, true
		}
	}
	return types.Property{}, false
}

// StatsForKey returns the collected statistics for one property.
func StatsForKey(a42s types.Arc42Statistics, key string) (types.SiteStatsType, bool) {
	for _, s := range a42s.Stats4Site {
		if s.Site == key {
			return s, true
		}
	}
	return types.SiteStatsType{}, false
}

// RepoNameForSite maps a property key to the GitHub repository behind it.
//
// The relation used to be a rule ("<host>-site") plus a table of exceptions,
// written down here and, inverted, in getRepoStatisticsForSite. It is now a
// plain column in types.Arc42properties: with arc42-template on the dashboard
// there is a property whose key is not a host at all, and a naming rule cannot
// describe that without inventing a host to derive it from.
func RepoNameForSite(key string) string {
	if property, known := PropertyByKey(key); known {
		return property.Repo
	}
	return ""
}

// getRepoStatisticsForSite collects everything the dashboard knows about the
// repository behind a property: the open counts, the open items, and what
// closed most recently. This func is called as Goroutine.
func getRepoStatisticsForSite(property types.Property, thisRepoStats *types.RepoStatsType, wg *sync.WaitGroup) {
	defer wg.Done()

	thisRepoStats.Site = property.Key

	if property.Planned {
		// Nothing to ask and nobody to ask it of. Querying GitHub for a
		// repository that does not exist would cost three failed calls per
		// collection run and three log lines saying so - about a state that
		// is not a failure.
		log.Debug().Msgf("%s is planned, not built: no repository query", property.Key)
		return
	}

	thisRepoStats.Repo = github.GithubArc42URL + property.Repo

	github.StatsForRepo(property.Repo, thisRepoStats)
	github.OpenItemsForRepo(property.Repo, thisRepoStats)
	github.RecentlyClosedForRepo(property.Repo, thisRepoStats)

}
