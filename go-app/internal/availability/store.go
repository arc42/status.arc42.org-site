package availability

import (
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"

	"arc42-status/internal/database"
	"arc42-status/internal/types"
)

// The prober writes through these functions and the service reads through
// them; nothing else touches the three status tables. All take *sql.DB so
// tests run against in-memory SQLite; callers pass database.GetDB().
//
// Every DATETIME column is read as CAST(... AS TEXT). Times are stored as
// text in database.DateTimeLayout, and that is what libsql hands back in
// production - but mattn/go-sqlite3 (the in-memory test driver, and the
// local dev driver) converts columns *declared* DATETIME into time.Time,
// which then arrives here as RFC3339 and fails to parse. The cast is a
// no-op on the stored text and keeps one parse path, the production one,
// under both drivers. Comparisons stay on the raw column so the indexes
// are still used.

func LastState(db *sql.DB, site string) (string, time.Time, bool, error) {
	var status, at string
	err := db.QueryRow(
		`SELECT status, CAST(changed_at AS TEXT) FROM status_snapshot
		 WHERE site = ? ORDER BY changed_at DESC, id DESC LIMIT 1`,
		site).Scan(&status, &at)
	if err == sql.ErrNoRows {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, err
	}
	t, err := time.ParseInLocation(database.DateTimeLayout, at, time.UTC)
	return status, t, err == nil, err
}

func WriteTransition(db *sql.DB, site, status string, at time.Time, responseMs int, detail, vantage string) error {
	_, err := db.Exec(
		`INSERT INTO status_snapshot (site, monitor_id, status, changed_at, response_ms, detail)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		site, vantage, status, at.UTC().Format(database.DateTimeLayout), responseMs, detail)
	return err
}

func UpsertDailyBucket(db *sql.DB, site string, day time.Time, addDownMinutes, addOutages int) error {
	_, err := db.Exec(
		`INSERT INTO status_bucket (site, bucket_start, bucket_kind, downtime_minutes, outage_count)
		 VALUES (?, ?, 'day', ?, ?)
		 ON CONFLICT (site, bucket_kind, bucket_start) DO UPDATE SET
		   downtime_minutes = downtime_minutes + excluded.downtime_minutes,
		   outage_count     = outage_count + excluded.outage_count`,
		site, day.UTC().Format(database.DateTimeLayout), addDownMinutes, addOutages)
	return err
}

func WriteProbeRun(db *sql.DB, at time.Time, vantage string, sitesChecked, durationMs int) error {
	_, err := db.Exec(
		`INSERT INTO probe_run (run_at, vantage, sites_checked, duration_ms)
		 VALUES (?, ?, ?, ?)`,
		at.UTC().Format(database.DateTimeLayout), vantage, sitesChecked, durationMs)
	return err
}

func ReadLastRun(db *sql.DB) (time.Time, bool, error) {
	var at string
	err := db.QueryRow(`SELECT CAST(run_at AS TEXT) FROM probe_run
		 ORDER BY run_at DESC LIMIT 1`).Scan(&at)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	t, err := time.ParseInLocation(database.DateTimeLayout, at, time.UTC)
	return t, err == nil, err
}

func readBuckets(db *sql.DB, site string, since time.Time) ([]BucketDay, error) {
	rows, err := db.Query(
		`SELECT CAST(bucket_start AS TEXT), downtime_minutes, outage_count FROM status_bucket
		 WHERE site = ? AND bucket_kind = 'day' AND bucket_start >= ?
		 ORDER BY bucket_start ASC`,
		site, since.UTC().Format(database.DateTimeLayout))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BucketDay, 0)
	for rows.Next() {
		var day string
		var b BucketDay
		if err := rows.Scan(&day, &b.DowntimeMinutes, &b.OutageCount); err != nil {
			return nil, err
		}
		if b.Day, err = time.ParseInLocation(database.DateTimeLayout, day, time.UTC); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func readSnapshots(db *sql.DB, site string, limit int) ([]SnapshotRow, error) {
	rows, err := db.Query(
		`SELECT status, CAST(changed_at AS TEXT), COALESCE(response_ms, 0), COALESCE(detail, '')
		 FROM status_snapshot WHERE site = ?
		 ORDER BY changed_at DESC, id DESC LIMIT ?`,
		site, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SnapshotRow, 0, limit)
	for rows.Next() {
		var at string
		var r SnapshotRow
		if err := rows.Scan(&r.Status, &at, &r.ResponseMs, &r.Detail); err != nil {
			return nil, err
		}
		if r.ChangedAt, err = time.ParseInLocation(database.DateTimeLayout, at, time.UTC); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ForSite assembles everything the surfaces show for one property. A read
// error degrades to "unmeasured" with a log line rather than failing the
// whole page: the statistics around it are still worth serving.
func ForSite(db *sql.DB, site string, now time.Time) types.SiteAvailability {
	av := types.SiteAvailability{}

	status, _, found, err := LastState(db, site)
	if err != nil {
		log.Error().Msgf("availability: reading last state for %s: %v", site, err)
		return av
	}
	if !found {
		return av // never probed: Measured stays false
	}
	av.Measured = true
	av.State = status

	if lastRun, runFound, err := ReadLastRun(db); err == nil && runFound {
		av.LastCheckedAgo = Ago(lastRun, now)
		av.Stale = now.Sub(lastRun) > StaleAfter
	} else {
		// transitions without any heartbeat: treat as stale, the
		// prober's bookkeeping is broken
		av.Stale = true
	}

	yearAgo := utcDay(now).AddDate(0, -12, 0)
	buckets, err := readBuckets(db, site, yearAgo)
	if err != nil {
		log.Error().Msgf("availability: reading buckets for %s: %v", site, err)
	}
	av.Uptime7d, _ = UptimeWindow(buckets, 7, now)
	av.Uptime30d, av.Uptime30dNr = UptimeWindow(buckets, 30, now)
	av.Uptime12m, _ = UptimeWindow(buckets, 365, now)
	av.MeasuredSince = MeasuredSince(buckets, now)
	av.Days = StripCells(buckets, now)

	snaps, err := readSnapshots(db, site, 20)
	if err != nil {
		log.Error().Msgf("availability: reading snapshots for %s: %v", site, err)
	}
	av.Incidents = Incidents(snaps, now)

	return av
}

// ForAllSites reads availability for every monitored property. One
// sequential pass: at most a few dozen small indexed queries against one
// connection, and the result lands in the same cache as the statistics.
func ForAllSites(db *sql.DB, now time.Time) map[string]types.SiteAvailability {
	out := make(map[string]types.SiteAvailability)
	for _, p := range types.Arc42properties {
		if types.Monitored(p) {
			out[p.Key] = ForSite(db, p.Key, now)
		}
	}
	return out
}

// Family folds the per-site readings into the verdict line above the
// table.
func Family(m map[string]types.SiteAvailability) types.FamilyAvailability {
	f := types.FamilyAvailability{}
	for _, av := range m {
		if !av.Measured {
			continue
		}
		f.Measured = true
		f.NrMonitored++
		if av.Stale {
			f.Stale = true
		}
		switch av.State {
		case "down":
			f.NrDown++
		case "degraded":
			f.NrDegraded++
		}
		if av.LastCheckedAgo != "" {
			f.LastCheckedAgo = av.LastCheckedAgo
		}
	}
	f.AllUp = f.Measured && f.NrDown == 0 && f.NrDegraded == 0
	return f
}
