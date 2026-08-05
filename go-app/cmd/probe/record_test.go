package main

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"arc42-status/internal/availability"
)

func probeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, stmt := range []string{
		`CREATE TABLE status_snapshot (
			id INTEGER PRIMARY KEY, site VARCHAR(100) NOT NULL,
			monitor_id VARCHAR(64), status VARCHAR(12) NOT NULL,
			changed_at DATETIME NOT NULL, response_ms INTEGER, detail VARCHAR(255))`,
		`CREATE TABLE status_bucket (
			site VARCHAR(100) NOT NULL, bucket_start DATETIME NOT NULL,
			bucket_kind VARCHAR(8) NOT NULL,
			downtime_minutes INTEGER NOT NULL DEFAULT 0,
			outage_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (site, bucket_kind, bucket_start))`,
		`CREATE TABLE probe_run (
			run_at DATETIME NOT NULL, vantage VARCHAR(64) NOT NULL,
			sites_checked INTEGER NOT NULL, duration_ms INTEGER NOT NULL)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

var rnow = time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestFirstRunWritesBaselineTransition(t *testing.T) {
	db := probeTestDB(t)
	r := Result{Site: "arc42.org", State: "up", ResponseMs: 200}
	if err := recordResult(db, r, rnow, time.Time{}, false, "gha"); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM status_snapshot`); n != 1 {
		t.Errorf("snapshot rows = %d, want 1 (baseline)", n)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM status_bucket WHERE downtime_minutes = 0`); n != 1 {
		t.Errorf("bucket rows = %d, want 1 coverage row", n)
	}
}

func TestUnchangedStateWritesNoTransition(t *testing.T) {
	db := probeTestDB(t)
	r := Result{Site: "arc42.org", State: "up", ResponseMs: 200}
	if err := recordResult(db, r, rnow.Add(-15*time.Minute), time.Time{}, false, "gha"); err != nil {
		t.Fatal(err)
	}
	if err := recordResult(db, r, rnow, rnow.Add(-15*time.Minute), true, "gha"); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM status_snapshot`); n != 1 {
		t.Errorf("snapshot rows = %d, want 1 (no change, no row)", n)
	}
}

func TestGoingDownWritesTransitionOutageAndDowntime(t *testing.T) {
	db := probeTestDB(t)
	up := Result{Site: "arc42.org", State: "up", ResponseMs: 200}
	if err := recordResult(db, up, rnow.Add(-15*time.Minute), time.Time{}, false, "gha"); err != nil {
		t.Fatal(err)
	}
	down := Result{Site: "arc42.org", State: "down", Detail: "timeout"}
	if err := recordResult(db, down, rnow, rnow.Add(-15*time.Minute), true, "gha"); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM status_snapshot WHERE status = 'down'`); n != 1 {
		t.Errorf("down transitions = %d, want 1", n)
	}
	var downMin, outages int
	if err := db.QueryRow(`SELECT SUM(downtime_minutes), SUM(outage_count) FROM status_bucket
		WHERE site = 'arc42.org'`).Scan(&downMin, &outages); err != nil {
		t.Fatal(err)
	}
	if downMin != 15 || outages != 1 {
		t.Errorf("downtime=%d outages=%d, want 15/1", downMin, outages)
	}
}

func TestDowntimeAttributionIsCapped(t *testing.T) {
	db := probeTestDB(t)
	up := Result{Site: "arc42.org", State: "up"}
	if err := recordResult(db, up, rnow.Add(-3*time.Hour), time.Time{}, false, "gha"); err != nil {
		t.Fatal(err)
	}
	down := Result{Site: "arc42.org", State: "down", Detail: "502"}
	// last run three hours ago (delayed schedule): cap applies
	if err := recordResult(db, down, rnow, rnow.Add(-3*time.Hour), true, "gha"); err != nil {
		t.Fatal(err)
	}
	var downMin int
	if err := db.QueryRow(`SELECT SUM(downtime_minutes) FROM status_bucket
		WHERE site = 'arc42.org'`).Scan(&downMin); err != nil {
		t.Fatal(err)
	}
	if downMin != availability.MaxDowntimePerRunMinutes {
		t.Errorf("downtime=%d, want capped %d", downMin, availability.MaxDowntimePerRunMinutes)
	}
}
