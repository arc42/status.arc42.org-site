package availability

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"arc42-status/internal/types"
)

// testDB returns an in-memory database carrying the three status tables,
// DDL mirroring internal/database/schema.hcl (kept in sync manually,
// ADR-0014).
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ddl := []string{
		`CREATE TABLE status_snapshot (
			id INTEGER PRIMARY KEY,
			site VARCHAR(100) NOT NULL,
			monitor_id VARCHAR(64),
			status VARCHAR(12) NOT NULL,
			changed_at DATETIME NOT NULL,
			response_ms INTEGER,
			detail VARCHAR(255))`,
		`CREATE INDEX idx_snapshot_site_changed ON status_snapshot (site, changed_at)`,
		`CREATE TABLE status_bucket (
			site VARCHAR(100) NOT NULL,
			bucket_start DATETIME NOT NULL,
			bucket_kind VARCHAR(8) NOT NULL,
			downtime_minutes INTEGER NOT NULL DEFAULT 0,
			outage_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (site, bucket_kind, bucket_start))`,
		`CREATE TABLE probe_run (
			run_at DATETIME NOT NULL,
			vantage VARCHAR(64) NOT NULL,
			sites_checked INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL)`,
	}
	for _, stmt := range ddl {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

var tnow = time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

func TestTransitionRoundtrip(t *testing.T) {
	db := testDB(t)
	if _, _, found, _ := LastState(db, "arc42.org"); found {
		t.Fatal("empty table must report not-found")
	}
	if err := WriteTransition(db, "arc42.org", "up", tnow.Add(-2*time.Hour), 123, "", "gha"); err != nil {
		t.Fatal(err)
	}
	if err := WriteTransition(db, "arc42.org", "down", tnow.Add(-1*time.Hour), 0, "timeout", "gha"); err != nil {
		t.Fatal(err)
	}
	status, at, found, err := LastState(db, "arc42.org")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if status != "down" || !at.Equal(tnow.Add(-1*time.Hour)) {
		t.Errorf("got %s@%v", status, at)
	}
}

func TestBucketUpsertAccumulates(t *testing.T) {
	db := testDB(t)
	day := tnow.Truncate(24 * time.Hour)
	if err := UpsertDailyBucket(db, "arc42.org", day, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := UpsertDailyBucket(db, "arc42.org", day, 15, 1); err != nil {
		t.Fatal(err)
	}
	if err := UpsertDailyBucket(db, "arc42.org", day, 15, 0); err != nil {
		t.Fatal(err)
	}
	var down, outages int
	err := db.QueryRow(`SELECT downtime_minutes, outage_count FROM status_bucket
		WHERE site = ? AND bucket_kind = 'day'`, "arc42.org").Scan(&down, &outages)
	if err != nil {
		t.Fatal(err)
	}
	if down != 30 || outages != 1 {
		t.Errorf("down=%d outages=%d, want 30/1", down, outages)
	}
}

func TestProbeRunHeartbeat(t *testing.T) {
	db := testDB(t)
	if _, found, _ := ReadLastRun(db); found {
		t.Fatal("empty probe_run must report not-found")
	}
	if err := WriteProbeRun(db, tnow.Add(-10*time.Minute), "gha", 10, 4200); err != nil {
		t.Fatal(err)
	}
	if err := WriteProbeRun(db, tnow.Add(-5*time.Minute), "gha", 10, 3900); err != nil {
		t.Fatal(err)
	}
	at, found, err := ReadLastRun(db)
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if !at.Equal(tnow.Add(-5 * time.Minute)) {
		t.Errorf("got %v, want newest run", at)
	}
}

func TestForSiteFreshUp(t *testing.T) {
	db := testDB(t)
	// eight fully-up days so the 7d window is covered
	for i := 7; i >= 0; i-- {
		day := tnow.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		if err := UpsertDailyBucket(db, "arc42.org", day, 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := WriteTransition(db, "arc42.org", "up", tnow.AddDate(0, 0, -7), 200, "", "gha"); err != nil {
		t.Fatal(err)
	}
	if err := WriteProbeRun(db, tnow.Add(-10*time.Minute), "gha", 10, 4000); err != nil {
		t.Fatal(err)
	}
	av := ForSite(db, "arc42.org", tnow)
	if !av.Measured || av.State != "up" || av.Stale {
		t.Errorf("got %+v", av)
	}
	if av.Uptime7d != "100%" {
		t.Errorf("Uptime7d = %q", av.Uptime7d)
	}
	if av.Uptime30d != "n/a" || av.Uptime30dNr != -1 {
		t.Errorf("30d must be n/a after 8 days: %q/%v", av.Uptime30d, av.Uptime30dNr)
	}
	if av.MeasuredSince != "2026-08" {
		t.Errorf("MeasuredSince = %q", av.MeasuredSince)
	}
	if len(av.Days) != 30 {
		t.Errorf("len(Days) = %d", len(av.Days))
	}
	if av.LastCheckedAgo != "10 min ago" {
		t.Errorf("LastCheckedAgo = %q", av.LastCheckedAgo)
	}
}

func TestForSiteStale(t *testing.T) {
	db := testDB(t)
	if err := WriteTransition(db, "arc42.org", "up", tnow.Add(-3*time.Hour), 200, "", "gha"); err != nil {
		t.Fatal(err)
	}
	if err := WriteProbeRun(db, tnow.Add(-2*time.Hour), "gha", 10, 4000); err != nil {
		t.Fatal(err)
	}
	av := ForSite(db, "arc42.org", tnow)
	if !av.Stale {
		t.Error("2h-old heartbeat must be stale")
	}
	if av.Token() != "unknown" {
		t.Errorf("Token = %q, want unknown", av.Token())
	}
}

func TestForSiteNeverProbed(t *testing.T) {
	db := testDB(t)
	av := ForSite(db, "arc42.org", tnow)
	if av.Measured {
		t.Error("no rows at all must mean unmeasured")
	}
	if av.Token() != "unmonitored" {
		t.Errorf("Token = %q", av.Token())
	}
}

func TestFamilyVerdict(t *testing.T) {
	m := map[string]types.SiteAvailability{
		"a": {Measured: true, State: "up", LastCheckedAgo: "10 min ago"},
		"b": {Measured: true, State: "down"},
		"c": {Measured: true, State: "degraded"},
	}
	f := Family(m)
	if !f.Measured || f.NrMonitored != 3 || f.NrDown != 1 || f.NrDegraded != 1 || f.AllUp {
		t.Errorf("got %+v", f)
	}
	if f.Token() != "down" {
		t.Errorf("Token = %q", f.Token())
	}
	allUp := map[string]types.SiteAvailability{
		"a": {Measured: true, State: "up", LastCheckedAgo: "10 min ago"},
	}
	if g := Family(allUp); !g.AllUp || g.Token() != "up" {
		t.Errorf("got %+v", g)
	}
	if h := Family(map[string]types.SiteAvailability{"a": {Measured: false}}); h.Measured {
		t.Errorf("unmeasured sites only: got %+v", h)
	}
}
