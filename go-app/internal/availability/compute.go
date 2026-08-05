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
