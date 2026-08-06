package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"arc42-status/internal/availability"
	"arc42-status/internal/database"
	"arc42-status/internal/slack"
	"arc42-status/internal/types"
)

// recordResult persists one confirmed measurement: a transition row when
// the state changed (or was never recorded), downtime attributed to
// today's bucket when the site is down, and always a coverage upsert so
// an all-up day is a row with zero downtime - a different fact from no
// row at all.
func recordResult(db *sql.DB, r Result, now, lastRunAt time.Time, haveLastRun bool, vantage string) error {
	prevState, _, found, err := availability.LastState(db, r.Site)
	if err != nil {
		return err
	}

	if !found || prevState != r.State {
		if err := availability.WriteTransition(db, r.Site, r.State, now, r.ResponseMs, r.Detail, vantage); err != nil {
			return err
		}
	}

	downMinutes := 0
	if r.State == "down" {
		downMinutes = availability.CycleMinutes
		if haveLastRun {
			since := int(now.Sub(lastRunAt).Minutes())
			if since > 0 {
				downMinutes = since
			}
		}
		if downMinutes > availability.MaxDowntimePerRunMinutes {
			downMinutes = availability.MaxDowntimePerRunMinutes
		}
	}
	outages := 0
	if r.State == "down" && found && prevState != "down" {
		outages = 1
	}
	if r.State == "down" && !found {
		outages = 1
	}

	day := now.UTC().Truncate(24 * time.Hour)
	return availability.UpsertDailyBucket(db, r.Site, day, downMinutes, outages)
}

func main() {
	start := time.Now().UTC()

	vantage := os.Getenv("PROBE_VANTAGE")
	if vantage == "" {
		vantage = "gha"
	}

	db := database.GetDB()
	if err := db.Ping(); err != nil {
		// the one failure that is the prober's own: nothing can be
		// recorded, so the run must fail loudly in the Actions log
		log.Fatal().Msgf("probe: database unreachable: %v", err)
	}

	lastRunAt, haveLastRun, err := availability.ReadLastRun(db)
	if err != nil {
		log.Fatal().Msgf("probe: cannot read last run: %v", err)
	}

	client := &http.Client{Timeout: RequestTimeoutSeconds * time.Second}

	var monitored []types.Property
	for _, p := range types.Arc42properties {
		if types.Monitored(p) {
			monitored = append(monitored, p)
		}
	}

	// measure concurrently - the showpiece pattern of this repo - then
	// record serially: SQLite/libSQL writes do not benefit from racing.
	results := make([]Result, len(monitored))
	var wg sync.WaitGroup
	for i, p := range monitored {
		wg.Add(1)
		go func(i int, p types.Property) {
			defer wg.Done()
			results[i] = ProbeSite(client, p)
		}(i, p)
	}
	wg.Wait()

	now := time.Now().UTC()
	recorded := 0
	for _, r := range results {
		if err := recordResult(db, r, now, lastRunAt, haveLastRun, vantage); err != nil {
			log.Error().Msgf("probe: recording %s: %v", r.Site, err)
			continue
		}
		recorded++
		log.Info().Msgf("probe: %s is %s %s", r.Site, r.State, r.Detail)
		if r.State == "down" {
			msg := fmt.Sprintf("Alert: availability check failed for %s (state: %s, detail: %s)", r.Site, r.State, r.Detail)
			slack.SendSlackMessage(msg)
		}
	}

	if err := availability.WriteProbeRun(db, now, vantage, recorded,
		int(time.Since(start).Milliseconds())); err != nil {
		log.Fatal().Msgf("probe: writing heartbeat: %v", err)
	}
	if recorded == 0 {
		log.Fatal().Msg("probe: no site could be recorded")
	}
	log.Info().Msgf("probe: recorded %d/%d sites in %dms", recorded, len(monitored),
		time.Since(start).Milliseconds())
	// down sites exit 0: a measured outage is a successful measurement
}
