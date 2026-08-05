# Availability Monitoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A GitHub-Actions-scheduled Go prober writes availability data into Turso; the statistics service reads it and renders it into the intro table's Status column, one line per tile, and an Availability section on each per-site page.

**Architecture:** `cmd/probe` is a batch job (wake, measure, record, exit) run every 15 min by a workflow; it writes state transitions to `status_snapshot`, daily rollups to `status_bucket`, and a heartbeat row per run to `probe_run`. A new `internal/availability` package computes uptime windows, strip cells, incidents and staleness at read time; `internal/domain` attaches the results to the cached statistics; three templates render them. No Slack in this iteration.

**Tech Stack:** Go 1.21 (module `arc42-status`), Turso/libSQL (prod) + SQLite (dev/test), Atlas-managed `schema.hcl`, Go `html/template` fragments over htmx, Jekyll static shell, GitHub Actions cron.

**Spec:** `docs/superpowers/specs/2026-08-05-availability-monitoring-design.md` — the requirements authority for every task below.

## Global Constraints

- Work in repo `status.arc42.org-site`, branch `dashboard-v2`. Never commit `go-app/set-api-keys.sh`.
- The working tree already carries uncommitted, unrelated changes (`.gitignore`, sandbox deletions). Commit ONLY the files each task names — always `git add <explicit paths>`, never `git add -A` or `git add .`.
- Run Go commands from `go-app/` (`cd go-app && go test ./...`). Tests must not need network or API keys: never import `internal/plausible`, `internal/github`, or `internal/domain` from test files or from `cmd/probe` (their init/env checks demand keys).
- All availability timestamps are UTC; DB datetime strings use `database.DateTimeLayout` = `"2006-01-02 15:04:05"`.
- States are exactly the CSS token suffixes: `up`, `degraded`, `down`, `unknown`, `unmonitored`. Day-cell states are exactly: `up`, `partial`, `down`, `nodata`.
- Uptime percentages count only `down` minutes as downtime; `degraded` counts as up. A window whose start predates the first measured day renders `n/a` — never a percentage over a shorter span than the label claims.
- Cadence facts: probe every 15 min (`CycleMinutes = 15`), stale after 45 min (3 cycles). Published copy says "checked every ~15 minutes", nothing more precise.
- No colour values in Go or templates: state enters HTML only as CSS class suffixes (existing contract).
- Go code comments follow the repo's house style: explain *why*, full sentences. Match surrounding density; don't narrate the obvious.
- Slack: out of scope. Do not add Slack calls anywhere.

---

### Task 1: Schema — commit status tables, add `probe_run`

**Files:**
- Modify: `go-app/internal/database/schema.hcl` (status_snapshot/status_bucket already in working tree, uncommitted; add probe_run)
- Modify: `Makefile` (target `db-apply-dev`)

**Interfaces:**
- Produces: tables `status_snapshot`, `status_bucket`, `probe_run` as declared below. Later tasks' tests create the same tables via SQL in test helpers; the column names here are the contract.

- [ ] **Step 1: Append the `probe_run` table to `schema.hcl`**

Append after the `status_bucket` block:

```hcl
# probe heartbeat: one row per prober run. Freshness on every surface is
# derived from the newest row here; if the workflow stops, this stops
# advancing and the page says "stale" instead of showing old green
# (ADR-0019, deviation noted there: heartbeat table instead of bucket
# timestamps).

table "probe_run" {
  schema = schema.main
  column "run_at" {
    null = false
    type = datetime
  }
  column "vantage" {
    null = false
    type = varchar(64)
  }
  column "sites_checked" {
    null = false
    type = integer
  }
  column "duration_ms" {
    null = false
    type = integer
  }
}
```

- [ ] **Step 2: Add a Makefile target to apply the schema to the local dev DB**

The dev/test DB is a plain SQLite file at `$HOME/arc42-stats-dev.db` (see `database.pathToDevDB`). Add to the Makefile, in the `# backend` section, and add `db-apply-dev` to `.PHONY`:

```makefile
db-apply-dev: ## Apply schema.hcl to the local dev database (needs atlas CLI)
	@command -v atlas >/dev/null 2>&1 || { \
		printf "atlas CLI not installed — https://atlasgo.io/getting-started\n"; exit 1; }
	atlas schema apply --auto-approve \
		--url "sqlite://$$HOME/arc42-stats-dev.db" \
		--to "file://$(APP_DIR)/internal/database/schema.hcl"
```

- [ ] **Step 3: Verify the schema parses and applies locally**

Run: `make db-apply-dev`
Expected: atlas reports the three new tables created (or "already in sync" for pre-existing ones). If atlas is not installed locally, verify parse-only: `atlas schema inspect --url "file://go-app/internal/database/schema.hcl" --dev-url "sqlite://file?mode=memory"` — any HCL syntax error fails here.

- [ ] **Step 4: Commit**

```bash
git add go-app/internal/database/schema.hcl Makefile
git commit -m "feat(schema): status tables committed, probe_run heartbeat added (ADR-0019)"
```

Note: applying to the PROD Turso DB is a rollout step the owner runs (same atlas command with the Turso URL + auth token); record nothing else here.

---

### Task 2: Types — `ExpectedContent`, availability structs, token helpers

**Files:**
- Modify: `go-app/internal/types/types.go`
- Test: `go-app/internal/types/types_availability_test.go` (create)

**Interfaces:**
- Produces (consumed by every later task):
  - `types.Property.ExpectedContent string`
  - `types.DayCell{Date, State string; DowntimeMinutes int}`
  - `types.Incident{State, StartString, EndString, Duration, Detail string}`
  - `types.SiteAvailability{Measured bool; State string; Stale bool; LastCheckedAgo string; Uptime7d, Uptime30d, Uptime12m string; Uptime30dNr float64; MeasuredSince string; Days []DayCell; Incidents []Incident}` with method `Token() string`
  - `types.FamilyAvailability{Measured bool; Stale bool; AllUp bool; NrMonitored, NrDown, NrDegraded int; LastCheckedAgo string}` with method `Token() string`
  - `types.SiteStatsType.Availability SiteAvailability` (new field)
  - `types.Arc42Statistics.Availability FamilyAvailability` (new field)
  - `types.Monitored(p Property) bool` — true iff `p.Host != "" && !p.Planned`

- [ ] **Step 1: Write the failing test**

Create `go-app/internal/types/types_availability_test.go`:

```go
package types

import "testing"

func TestSiteAvailabilityToken(t *testing.T) {
	cases := []struct {
		name string
		av   SiteAvailability
		want string
	}{
		{"never measured", SiteAvailability{Measured: false, State: "up"}, "unmonitored"},
		{"stale overrides state", SiteAvailability{Measured: true, Stale: true, State: "up"}, "unknown"},
		{"fresh up", SiteAvailability{Measured: true, State: "up"}, "up"},
		{"fresh down", SiteAvailability{Measured: true, State: "down"}, "down"},
		{"fresh degraded", SiteAvailability{Measured: true, State: "degraded"}, "degraded"},
	}
	for _, c := range cases {
		if got := c.av.Token(); got != c.want {
			t.Errorf("%s: Token() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFamilyAvailabilityToken(t *testing.T) {
	cases := []struct {
		name string
		f    FamilyAvailability
		want string
	}{
		{"never measured", FamilyAvailability{}, "unmonitored"},
		{"stale", FamilyAvailability{Measured: true, Stale: true}, "unknown"},
		{"one down wins", FamilyAvailability{Measured: true, NrDown: 1, NrDegraded: 2}, "down"},
		{"degraded without down", FamilyAvailability{Measured: true, NrDegraded: 1}, "degraded"},
		{"all up", FamilyAvailability{Measured: true, AllUp: true}, "up"},
	}
	for _, c := range cases {
		if got := c.f.Token(); got != c.want {
			t.Errorf("%s: Token() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestMonitored(t *testing.T) {
	if Monitored(Property{Key: "arc42-template"}) {
		t.Error("hostless property must not be monitored")
	}
	if Monitored(Property{Key: "examples.arc42.org", Host: "examples.arc42.org", Planned: true}) {
		t.Error("planned property must not be monitored")
	}
	if !Monitored(Property{Key: "arc42.org", Host: "arc42.org"}) {
		t.Error("hosted, built property must be monitored")
	}
}

func TestEveryMonitoredPropertyDeclaresExpectedContent(t *testing.T) {
	for _, p := range Arc42properties {
		if Monitored(p) && p.ExpectedContent == "" {
			t.Errorf("%s is monitored but declares no ExpectedContent", p.Key)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd go-app && go test ./internal/types/ -run 'Token|Monitored|ExpectedContent' -v`
Expected: FAIL — compile errors, the types don't exist yet.

- [ ] **Step 3: Implement**

In `types.go`:

a) Add to `Property` (after `Planned bool`):

```go
	// ExpectedContent is the substring the prober requires in the response
	// body before it calls the site up: a build that serves an empty 200
	// page is degraded, not up. Empty for properties that are never probed
	// (no host, or planned).
	ExpectedContent string
```

b) Populate `ExpectedContent` in `Arc42properties` for every entry with a Host that is not Planned. Exact values: `"arc42"` for `arc42.org`, `arc42.de`, `docs.arc42.org`, `quality.arc42.org`, `faq.arc42.org`, `canvas.arc42.org`, `status.arc42.org`, `trainings.arc42.org`, `meta.arc42.org`; `"minion"` for `pdfminion.arc42.org`. Leave `arc42-template` and `examples.arc42.org` without.

c) Add `Monitored`:

```go
// Monitored says the prober checks this property: it has a site to probe
// and the site exists. Repo-only and planned properties are never probed,
// and their availability renders as "unmonitored" - a different fact from
// "probed and down".
func Monitored(p Property) bool {
	return p.Host != "" && !p.Planned
}
```

d) Add the availability types (place after `ClosedItem`):

```go
// DayCell is one day of the 30-day availability strip. State is a CSS
// class suffix: "up", "partial", "down", or "nodata" for days before
// measurement began (or a gap in it) - which is a different fact from a
// day with zero downtime.
type DayCell struct {
	Date            string // "2006-01-02"
	State           string
	DowntimeMinutes int
}

// Incident is one contiguous not-up period, reconstructed from
// consecutive status_snapshot transitions for the detail page.
type Incident struct {
	State       string // "down" or "degraded"
	StartString string // e.g. "02 Aug 22:00"
	EndString   string // e.g. "02 Aug 22:15", or "ongoing"
	Duration    string // e.g. "15 min", "2 h 30 min"; empty while ongoing
	Detail      string // what the probe saw: "timeout", "slow: 3480ms", "502"
}

// SiteAvailability is everything one property's surfaces render about
// uptime. Measured false means the prober has never recorded this site;
// every other field is then meaningless and the token is "unmonitored".
type SiteAvailability struct {
	Measured       bool
	State          string // latest recorded state: "up", "degraded", "down"
	Stale          bool   // newest probe_run older than availability.StaleAfter
	LastCheckedAgo string // e.g. "12 min ago"
	Uptime7d       string // "100%", "99.98%", or "n/a"
	Uptime30d      string
	Uptime12m      string
	Uptime30dNr    float64 // -1 when Uptime30d is "n/a"; the table's sort key
	MeasuredSince  string  // "2026-08" while any window is uncovered, else ""
	Days           []DayCell  // exactly 30, oldest first
	Incidents      []Incident // newest first, capped in internal/availability
}

// Token maps availability onto the CSS status tokens. Staleness beats the
// recorded state: an old green is the one thing this page must never show.
func (a SiteAvailability) Token() string {
	if !a.Measured {
		return "unmonitored"
	}
	if a.Stale {
		return "unknown"
	}
	return a.State
}

// FamilyAvailability is the one-line verdict above the table.
type FamilyAvailability struct {
	Measured       bool
	Stale          bool
	AllUp          bool
	NrMonitored    int
	NrDown         int
	NrDegraded     int
	LastCheckedAgo string
}

func (f FamilyAvailability) Token() string {
	switch {
	case !f.Measured:
		return "unmonitored"
	case f.Stale:
		return "unknown"
	case f.NrDown > 0:
		return "down"
	case f.NrDegraded > 0:
		return "degraded"
	default:
		return "up"
	}
}
```

e) Add field `Availability SiteAvailability` to `SiteStatsType` (after `NrUntriaged`), and field `Availability FamilyAvailability` to `Arc42Statistics` (after `Totals`).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd go-app && go test ./internal/types/ -v`
Expected: PASS (all, including pre-existing tests if any).

- [ ] **Step 5: Build the whole module to catch template-independent breakage**

Run: `cd go-app && go build ./...`
Expected: clean build.

- [ ] **Step 6: Commit**

```bash
git add go-app/internal/types/types.go go-app/internal/types/types_availability_test.go
git commit -m "feat(types): availability structs, tokens, ExpectedContent per property"
```

---

### Task 3: `internal/availability` — pure computation

**Files:**
- Create: `go-app/internal/availability/compute.go`
- Test: `go-app/internal/availability/compute_test.go`

**Interfaces:**
- Consumes: `types.DayCell`, `types.Incident` (Task 2).
- Produces (consumed by Tasks 4, 6):
  - Constants: `CycleMinutes = 15`, `StaleAfter = 45 * time.Minute`, `PartialDayMaxDownMinutes = 240`, `MaxIncidentsShown = 5`, `MaxDowntimePerRunMinutes = 30`
  - `type BucketDay struct { Day time.Time; DowntimeMinutes int; OutageCount int }`
  - `type SnapshotRow struct { Status string; ChangedAt time.Time; ResponseMs int; Detail string }`
  - `func Ago(from, now time.Time) string`
  - `func UptimeWindow(buckets []BucketDay, windowDays int, now time.Time) (label string, pct float64)` — buckets sorted oldest-first, pct is `-1` when label is `"n/a"`
  - `func MeasuredSince(buckets []BucketDay, now time.Time) string` — `"2026-08"` style, `""` when the 12m window is fully covered
  - `func StripCells(buckets []BucketDay, now time.Time) []types.DayCell` — exactly 30, oldest first
  - `func Incidents(rows []SnapshotRow, now time.Time) []types.Incident` — rows newest-first in, incidents newest-first out, capped at `MaxIncidentsShown`

- [ ] **Step 1: Write the failing tests**

Create `compute_test.go`:

```go
package availability

import (
	"testing"
	"time"
)

// day is a helper: UTC midnight n days before now.
func day(now time.Time, n int) time.Time {
	return now.UTC().AddDate(0, 0, -n).Truncate(24 * time.Hour)
}

var now = time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

// fullDays builds one zero-downtime bucket per day for the last n days
// (including today), oldest first.
func fullDays(n int) []BucketDay {
	out := make([]BucketDay, 0, n)
	for i := n - 1; i >= 0; i-- {
		out = append(out, BucketDay{Day: day(now, i)})
	}
	return out
}

func TestAgo(t *testing.T) {
	cases := []struct {
		from time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-12 * time.Minute), "12 min ago"},
		{now.Add(-3 * time.Hour), "3 h ago"},
		{now.Add(-49 * time.Hour), "2 days ago"},
	}
	for _, c := range cases {
		if got := Ago(c.from, now); got != c.want {
			t.Errorf("Ago(%v) = %q, want %q", c.from, got, c.want)
		}
	}
}

func TestUptimeWindowAllUp(t *testing.T) {
	label, pct := UptimeWindow(fullDays(10), 7, now)
	if label != "100%" || pct != 100 {
		t.Errorf("got %q/%v, want 100%%/100", label, pct)
	}
}

func TestUptimeWindowWithDowntime(t *testing.T) {
	buckets := fullDays(10)
	// 144 minutes down on one full (non-today) day
	buckets[5].DowntimeMinutes = 144
	label, pct := UptimeWindow(buckets, 7, now)
	// covered: 6 full days x 1440 + 720 today = 9360 min; 144 down
	// => 98.4615...% -> "98.46%"
	if label != "98.46%" {
		t.Errorf("label = %q, want 98.46%%", label)
	}
	if pct < 98.4 || pct > 98.5 {
		t.Errorf("pct = %v", pct)
	}
}

func TestUptimeWindowUncoveredIsNA(t *testing.T) {
	// measurement began 3 days ago: the 7d window is not covered
	label, pct := UptimeWindow(fullDays(3), 7, now)
	if label != "n/a" || pct != -1 {
		t.Errorf("got %q/%v, want n/a/-1", label, pct)
	}
}

func TestUptimeWindowNoData(t *testing.T) {
	label, pct := UptimeWindow(nil, 7, now)
	if label != "n/a" || pct != -1 {
		t.Errorf("got %q/%v, want n/a/-1", label, pct)
	}
}

func TestMeasuredSince(t *testing.T) {
	if got := MeasuredSince(fullDays(10), now); got != "2026-08" {
		t.Errorf("partial coverage: got %q, want 2026-08", got)
	}
	if got := MeasuredSince(fullDays(370), now); got != "" {
		t.Errorf("full 12m coverage: got %q, want empty", got)
	}
	if got := MeasuredSince(nil, now); got != "" {
		t.Errorf("no data: got %q, want empty", got)
	}
}

func TestStripCells(t *testing.T) {
	buckets := fullDays(10)
	buckets[9].DowntimeMinutes = 30   // today: partial
	buckets[4].DowntimeMinutes = 500  // beyond PartialDayMaxDownMinutes: down
	cells := StripCells(buckets, now)
	if len(cells) != 30 {
		t.Fatalf("len = %d, want 30", len(cells))
	}
	if cells[0].State != "nodata" {
		t.Errorf("oldest cell = %q, want nodata (before measurement)", cells[0].State)
	}
	if last := cells[29]; last.State != "partial" || last.DowntimeMinutes != 30 {
		t.Errorf("today = %+v, want partial/30", last)
	}
	if cells[24].State != "down" {
		t.Errorf("cells[24] = %q, want down", cells[24].State)
	}
	if cells[28].State != "up" {
		t.Errorf("cells[28] = %q, want up", cells[28].State)
	}
	if cells[29].Date != "2026-08-20" {
		t.Errorf("today's date = %q", cells[29].Date)
	}
}

func TestIncidentsReconstruction(t *testing.T) {
	rows := []SnapshotRow{ // newest first, as the store returns them
		{Status: "up", ChangedAt: now.Add(-1 * time.Hour)},
		{Status: "down", ChangedAt: now.Add(-90 * time.Minute), Detail: "timeout"},
		{Status: "up", ChangedAt: now.Add(-24 * time.Hour)},
		{Status: "degraded", ChangedAt: now.Add(-25 * time.Hour), Detail: "slow: 3480ms"},
		{Status: "up", ChangedAt: now.Add(-48 * time.Hour)},
	}
	inc := Incidents(rows, now)
	if len(inc) != 2 {
		t.Fatalf("len = %d, want 2", len(inc))
	}
	first := inc[0] // newest first
	if first.State != "down" || first.Detail != "timeout" || first.Duration != "30 min" {
		t.Errorf("newest incident = %+v", first)
	}
	if inc[1].State != "degraded" || inc[1].Duration != "1 h 0 min" {
		t.Errorf("older incident = %+v", inc[1])
	}
}

func TestOngoingIncident(t *testing.T) {
	rows := []SnapshotRow{
		{Status: "down", ChangedAt: now.Add(-20 * time.Minute), Detail: "502"},
		{Status: "up", ChangedAt: now.Add(-3 * time.Hour)},
	}
	inc := Incidents(rows, now)
	if len(inc) != 1 {
		t.Fatalf("len = %d, want 1", len(inc))
	}
	if inc[0].EndString != "ongoing" {
		t.Errorf("EndString = %q, want ongoing", inc[0].EndString)
	}
}

func TestIncidentsCap(t *testing.T) {
	rows := make([]SnapshotRow, 0, 20)
	// ten alternating down/up pairs, newest first
	for i := 0; i < 10; i++ {
		rows = append(rows,
			SnapshotRow{Status: "up", ChangedAt: now.Add(-time.Duration(2*i+1) * time.Hour)},
			SnapshotRow{Status: "down", ChangedAt: now.Add(-time.Duration(2*i+2) * time.Hour)})
	}
	if got := len(Incidents(rows, now)); got != MaxIncidentsShown {
		t.Errorf("len = %d, want %d", got, MaxIncidentsShown)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd go-app && go test ./internal/availability/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement `compute.go`**

```go
// Package availability turns the prober's Turso rows into what the three
// surfaces render: current state, uptime windows, the 30-day strip,
// incidents, and staleness. Computation is at read time; the prober only
// records facts (ADR-0019).
package availability

import (
	"fmt"
	"time"

	"arc42-status/internal/types"
)

const (
	// CycleMinutes is the probe cadence the workflow declares. The page
	// copy says "~15 minutes"; keep the two in sync.
	CycleMinutes = 15

	// StaleAfter is three missed cycles. After that every surface shows
	// "unknown" rather than the last recorded state: absence of data is
	// itself the signal (ADR-0019).
	StaleAfter = 45 * time.Minute

	// PartialDayMaxDownMinutes separates a "partial" strip cell from a
	// "down" one: up to four hours down is a bad day, more is a lost one.
	PartialDayMaxDownMinutes = 240

	// MaxIncidentsShown caps the detail page's incident list.
	MaxIncidentsShown = 5

	// MaxDowntimePerRunMinutes caps how much downtime one probe run may
	// attribute: two cycles, so a delayed run cannot invent an hour-long
	// outage from a fifteen-minute one.
	MaxDowntimePerRunMinutes = 30

	minutesPerDay = 24 * 60
	stripDays     = 30
)

// BucketDay is one status_bucket row: one site, one UTC day.
type BucketDay struct {
	Day             time.Time
	DowntimeMinutes int
	OutageCount     int
}

// SnapshotRow is one status_snapshot row: a recorded state transition.
type SnapshotRow struct {
	Status     string
	ChangedAt  time.Time
	ResponseMs int
	Detail     string
}

// Ago phrases how long past `from` is, for "last check ..." lines.
func Ago(from, now time.Time) string {
	d := now.Sub(from)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}

func utcDay(t time.Time) time.Time {
	return t.UTC().Truncate(24 * time.Hour)
}

// coveredMinutes is how much of one bucket day lies in the past: a full
// day for yesterday and older, the elapsed part for today. The
// denominator of every percentage, so today can never dilute it.
func coveredMinutes(bucketDay time.Time, now time.Time) int {
	if bucketDay.Equal(utcDay(now)) {
		return int(now.UTC().Sub(bucketDay).Minutes())
	}
	return minutesPerDay
}

// UptimeWindow computes one labelled percentage over the last windowDays
// days. It refuses to answer ("n/a", -1) when measurement started inside
// the window: a 12m figure over three weeks of data would be a lie with
// decimals (spec: honesty over completeness).
func UptimeWindow(buckets []BucketDay, windowDays int, now time.Time) (string, float64) {
	if len(buckets) == 0 {
		return types.NotAvailable, -1
	}
	windowStart := utcDay(now).AddDate(0, 0, -(windowDays - 1))
	// buckets are oldest-first; the first row is the start of measurement
	if buckets[0].Day.After(windowStart) {
		return types.NotAvailable, -1
	}
	covered, down := 0, 0
	for _, b := range buckets {
		if b.Day.Before(windowStart) {
			continue
		}
		c := coveredMinutes(b.Day, now)
		covered += c
		d := b.DowntimeMinutes
		if d > c {
			d = c
		}
		down += d
	}
	if covered == 0 {
		return types.NotAvailable, -1
	}
	pct := 100 * (1 - float64(down)/float64(covered))
	if down == 0 {
		return "100%", 100
	}
	return fmt.Sprintf("%.2f%%", pct), pct
}

// MeasuredSince names the month measurement began, but only while any of
// the three windows is still uncovered - once 12 months are on record
// the qualifier would be noise.
func MeasuredSince(buckets []BucketDay, now time.Time) string {
	if len(buckets) == 0 {
		return ""
	}
	twelveMonthsAgo := utcDay(now).AddDate(0, -12, 0)
	if !buckets[0].Day.After(twelveMonthsAgo) {
		return ""
	}
	return buckets[0].Day.Format("2006-01")
}

// StripCells maps the last 30 days onto strip cells, oldest first. Days
// without a bucket row are "nodata": before measurement began, or the
// prober was not running - either way, not a green day.
func StripCells(buckets []BucketDay, now time.Time) []types.DayCell {
	byDay := make(map[string]BucketDay, len(buckets))
	for _, b := range buckets {
		byDay[b.Day.Format("2006-01-02")] = b
	}
	cells := make([]types.DayCell, 0, stripDays)
	for i := stripDays - 1; i >= 0; i-- {
		date := utcDay(now).AddDate(0, 0, -i).Format("2006-01-02")
		cell := types.DayCell{Date: date, State: "nodata"}
		if b, ok := byDay[date]; ok {
			switch {
			case b.DowntimeMinutes == 0:
				cell.State = "up"
			case b.DowntimeMinutes <= PartialDayMaxDownMinutes:
				cell.State = "partial"
			default:
				cell.State = "down"
			}
			cell.DowntimeMinutes = b.DowntimeMinutes
		}
		cells = append(cells, cell)
	}
	return cells
}

// incidentTimeLayout is how the detail page prints incident bounds.
const incidentTimeLayout = "02 Jan 15:04"

func humanDuration(d time.Duration) string {
	mins := int(d.Minutes())
	if mins < 60 {
		return fmt.Sprintf("%d min", mins)
	}
	return fmt.Sprintf("%d h %d min", mins/60, mins%60)
}

// Incidents reconstructs not-up periods from transition rows (newest
// first, as the store returns them): each period starts at a non-up
// transition and ends at the next transition to up, or is ongoing.
func Incidents(rows []SnapshotRow, now time.Time) []types.Incident {
	// walk oldest -> newest, then reverse
	incidents := make([]types.Incident, 0)
	var open *types.Incident
	var openStart time.Time
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		if r.Status == "up" {
			if open != nil {
				open.EndString = r.ChangedAt.Format(incidentTimeLayout)
				open.Duration = humanDuration(r.ChangedAt.Sub(openStart))
				incidents = append(incidents, *open)
				open = nil
			}
			continue
		}
		if open != nil {
			// down -> degraded (or the reverse) without an up in
			// between: close the first period here, the state changed.
			open.EndString = r.ChangedAt.Format(incidentTimeLayout)
			open.Duration = humanDuration(r.ChangedAt.Sub(openStart))
			incidents = append(incidents, *open)
		}
		open = &types.Incident{
			State:       r.Status,
			StartString: r.ChangedAt.Format(incidentTimeLayout),
			Detail:      r.Detail,
		}
		openStart = r.ChangedAt
	}
	if open != nil {
		open.EndString = "ongoing"
		incidents = append(incidents, *open)
	}
	// newest first, capped
	for i, j := 0, len(incidents)-1; i < j; i, j = i+1, j-1 {
		incidents[i], incidents[j] = incidents[j], incidents[i]
	}
	if len(incidents) > MaxIncidentsShown {
		incidents = incidents[:MaxIncidentsShown]
	}
	return incidents
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd go-app && go test ./internal/availability/ -v`
Expected: PASS. If `TestUptimeWindowWithDowntime` disagrees on rounding, fix the expectation from the actual arithmetic (9360 covered, 144 down → 98.4615% → `"98.46%"`), not by changing the format.

- [ ] **Step 5: Commit**

```bash
git add go-app/internal/availability/compute.go go-app/internal/availability/compute_test.go
git commit -m "feat(availability): pure window/strip/incident/staleness computation"
```

---

### Task 4: `internal/availability` — store (reads and writes)

**Files:**
- Create: `go-app/internal/availability/store.go`
- Test: `go-app/internal/availability/store_test.go`

**Interfaces:**
- Consumes: Task 3 types/functions; `database.DateTimeLayout`; `types.SiteAvailability`, `types.FamilyAvailability`, `types.Monitored`, `types.Arc42properties`.
- Produces (consumed by Tasks 5–8):
  - `func LastState(db *sql.DB, site string) (status string, at time.Time, found bool, err error)`
  - `func WriteTransition(db *sql.DB, site, status string, at time.Time, responseMs int, detail, vantage string) error`
  - `func UpsertDailyBucket(db *sql.DB, site string, day time.Time, addDownMinutes, addOutages int) error`
  - `func WriteProbeRun(db *sql.DB, at time.Time, vantage string, sitesChecked, durationMs int) error`
  - `func ReadLastRun(db *sql.DB) (time.Time, bool, error)`
  - `func ForSite(db *sql.DB, site string, now time.Time) types.SiteAvailability`
  - `func ForAllSites(db *sql.DB, now time.Time) map[string]types.SiteAvailability` — one entry per monitored property
  - `func Family(m map[string]types.SiteAvailability) types.FamilyAvailability`

- [ ] **Step 1: Write the failing tests**

Create `store_test.go`. The test DB is in-memory SQLite (driver `sqlite3`, already a module dependency) with tables created from DDL that mirrors `schema.hcl` — if a column name here drifts from `schema.hcl`, these tests are the tripwire, so keep the DDL in one helper:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd go-app && go test ./internal/availability/ -run 'Transition|Bucket|ProbeRun|ForSite|Family' -v`
Expected: FAIL — functions not defined.

- [ ] **Step 3: Implement `store.go`**

```go
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

func LastState(db *sql.DB, site string) (string, time.Time, bool, error) {
	var status, at string
	err := db.QueryRow(
		`SELECT status, changed_at FROM status_snapshot
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
	err := db.QueryRow(`SELECT run_at FROM probe_run ORDER BY run_at DESC LIMIT 1`).Scan(&at)
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
		`SELECT bucket_start, downtime_minutes, outage_count FROM status_bucket
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
		`SELECT status, changed_at, COALESCE(response_ms, 0), COALESCE(detail, '')
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd go-app && go test ./internal/availability/ -v`
Expected: PASS (Tasks 3 + 4 tests together).

- [ ] **Step 5: Commit**

```bash
git add go-app/internal/availability/store.go go-app/internal/availability/store_test.go
git commit -m "feat(availability): Turso read/write layer with in-memory SQLite tests"
```

---

### Task 5: `cmd/probe` — classification and 2-of-3 confirmation

**Files:**
- Create: `go-app/cmd/probe/probe.go`
- Test: `go-app/cmd/probe/probe_test.go`

**Interfaces:**
- Consumes: `types.Property.{Key,Host,ExpectedContent}`, `types.Monitored`.
- Produces (consumed by Task 6's `main.go`, same package):
  - `type Result struct { Site, State, Detail string; ResponseMs int }`
  - `func classify(statusCode, elapsedMs int, body, expected string, err error) (state, detail string)`
  - `func probeOnce(client *http.Client, url, expected string) (state, detail string, responseMs int)`
  - `func ProbeSite(client *http.Client, p types.Property) Result`
  - `var pause = time.Sleep` (package var, injectable for tests)
  - Constants: `RequestTimeoutSeconds = 10`, `SlowMs = 2000`, `confirmAttempts = 3`, `confirmNeeded = 2`, `retryPauseSeconds = 5`

- [ ] **Step 1: Write the failing tests**

Create `go-app/cmd/probe/probe_test.go`:

```go
package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"arc42-status/internal/types"
)

func init() {
	pause = func(time.Duration) {} // no real sleeping in tests
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		elapsedMs  int
		body       string
		err        error
		wantState  string
		wantDetail string
	}{
		{"fast 200 with content", 200, 150, "<html>arc42 rocks</html>", nil, "up", ""},
		{"slow 200", 200, 3480, "<html>arc42</html>", nil, "degraded", "slow: 3480ms"},
		{"content missing", 200, 150, "<html>empty</html>", nil, "degraded", "content"},
		{"http error status", 502, 80, "", nil, "down", "502"},
		{"transport error", 0, 0, "", errors.New("dial tcp: timeout"), "down", "timeout"},
	}
	for _, c := range cases {
		state, detail := classify(c.statusCode, c.elapsedMs, c.body, "arc42", c.err)
		if state != c.wantState || detail != c.wantDetail {
			t.Errorf("%s: got %s/%q, want %s/%q", c.name, state, detail, c.wantState, c.wantDetail)
		}
	}
}

func TestProbeSiteUpNeedsOneAttempt(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("arc42"))
	}))
	defer srv.Close()

	p := types.Property{Key: "test.site", Host: srv.Listener.Addr().String(), ExpectedContent: "arc42"}
	res := probeURL(srv.Client(), p, srv.URL)
	if res.State != "up" {
		t.Errorf("state = %s", res.State)
	}
	if hits.Load() != 1 {
		t.Errorf("an up verdict must need exactly one attempt, got %d", hits.Load())
	}
}

func TestProbeSiteDownNeedsConfirmation(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	p := types.Property{Key: "test.site", ExpectedContent: "arc42"}
	res := probeURL(srv.Client(), p, srv.URL)
	if res.State != "down" || res.Detail != "502" {
		t.Errorf("got %s/%q", res.State, res.Detail)
	}
	if hits.Load() < 2 {
		t.Errorf("down needs 2-of-3 confirmation, saw only %d attempts", hits.Load())
	}
}

func TestProbeSiteFlickerIsNotDown(t *testing.T) {
	// first attempt fails, the two confirmations succeed: one dropped
	// packet must not write an incident
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("arc42"))
	}))
	defer srv.Close()

	p := types.Property{Key: "test.site", ExpectedContent: "arc42"}
	res := probeURL(srv.Client(), p, srv.URL)
	if res.State != "up" {
		t.Errorf("state = %s, want up (failure not confirmed)", res.State)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd go-app && go test ./cmd/probe/ -v`
Expected: FAIL — package/files don't exist.

- [ ] **Step 3: Implement `probe.go`**

Note the test hook: tests call `probeURL(client, property, url)` so httptest URLs work; production code calls `ProbeSite`, which derives `https://<Host>/`.

```go
// Command probe is the availability prober of ADR-0019: a batch job run
// by a scheduled GitHub Actions workflow. It measures every monitored
// arc42 property once, records transitions and daily rollups in Turso,
// and exits. It is deliberately not a service - nothing to keep running,
// nothing to pay for.
package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"arc42-status/internal/types"
)

const (
	// RequestTimeoutSeconds bounds one attempt; a site slower than this
	// is down for practical purposes.
	RequestTimeoutSeconds = 10

	// SlowMs is the degraded threshold: reachable, but not healthy.
	SlowMs = 2000

	// A failure must be confirmed before it is recorded: confirmNeeded
	// of confirmAttempts attempts, retryPauseSeconds apart, must fail.
	// One dropped packet must never write an incident.
	confirmAttempts   = 3
	confirmNeeded     = 2
	retryPauseSeconds = 5
)

// pause is time.Sleep, injectable so the confirmation tests don't wait.
var pause = time.Sleep

// Result is one site's confirmed measurement.
type Result struct {
	Site       string
	State      string // "up", "degraded", "down"
	Detail     string
	ResponseMs int
}

// classify maps one attempt's raw outcome onto a state, per the table in
// ADR-0019.
func classify(statusCode, elapsedMs int, body, expected string, err error) (string, string) {
	if err != nil {
		detail := "error"
		msg := err.Error()
		switch {
		case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded"):
			detail = "timeout"
		case strings.Contains(msg, "no such host"):
			detail = "dns"
		case strings.Contains(msg, "certificate") || strings.Contains(msg, "tls"):
			detail = "tls"
		}
		return "down", detail
	}
	if statusCode < 200 || statusCode > 299 {
		return "down", fmt.Sprintf("%d", statusCode)
	}
	if elapsedMs > SlowMs {
		return "degraded", fmt.Sprintf("slow: %dms", elapsedMs)
	}
	if expected != "" && !strings.Contains(body, expected) {
		// serves 200 but the build broke - the case pure ping
		// monitoring misses
		return "degraded", "content"
	}
	return "up", ""
}

func probeOnce(client *http.Client, url, expected string) (string, string, int) {
	start := time.Now()
	resp, err := client.Get(url)
	elapsedMs := int(time.Since(start).Milliseconds())
	if err != nil {
		state, detail := classify(0, elapsedMs, "", expected, err)
		return state, detail, elapsedMs
	}
	defer resp.Body.Close()
	// 1 MB is plenty to find the expected substring; the pages are small
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	state, detail := classify(resp.StatusCode, elapsedMs, string(body), expected, nil)
	return state, detail, elapsedMs
}

// probeURL measures one URL with confirmation. Split from ProbeSite so
// tests can point it at an httptest server.
func probeURL(client *http.Client, p types.Property, url string) Result {
	state, detail, ms := probeOnce(client, url, p.ExpectedContent)
	if state == "up" {
		return Result{Site: p.Key, State: state, ResponseMs: ms}
	}
	// not up: confirm before declaring. The first attempt already
	// counts as one vote.
	failVotes := 1
	lastState, lastDetail, lastMs := state, detail, ms
	for attempt := 1; attempt < confirmAttempts; attempt++ {
		pause(retryPauseSeconds * time.Second)
		s, d, m := probeOnce(client, url, p.ExpectedContent)
		if s == "up" {
			continue
		}
		failVotes++
		lastState, lastDetail, lastMs = s, d, m
	}
	if failVotes >= confirmNeeded {
		return Result{Site: p.Key, State: lastState, Detail: lastDetail, ResponseMs: lastMs}
	}
	return Result{Site: p.Key, State: "up", ResponseMs: ms}
}

// ProbeSite measures one property at its public URL.
func ProbeSite(client *http.Client, p types.Property) Result {
	return probeURL(client, p, "https://"+p.Host+"/")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd go-app && go test ./cmd/probe/ -v`
Expected: PASS. (`go build ./...` will fail until Task 6 adds `main()` — that's expected; `go vet ./cmd/probe` may complain about a main package without main, ignore until Task 6.)

- [ ] **Step 5: Commit**

```bash
git add go-app/cmd/probe/probe.go go-app/cmd/probe/probe_test.go
git commit -m "feat(probe): outcome classification with 2-of-3 failure confirmation"
```

---

### Task 6: `cmd/probe` — persistence wiring and `main`

**Files:**
- Create: `go-app/cmd/probe/main.go`
- Test: `go-app/cmd/probe/record_test.go`
- Modify: `Makefile` (target `probe`)

**Interfaces:**
- Consumes: Task 5's `ProbeSite`/`Result`; Task 4's store functions; `database.GetDB()`; `availability.CycleMinutes`, `availability.MaxDowntimePerRunMinutes`.
- Produces: `func recordResult(db *sql.DB, r Result, now, lastRunAt time.Time, haveLastRun bool, vantage string) error` — also the `record_test.go` contract.

- [ ] **Step 1: Write the failing test**

Create `go-app/cmd/probe/record_test.go`. It reuses the availability package's DDL idea but needs its own helper (test helpers don't cross packages):

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd go-app && go test ./cmd/probe/ -run 'Baseline|Unchanged|GoingDown|Capped' -v`
Expected: FAIL — `recordResult` not defined.

- [ ] **Step 3: Implement `main.go`**

```go
package main

import (
	"database/sql"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"arc42-status/internal/availability"
	"arc42-status/internal/database"
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd go-app && go test ./cmd/probe/ -v && go build ./...`
Expected: PASS, clean build.

- [ ] **Step 5: Add the Makefile target**

Add to the `# backend` section, and `probe` to `.PHONY`:

```makefile
probe: check-secrets ## Run the availability prober once against the dev DB
	cd $(APP_DIR) && source ./set-api-keys.sh && go run ./cmd/probe
```

- [ ] **Step 6: Run the prober against the dev DB (integration check)**

Precondition: Task 1's `make db-apply-dev` was run. Run: `make probe`
Expected: log lines "probe: <site> is up" for the ten monitored sites (network permitting), exit 0. Then `sqlite3 ~/arc42-stats-dev.db "SELECT COUNT(*) FROM probe_run; SELECT site, status FROM status_snapshot;"` shows one heartbeat and ten baseline transitions. If the sandbox blocks outbound HTTP, note it and let the workflow (Task 7) be the integration proof.

- [ ] **Step 7: Commit**

```bash
git add go-app/cmd/probe/main.go go-app/cmd/probe/record_test.go Makefile
git commit -m "feat(probe): record transitions, buckets and heartbeat; make probe target"
```

---

### Task 7: GitHub Actions workflow

**Files:**
- Create: `.github/workflows/probe.yml`

**Interfaces:**
- Consumes: `cmd/probe` (Task 6). Repo secret `TURSO_AUTH_TOKEN` (owner sets it at rollout; the deploy workflow's Turso access proves the account has one).

- [ ] **Step 1: Write the workflow**

```yaml
# Availability prober (ADR-0019): scheduled batch job, every 15 minutes.
#
# Facts this design accepts (verified 2026-08-04, see the ADR):
#   - scheduled runs only fire on the default branch (main), so this
#     workflow is dormant until merged there
#   - GitHub delays cron under load; the page says "~15 min" for a reason
#   - after 60 days without repo activity GitHub disables the schedule,
#     which surfaces on the page as growing staleness, never as silence
name: availability probe
on:
  schedule:
    - cron: '*/15 * * * *'
  workflow_dispatch:

permissions:
  contents: read

# never overlap two runs: a delayed run and its successor would double-
# count downtime minutes
concurrency:
  group: probe
  cancel-in-progress: false

defaults:
  run:
    working-directory: ./go-app

jobs:
  probe:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true
      - run: go run ./cmd/probe
        env:
          ENVIRONMENT: PROD
          TURSO_AUTH_TOKEN: ${{ secrets.TURSO_AUTH_TOKEN }}
```

- [ ] **Step 2: Validate the YAML**

Run: `ruby -ryaml -e "YAML.load_file('.github/workflows/probe.yml'); puts 'ok'"` (or `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/probe.yml')); print('ok')"` — whichever is installed).
Expected: `ok`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/probe.yml
git commit -m "feat(probe): scheduled GitHub Actions workflow, every 15 minutes"
```

---

### Task 8: Service read path — domain wiring, `/siteAvailability` endpoint and template

**Files:**
- Modify: `go-app/internal/domain/domain.go`
- Modify: `go-app/internal/api/apiGateway.go`
- Create: `go-app/internal/api/siteAvailability.gohtml`

**Interfaces:**
- Consumes: `availability.ForAllSites`, `availability.Family` (Task 4); `types.SiteStatsType.Availability`, `types.Arc42Statistics.Availability` (Task 2); existing `servePropertyFragment`, `types.SiteDetailData`.
- Produces: route `GET /siteAvailability?site=<key>` returning the fragment whose root is `<div id="siteAvailability" class="site-availability">`; `domain.LoadStats4AllSites` results now carry availability (consumed by Tasks 9, 10).

- [ ] **Step 1: Wire availability into the collection run**

In `domain.go`, add imports `"arc42-status/internal/availability"` and `"arc42-status/internal/database"`. In `LoadStats4AllSites`, after the `for` loop that transfers goroutine results (after line ~118) and before `calculateTotals`, insert:

```go
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
```

- [ ] **Step 2: Register the endpoint**

In `apiGateway.go`:
- Add constant: `const SiteAvailabilityTmpl = "siteAvailability.gohtml"` next to the other template constants.
- Add handler next to `siteTrafficHandler`:

```go
// siteAvailabilityHandler returns the Availability section for one
// property: current state, the 30-day strip, the three windows, and
// recent incidents. Same fragment-per-heading contract as siteDetail and
// siteTraffic.
func siteAvailabilityHandler(w http.ResponseWriter, r *http.Request) {
	servePropertyFragment(w, r, SiteAvailabilityTmpl)
}
```

- Register in `StartAPIServer`: `mux.HandleFunc("/siteAvailability", siteAvailabilityHandler)` after the `/siteTraffic` line.

- [ ] **Step 3: Write the template**

Create `go-app/internal/api/siteAvailability.gohtml`:

```html
<!-- golang template for one property's availability section -->
<!--
  Served at /siteAvailability?site=<key>, swapped into #siteAvailability
  under the Availability heading of /site/<key>/. Fills the region that
  carried the ADR-0019 placeholder.

  Honesty rules, in order: a hostless property has nothing to probe (the
  Jekyll include normally skips the region, this branch answers a direct
  request); an unprobed property is "not monitored yet", never a zero;
  stale data says it is stale and shows "unknown"; only fresh data shows
  a state.

  Class contract for the stylesheet:
    .site-availability            wrapper == #siteAvailability
      .avail-state                current state line (token + text)
      .avail-strip                30 day cells, oldest first
        .avail-strip__day--up / --partial / --down / --nodata
      .avail-figures              the three windows, one line
      .avail-incidents            recent incidents list
        .avail-incident > .avail-incident__span + .avail-incident__what
      .detail__empty              hostless note
-->
{{ $av := .Site.Availability }}
<div id="siteAvailability" class="site-availability">

{{ if not .Site.Host }}
    <p class="detail__empty">A repository, not a site &mdash; there is nothing to probe,
        so no availability exists for it.</p>
{{ else if not $av.Measured }}
    <p class="status-verdict">
        <span class="status-token status-token--unmonitored" role="img" aria-label="not monitored"></span>
        <span><b>Not monitored yet.</b> The prober has not recorded any data for
        {{ .Site.Site }}; this section fills itself from the first run onward.</span>
    </p>
{{ else }}

    <p class="avail-state">
        <span class="status-token status-token--{{ $av.Token }}" role="img" aria-label="{{ $av.Token }}"></span>
        <span><b>{{ $av.Token }}</b>
        {{ if $av.Stale }}&mdash; data is stale, last check {{ $av.LastCheckedAgo }}
        {{ else }}&middot; checked every ~15 minutes &middot; last check {{ $av.LastCheckedAgo }}{{ end }}</span>
    </p>

    <div class="avail-strip" role="img"
         aria-label="daily availability of {{ .Site.Site }} over the last 30 days">
        {{ range $av.Days }}<span class="avail-strip__day avail-strip__day--{{ .State }}"
              title="{{ .Date }}{{ if eq .State "nodata" }}: not measured{{ else }}: {{ .DowntimeMinutes }} min down{{ end }}"></span>{{ end }}
    </div>

    <p class="avail-figures">
        7d&nbsp;{{ $av.Uptime7d }} &middot; 30d&nbsp;{{ $av.Uptime30d }} &middot; 12m&nbsp;{{ $av.Uptime12m }}{{ if $av.MeasuredSince }}
        &middot; measured since {{ $av.MeasuredSince }}{{ end }}
    </p>

    {{ if $av.Incidents }}
    <h3 class="avail-subhead">Recent incidents</h3>
    <ul class="avail-incidents">
        {{ range $av.Incidents }}
        <li class="avail-incident">
            <span class="avail-incident__span">{{ .StartString }} &ndash; {{ .EndString }}</span>
            <span class="avail-incident__what">{{ .State }}{{ if .Detail }} ({{ .Detail }}){{ end }}{{ if .Duration }}, {{ .Duration }}{{ end }}</span>
        </li>
        {{ end }}
    </ul>
    {{ else }}
    <p class="avail-noincidents">No incidents on record.</p>
    {{ end }}

{{ end }}
</div>
```

- [ ] **Step 4: Verify build and existing tests**

Run: `cd go-app && go build ./... && go test ./...`
Expected: clean. (Template files are embedded via the existing `//go:embed *.gohtml`; a parse error would surface at request time — Task 12's rendercheck covers that.)

- [ ] **Step 5: Commit**

```bash
git add go-app/internal/domain/domain.go go-app/internal/api/apiGateway.go go-app/internal/api/siteAvailability.gohtml
git commit -m "feat(api): availability in the collection run, /siteAvailability fragment"
```

---

### Task 9: Intro table — Status column, family verdict, home.md

**Files:**
- Modify: `go-app/internal/api/arc42statistics.gohtml`
- Modify: `docs/_pages/home.md`
- Modify: `docs/assets/css/arc42-status-style.css`

**Interfaces:**
- Consumes: `.Availability` on rows (`SiteAvailability`: `Token`, `Uptime30d`, `Uptime30dNr`) and on the stats object (`FamilyAvailability`: `Token`, `Measured`, `Stale`, `AllUp`, `NrMonitored`, `NrDown`, `NrDegraded`, `LastCheckedAgo`) — Tasks 2, 8.

- [ ] **Step 1: Add the family verdict to the fragment**

In `arc42statistics.gohtml`, insert immediately before `<table id="sortableStatsTable" ...>`:

```html
{{ with .Availability }}
<!-- The verdict is rendered by the service, not by Jekyll: only the
     process that read the data may summarise it. When the service is
     unreachable the static shell shows its error panel instead, so a
     stale build can never claim anything (ADR-0019 honesty chain). -->
<p class="status-verdict">
    <span class="status-token status-token--{{ .Token }}" role="img" aria-label="{{ .Token }}"></span>
    <span>{{ if not .Measured }}<b>Availability is not measured yet.</b>
        The prober has not recorded any data; the Status column fills itself
        from its first run onward.
    {{ else if .Stale }}<b>Availability data is stale.</b>
        Last successful check {{ .LastCheckedAgo }} &mdash; showing the last
        known state as &ldquo;unknown&rdquo;, not as green.
    {{ else if .AllUp }}<b>All {{ .NrMonitored }} monitored sites up.</b>
        Checked every ~15 minutes; last check {{ .LastCheckedAgo }}.
    {{ else }}<b>{{ if .NrDown }}{{ .NrDown }} down{{ end }}{{ if and .NrDown .NrDegraded }}, {{ end }}{{ if .NrDegraded }}{{ .NrDegraded }} degraded{{ end }}</b>
        of {{ .NrMonitored }} monitored sites. Last check {{ .LastCheckedAgo }}.{{ end }}</span>
</p>
{{ end }}
```

- [ ] **Step 2: Fill the Status column**

Replace the status cell (currently the `status-token--unmonitored` span, lines ~54–56) with:

```html
            <td class="status-cell border-left-black" data-order="{{ printf "%.2f" .Availability.Uptime30dNr }}">
                <a class="status-cell__link" href="/site/{{ .Site }}/#availability"
                   aria-label="availability of {{ .Site }}: {{ .Availability.Token }}, 30 days {{ .Availability.Uptime30d }}">
                    <span class="status-token status-token--{{ .Availability.Token }}" role="img" aria-hidden="true"></span><span
                       class="status-cell__pct">{{ .Availability.Uptime30d }}</span>
                </a>
            </td>
```

Also update the `<caption>`: replace the sentence `Availability is not measured yet, so every site shows "not monitored".` with `The Status column shows current availability and 30-day uptime; "n/a" means the measurement window is not filled yet.`

- [ ] **Step 3: Update `home.md`**

a) Delete the static `status-verdict` block (the `<p class="status-verdict">…</p>` with "Availability is not measured yet", lines 34–40) — the service renders the verdict now. Leave `#stats-region` and the skeleton untouched.

b) In the DataTables init at the bottom, delete the `columnDefs` line and its comment (`// column 1 is Status: identical in every row until ADR-19 lands` and `columnDefs: [{ orderable: false, targets: [1] }]`) so the Status column sorts via its `data-order` attribute.

- [ ] **Step 4: CSS for the status cell**

Append to `docs/assets/css/arc42-status-style.css`, after the existing `.status-cell` rules (line ~133):

```css
/* Status column with data: token + 30d percentage, one click target to
   the property's availability section. */
.status-cell__link {
  display: inline-flex;
  align-items: center;
  gap: 0.35em;
  text-decoration: none;
  color: inherit;
}
.status-cell__link:hover .status-cell__pct {
  text-decoration: underline;
}
.status-cell__pct {
  font-variant-numeric: tabular-nums;
  font-size: 0.85em;
}
```

- [ ] **Step 5: Verify**

Run: `cd go-app && go build ./... && go test ./...`
Expected: clean. Full visual verification lands with Task 12 (rendercheck) and Step 6 of Task 13 (local run).

- [ ] **Step 6: Commit**

```bash
git add go-app/internal/api/arc42statistics.gohtml docs/_pages/home.md docs/assets/css/arc42-status-style.css
git commit -m "feat(table): live Status column and family verdict replace the placeholder"
```

---

### Task 10: Tiles — one availability line

**Files:**
- Modify: `go-app/internal/api/tiles.gohtml`
- Modify: `docs/assets/css/arc42-status-style.css`

**Interfaces:**
- Consumes: `.Availability` (`Token`, `Measured`, `State`, `Stale`, `Uptime30d`) on each tile's `SiteStatsType` — Tasks 2, 8. `.Host` decides whether the line exists at all.

- [ ] **Step 1: Add the line to the tile body**

In `tiles.gohtml`, inside the `{{ else }}` branch (non-planned tiles), directly after the closing `</p>` of `.tile__stats` and before `<div class="tile__lists">`:

```html
            {{ if .Host }}
            <!-- Availability: present only for properties with a site to
                 probe. "not monitored yet" is the pre-first-run state,
                 never a zero (ADR-0019). -->
            <p class="tile__avail">
                <span class="status-token status-token--{{ .Availability.Token }}" role="img" aria-label="availability: {{ .Availability.Token }}"></span>
                {{ if .Availability.Measured }}<span class="tile__availtext">{{ .Availability.Token }}
                    &middot; {{ .Availability.Uptime30d }} (30d){{ if .Availability.Stale }} &middot; stale{{ end }}</span>
                {{ else }}<span class="tile__availtext">not monitored yet</span>{{ end }}
            </p>
            {{ end }}
```

Also update the class-contract comment block at the top of the file: add `.tile__avail > .status-token + .tile__availtext   availability line (hosted properties)` under the `.tile__stats` line.

- [ ] **Step 2: CSS**

Append to `arc42-status-style.css`, after the tile stats rules (`.tile__stats`, line ~605):

```css
/* Tile availability line: token + "up · 99.98% (30d)". State colour
   comes from the token alone; the band above stays identity-only. */
.tile__avail {
  display: flex;
  align-items: center;
  gap: 0.4em;
  margin: 0.35rem 0 0;
  font-size: 0.85rem;
}
.tile__availtext {
  font-variant-numeric: tabular-nums;
}
```

- [ ] **Step 3: Verify**

Run: `cd go-app && go build ./...`
Expected: clean (template parse verified by rendercheck in Task 12).

- [ ] **Step 4: Commit**

```bash
git add go-app/internal/api/tiles.gohtml docs/assets/css/arc42-status-style.css
git commit -m "feat(tiles): availability line on hosted tiles"
```

---

### Task 11: Detail pages — fourth htmx region, strip and incident CSS

**Files:**
- Modify: `docs/_includes/site-detail.html`
- Modify: `docs/assets/css/arc42-status-style.css`

**Interfaces:**
- Consumes: `GET {{ site.stats_api }}/siteAvailability?site=<key>` (Task 8), `window.arc42Status.wire(regionId, slotId, path, noun)` (existing `status-regions.js`), the template's class contract (Task 8 Step 3 comment block).

- [ ] **Step 1: Replace the placeholder with a live region**

In `site-detail.html`, replace the whole `<div class="site-availability" id="availability-detail" ...>…</div>` placeholder block (below `<h2 id="availability">Availability</h2>`, including its explanatory `{%- comment -%}` block) with:

```html
{%- comment -%}
  The fourth fragment region, wired like the other three. Hostless
  properties get no region at all: there is no site to probe, and an
  empty section would imply one - the sentence below says so instead.
{%- endcomment -%}
{% if s.host %}
<div id="availability-region" class="stats-region" aria-live="polite" aria-busy="true">

  <div id="siteAvailability"
       hx-get="{{ site.stats_api }}/siteAvailability?site={{ s.key }}"
       hx-trigger="load"
       hx-swap="outerHTML">

    <div class="detail-skeleton" aria-hidden="true">
      <span class="skeleton-bar" style="width:45%"></span>
      <span class="skeleton-bar" style="width:90%;height:1.1rem"></span>
      <span class="skeleton-bar" style="width:60%"></span>
    </div>
    <p class="stats-loading-note">Asking the probe records how {{ s.key }} has been doing&hellip;</p>

  </div>

</div>
{% else %}
<p class="site-page__nodata">
  <span class="status-token status-token--unmonitored" role="img" aria-label="not applicable"></span>
  <span><b>Nothing to probe.</b> {{ s.key }} is a repository without a site of its own,
  so availability does not apply to it.</span>
</p>
{% endif %}
```

- [ ] **Step 2: Wire the region**

In the `<script>` block at the bottom of `site-detail.html` (the one with the two existing `wire` calls), add inside a Liquid guard:

```html
  {% if s.host %}window.arc42Status.wire('availability-region', 'siteAvailability',
                          '/siteAvailability?site={{ s.key }}', 'availability');{% endif %}
```

- [ ] **Step 3: Strip and incident CSS**

The five state hues already exist as background colours on `.status-token--up/--degraded/--down/--unknown/--unmonitored` (lines ~49–100 of `arc42-status-style.css`). First extract them: define custom properties on `:root` at the top of the status-token section — `--state-up`, `--state-degraded`, `--state-down`, `--state-unknown`, `--state-unmonitored` — each holding exactly the background-color value currently written in the corresponding token rule, then change those token rules to use `var(--state-…)`. No hue changes, pure extraction.

Then append:

```css
/* Availability section on the per-site pages -------------------------- */

.avail-state {
  display: flex;
  align-items: center;
  gap: 0.4em;
}

/* The 30-day strip: one flex cell per day, oldest left. Cells reuse the
   token hues via the custom properties above; "partial" is the degraded
   hue - a day with some downtime is a degraded day. */
.avail-strip {
  display: flex;
  gap: 2px;
  max-width: 32rem;
  margin: 0.5rem 0;
}
.avail-strip__day {
  flex: 1 1 0;
  height: 1.1rem;
  border-radius: 2px;
  min-width: 4px;
}
.avail-strip__day--up      { background: var(--state-up); }
.avail-strip__day--partial { background: var(--state-degraded); }
.avail-strip__day--down    { background: var(--state-down); }
.avail-strip__day--nodata  { background: var(--state-unmonitored); opacity: 0.35; }

.avail-figures {
  font-variant-numeric: tabular-nums;
}

.avail-subhead {
  font-size: 1rem;
  margin: 1rem 0 0.25rem;
}
.avail-incidents {
  list-style: none;
  margin: 0;
  padding: 0;
}
.avail-incident {
  display: flex;
  gap: 0.75em;
  font-size: 0.9rem;
  padding: 0.15rem 0;
}
.avail-incident__span {
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.avail-noincidents {
  font-size: 0.9rem;
}
```

- [ ] **Step 4: Verify the Jekyll side builds**

Run: `make build-site` (needs Docker). If Docker is unavailable, at minimum check Liquid syntax by eye and rely on CI's pages build.
Expected: build succeeds; `docs/_site/site/arc42.org/index.html` contains `availability-region`, and `docs/_site/site/arc42-template/index.html` contains "Nothing to probe".

- [ ] **Step 5: Commit**

```bash
git add docs/_includes/site-detail.html docs/assets/css/arc42-status-style.css
git commit -m "feat(detail): live availability region replaces the ADR-19 placeholder"
```

---

### Task 12: rendercheck — availability fixtures

**Files:**
- Modify: `go-app/cmd/rendercheck/main.go`

**Interfaces:**
- Consumes: every template and type above. rendercheck runs via `source ./set-api-keys.sh >/dev/null && go run ./cmd/rendercheck [outdir]` (it imports domain → plausible, whose init wants a key; no call is made).

- [ ] **Step 1: Extend the fixtures**

In `fixtureRows()` (wherever rows are built), give at least four distinct availability shapes so every template branch renders:

```go
	// availability fixtures: one of each shape the templates branch on
	availUp := types.SiteAvailability{
		Measured: true, State: "up", LastCheckedAgo: "12 min ago",
		Uptime7d: "100%", Uptime30d: "99.98%", Uptime12m: "n/a",
		Uptime30dNr: 99.98, MeasuredSince: "2026-08",
		Days:      fixtureDays(),
		Incidents: []types.Incident{{State: "down", StartString: "14 Aug 09:15", EndString: "14 Aug 09:45", Duration: "30 min", Detail: "timeout"}},
	}
	availDown := types.SiteAvailability{
		Measured: true, State: "down", LastCheckedAgo: "3 min ago",
		Uptime7d: "97.32%", Uptime30d: "99.12%", Uptime12m: "n/a",
		Uptime30dNr: 99.12, MeasuredSince: "2026-08",
		Days:      fixtureDays(),
		Incidents: []types.Incident{{State: "down", StartString: "20 Aug 11:45", EndString: "ongoing", Detail: "502"}},
	}
	availStale := types.SiteAvailability{
		Measured: true, State: "up", Stale: true, LastCheckedAgo: "4 h ago",
		Uptime7d: "100%", Uptime30d: "100%", Uptime12m: "n/a",
		Uptime30dNr: 100, MeasuredSince: "2026-08", Days: fixtureDays(),
	}
	availNone := types.SiteAvailability{} // never probed
```

with a helper:

```go
// fixtureDays is a 30-cell strip with every cell state present.
func fixtureDays() []types.DayCell {
	days := make([]types.DayCell, 0, 30)
	for i := 0; i < 30; i++ {
		cell := types.DayCell{Date: fmt.Sprintf("2026-07-%02d", i+1), State: "up"}
		switch i {
		case 0, 1:
			cell.State = "nodata"
		case 13:
			cell.State = "partial"
			cell.DowntimeMinutes = 30
		case 20:
			cell.State = "down"
			cell.DowntimeMinutes = 480
		}
		days = append(days, cell)
	}
	return days
}
```

Assign: `availUp` to most hosted rows, `availDown` to one InTable row, `availStale` to one, `availNone` to one hosted row (pre-first-run look); leave `arc42-template` and the planned row untouched (zero value = unmeasured). Set a matching `types.FamilyAvailability` on the fixture `Arc42Statistics`:

```go
	stats.Availability = types.FamilyAvailability{
		Measured: true, NrMonitored: 10, NrDown: 1,
		LastCheckedAgo: "3 min ago",
	}
```

- [ ] **Step 2: Render the new fragment too**

After the existing siteDetail/siteTraffic render calls (follow the pattern in the file — a render per template with a `types.SiteDetailData`), add:

```go
	availPath := filepath.Join(outDir, "siteAvailability.html")
	render("internal/api/siteAvailability.gohtml", availPath, types.SiteDetailData{
		Site:              rowWithAvailability, // pick the availDown row
		LastUpdatedString: stats.LastUpdatedString,
	})
```

and a second render with the `availNone` row to a file `siteAvailability-unmonitored.html`, and a third with the arc42-template (hostless) row to `siteAvailability-hostless.html`.

- [ ] **Step 3: Run rendercheck**

Run: `cd go-app && source ./set-api-keys.sh >/dev/null && go run ./cmd/rendercheck /tmp/rendercheck-avail`
Expected: exits 0; table check still passes (the Status column markup changed but column count did not). Open `/tmp/rendercheck-avail/table.html` and `siteAvailability.html` in a browser and confirm: verdict line above the table, token+percentage in every row, strip with 30 cells, incident list.

- [ ] **Step 4: Commit**

```bash
git add go-app/cmd/rendercheck/main.go
git commit -m "test(rendercheck): availability fixtures for all template branches"
```

---

### Task 13: Documentation — ADR flip, arc42 docs, README

**Files:**
- Modify: `documentation/adrs/0019-availability-monitoring-with-github-actions-prober.md`
- Modify: `documentation/arc42/chapters/05_building_block_view.adoc`
- Modify: `README.md`

- [ ] **Step 1: Flip ADR-0019 to Accepted, note the deviations**

Replace the `## Status` section body (`Proposed`) with:

```markdown
Accepted (2026-08-05), implemented with two deviations:

1. **Freshness heartbeat.** A `probe_run` table (one row per run) carries
   "last checked", instead of deriving it from the newest bucket
   timestamp: a bucket row is per-site and per-day, a heartbeat is
   per-run, and staleness is a fact about the run.
2. **Slack alerting deferred.** The honesty chain's layer 3 is not built
   in this iteration; layers 1 (static error panel) and 2 (derived
   staleness) are.
```

- [ ] **Step 2: Building block view**

In `05_building_block_view.adoc`, add a short subsection (match the file's existing heading level and style — read it first) describing: `cmd/probe` (batch job, GitHub Actions, every 15 min), `internal/availability` (read/write layer and computation), the three tables, and the reference to ADR-0019. Five to ten lines of prose, no diagram change required.

- [ ] **Step 3: README**

In `README.md`, in the "Local development" section after the `cp go-app/set-api-keys.sh.template …` paragraph, add:

```markdown
Availability is measured by a scheduled GitHub Actions workflow
(`.github/workflows/probe.yml`, every ~15 minutes, ADR-0019) that runs
`go-app/cmd/probe` and writes into Turso. It needs the repository secret
`TURSO_AUTH_TOKEN`. Locally: `make db-apply-dev` once, then `make probe`.
```

- [ ] **Step 4: Commit**

```bash
git add documentation/adrs/0019-availability-monitoring-with-github-actions-prober.md documentation/arc42/chapters/05_building_block_view.adoc README.md
git commit -m "docs: ADR-0019 accepted with deviations; probe in building blocks and README"
```

---

## Final verification (after all tasks)

- [ ] `cd go-app && go test ./...` — all green
- [ ] `cd go-app && go build ./...` and `golangci-lint run` (if installed locally; CI runs it regardless)
- [ ] rendercheck output eyeballed (Task 12 Step 3)
- [ ] Local end-to-end: `make db-apply-dev && make probe`, then `make backend` + `make site`, open `http://localhost:4000` — verdict line and Status column show real dev-DB data; open a site page — availability region fills
- [ ] Superpowers verification skill: run `superpowers:verification-before-completion` before declaring done

## Rollout (owner steps, after merge to main)

1. Apply the schema to prod: `atlas schema apply --url "<turso prod url with auth>" --to "file://go-app/internal/database/schema.hcl"` (the three new tables; Atlas reports exactly what it will create).
2. Set the repo secret `TURSO_AUTH_TOKEN` (Settings → Secrets → Actions).
3. Merge to `main` — the schedule only fires there. Trigger one `workflow_dispatch` run and check the Actions log and the page.
4. Deploy the Go service to fly.io (existing `fly.yml` on push to main does this).
