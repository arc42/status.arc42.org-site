# 3. use Arc42Statistics struct to collect results

Date: 2023-12-01

## Status

Accepted

## Context

Within the `domain` package we need to collect the results of the various API calls (to plausible.io and github.com).

## Decision

We use a complex struct (`types.Arc42Statistics`) to collect these results:

```
type Arc42Statistics struct {
    // some meta info, like AppVersion and time of last update
    
	// some info on the server that collected these results (e.g. fly.io region)
	
	// Stats4Site contains the statistics per site or subdomain
	Stats4Site [len(Arc42sites)]SiteStats

	// Totals contains the sum of all the statistics over all sites
	Totals TotalsForAllSites
}
```

Core of this is `Stats4Site`, which holds the statistics (visitors and pageviews) for all sites:

### `Stats4Site`

```
type SiteStats struct {
	Site           string // site name
	Repo           string // the URL of the GitHub repository
	
	// the following are received from GitHub.com
	NrOfOpenBugs   int    // the number of open bugs in that repo
	NrOfOpenIssues int    // number of open issues
	
	// the following are received from plausible.io
	Visitors7d     string
	Pageviews7d    string
	Visitors30d    string
	Pageviews30d   string
	Visitors12m    string
	Pageviews12m   string
}
```

Each of the three time periods (7D, 30d, 12m) need a distinct call to plausible.io, so these can be called in parallel goroutines.

See ADR-0004 how Goroutines are used.

## Consequences

* The various funcs calling Plausible and GitHub need to return parts of these structs, so we can collect
