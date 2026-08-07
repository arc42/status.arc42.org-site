package probe

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"arc42-status/internal/availability"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	schema := `
	CREATE TABLE status_snapshot (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		site TEXT NOT NULL,
		status TEXT NOT NULL,
		changed_at TIMESTAMP NOT NULL,
		response_ms INTEGER,
		detail TEXT,
		monitor_id TEXT
	);
	CREATE TABLE status_bucket (
		site TEXT NOT NULL,
		bucket_start TIMESTAMP NOT NULL,
		bucket_kind TEXT NOT NULL DEFAULT 'day',
		downtime_minutes INTEGER DEFAULT 0,
		outage_count INTEGER DEFAULT 0,
		PRIMARY KEY (site, bucket_kind, bucket_start)
	);
	CREATE TABLE probe_run (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_at TIMESTAMP NOT NULL,
		vantage TEXT NOT NULL,
		sites_checked INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestRecordResultTransitions(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	t0 := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

	// First run: site goes up. Must write transition + bucket
	r1 := Result{Site: "arc42.org", State: "up", ResponseMs: 120}
	if err := recordResult(db, r1, t0, time.Time{}, false, "gha"); err != nil {
		t.Fatalf("record initial up: %v", err)
	}

	state, _, found, err := availability.LastState(db, "arc42.org")
	if err != nil || !found || state != "up" {
		t.Errorf("want state=up, got state=%s, found=%v, err=%v", state, found, err)
	}

	// Second run 15 min later: still up. Must NOT add a transition row
	t1 := t0.Add(15 * time.Minute)
	r2 := Result{Site: "arc42.org", State: "up", ResponseMs: 110}
	if err := recordResult(db, r2, t1, t0, true, "gha"); err != nil {
		t.Fatalf("record second up: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM status_snapshot WHERE site = 'arc42.org'").Scan(&count)
	if count != 1 {
		t.Errorf("want 1 transition row, got %d", count)
	}

	// Third run 15 min later: site goes down. Must add transition + downtime
	t2 := t1.Add(15 * time.Minute)
	r3 := Result{Site: "arc42.org", State: "down", Detail: "502", ResponseMs: 40}
	if err := recordResult(db, r3, t2, t1, true, "gha"); err != nil {
		t.Fatalf("record down: %v", err)
	}

	db.QueryRow("SELECT COUNT(*) FROM status_snapshot WHERE site = 'arc42.org'").Scan(&count)
	if count != 2 {
		t.Errorf("want 2 transition rows, got %d", count)
	}

	var downMinutes, outages int
	day := t0.Truncate(24 * time.Hour)
	err = db.QueryRow("SELECT downtime_minutes, outage_count FROM status_bucket WHERE site = 'arc42.org' AND bucket_start = ?", day.Format("2006-01-02 15:04:05")).Scan(&downMinutes, &outages)
	if err != nil {
		t.Fatalf("read bucket: %v", err)
	}
	if downMinutes != 15 || outages != 1 {
		t.Errorf("want downMinutes=15, outages=1, got downMinutes=%d, outages=%d", downMinutes, outages)
	}
}
