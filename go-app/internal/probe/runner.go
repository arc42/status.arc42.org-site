package probe

import (
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"arc42-status/internal/availability"
	"arc42-status/internal/slack"
	"arc42-status/internal/types"
)

// recordResult persists one confirmed measurement: a transition row when
// the state changed (or was never recorded), downtime attributed to
// today's bucket when the site is down, and always a coverage upsert so
// an all-up day is a row with zero downtime.
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

// RunAll measures all monitored properties concurrently, records outcomes in Turso,
// sends Slack alerts on failure, and writes the heartbeat probe_run row.
func RunAll(db *sql.DB, vantage string) (int, int, error) {
	start := time.Now().UTC()

	if vantage == "" {
		vantage = "cron-job.org"
	}

	if err := db.Ping(); err != nil {
		return 0, 0, fmt.Errorf("database unreachable: %w", err)
	}

	lastRunAt, haveLastRun, err := availability.ReadLastRun(db)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read last run: %w", err)
	}

	client := &http.Client{Timeout: RequestTimeoutSeconds * time.Second}

	var monitored []types.Property
	for _, p := range types.Arc42properties {
		if types.Monitored(p) {
			monitored = append(monitored, p)
		}
	}

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

	durationMs := int(time.Since(start).Milliseconds())
	if err := availability.WriteProbeRun(db, now, vantage, recorded, durationMs); err != nil {
		return recorded, len(monitored), fmt.Errorf("writing heartbeat: %w", err)
	}

	log.Info().Msgf("probe: recorded %d/%d sites in %dms", recorded, len(monitored), durationMs)
	return recorded, len(monitored), nil
}
