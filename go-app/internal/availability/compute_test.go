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
	buckets[9].DowntimeMinutes = 30  // today: partial
	buckets[4].DowntimeMinutes = 500 // beyond PartialDayMaxDownMinutes: down
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
