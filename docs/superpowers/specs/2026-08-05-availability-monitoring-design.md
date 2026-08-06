# Availability monitoring — design

Date: 2026-08-05
Status: approved by owner (this conversation); implements ADR-0019
Owner decisions folded in: adopt ADR-0019 as-is; fill the existing Status
column (no second table row); one status line per tile; detail page gets
strip + figures + incidents; **no Slack messaging in this iteration**.

## Goal

status.arc42.org measures and reports real availability for every hosted
arc42 property. Three surfaces show it: the intro table's Status column,
one line per tile, and an Availability section on each per-site page.
Cost target: €0/month, no external monitoring provider, data in the
project's own Turso database.

## Architecture (per ADR-0019)

```
GitHub Actions (cron */15)          the 10 hosted arc42 sites
        │  go run ./cmd/probe  ──────────► HTTP GET (concurrent)
        │            ├──► Turso: status_snapshot (row only on state change)
        │            ├──► Turso: status_bucket   (daily rollup upsert)
        │            └──► Turso: probe_run       (one heartbeat row per run)
        ▼
arc42-stats (fly.io, asleep by default)
        └── on request: internal/availability reads Turso,
            renders into /statsTable, /tiles, /siteAvailability
```

The prober is a batch job, not a service: wake, measure, record, exit.
Slack alerting (ADR-0019's layer 3) is **deferred**; the honesty chain in
this iteration is the static error panel (layer 1) and derived staleness
(layer 2).

## 1. Prober — `go-app/cmd/probe`

- Walks `types.Arc42properties`; probes every property with `Host != ""`
  and `!Planned` — currently 10 of 12 (skips `arc42-template`,
  `examples.arc42.org`).
- New field on `types.Property`: `ExpectedContent string` — per-site
  substring the response body must contain (e.g. `arc42`). The property
  list stays the single declaration of the family.
- One GET per site over concurrent goroutines, 10 s timeout.

| Outcome | State | `detail` |
|---|---|---|
| 2xx, latency ≤ 2000 ms, expected content present | `up` | — |
| 2xx, latency > 2000 ms | `degraded` | `slow: 3480ms` |
| 2xx, expected content missing | `degraded` | `content` |
| non-2xx, TLS error, DNS failure, timeout | `down` | `502` / `timeout` / … |

- A failure is confirmed 2-of-3 (attempts 5 s apart) before being
  recorded as `down`.
- Writes:
  - `status_snapshot`: one row **only on state change** — site, status,
    changed_at, response_ms, detail, monitor_id = `gha-<runner-region>`.
  - `status_bucket`: upsert today's row per site (`bucket_kind = "day"`),
    accumulating `downtime_minutes` (state-duration since previous run,
    capped at cycle length) and `outage_count` (increment on up→down
    transition).
  - `probe_run`: one append per run — the freshness heartbeat.
- Exit code 0 even when sites are down (a down site is a successful
  measurement). Non-zero only when the prober itself fails (Turso
  unreachable, no properties resolvable).

## 2. Schema addition — `probe_run`

New table in `internal/database/schema.hcl` (Atlas-managed, kept in sync
with Go manually per ADR-0014):

```hcl
table "probe_run" {
  column "run_at"        { type = datetime, null = false }
  column "vantage"       { type = varchar(64), null = false }
  column "sites_checked" { type = integer, null = false }
  column "duration_ms"   { type = integer, null = false }
}
```

~96 rows/day. All writes together (≈1000 bucket upserts + heartbeats per
day) stay far inside Turso's 10 M writes/month free allowance.
`status_snapshot` and `status_bucket` already exist in the schema and are
used unchanged.

## 3. Read path — `go-app/internal/availability`

New package; the API handlers call it. Per site it returns:

- **Current state** (latest snapshot row) and last-checked age (newest
  `probe_run.run_at`).
- **Uptime percentages** for 7 d / 30 d / 12 m, computed from daily
  buckets: `1 − downtime_minutes / covered_minutes`. A window not fully
  covered by measurement renders honestly:
  `12m n/a (measured since 2026-08)` — never a percentage computed over
  a shorter span than the label claims.
- **30 day-cells** for the strip: each day `up` (0 min down),
  `degraded/partial` (< threshold), `down` (above), `nodata`.
- **Recent incidents** (last ≈5): reconstructed from consecutive
  `status_snapshot` transitions — start, end (or "ongoing"), duration,
  state, detail.
- **Staleness**: when the newest `probe_run` is older than 3 cycles
  (~45 min), every surface renders `unknown` tokens plus
  "last checked X ago" instead of stale green.

Results are cached with the same approach as the existing statistics
cache (ADR-0015), so one page view costs one Turso read burst.

## 4. Display surfaces

### 4.1 Intro table (`/statsTable` fragment)

- Status column cell: status token + 30 d percentage (`● 99.98%`),
  linking to `/site/<key>/#availability`. Numeric `data-order` attribute
  so DataTables sorts it; the `orderable: false` exception for column 1
  in `home.md` is removed.
- The static "Availability is not measured yet" banner in `home.md` is
  replaced by a service-rendered family verdict at the top of the
  fragment: "All 10 monitored sites up · checked every ~15 min, last
  check 12 min ago", or the worst current state when something is down,
  or the staleness wording. The published copy states the ~15 min
  cadence so no reader infers more precision than exists.
- Rows out of the table / repo-only properties: unaffected.

### 4.2 Tiles (`/tiles` fragment)

One status line in the body of each hosted, non-planned tile:
`● up · 99.98% (30d)`. Repo-only (`arc42-template`) and planned
(`examples.arc42.org`) tiles are unchanged. No colour values in Go —
tokens are CSS classes keyed on state, per the existing contract.

### 4.3 Detail pages (`/siteAvailability` endpoint, new)

The ADR-19 placeholder in `docs/_includes/site-detail.html` becomes a
fourth htmx region wired exactly like the other three
(`window.arc42Status.wire`), fetching
`/siteAvailability?site=<key>` rendered by a new
`siteAvailability.gohtml`:

- current state + last-checked age,
- 30-day strip (day cells coloured by downtime; `nodata` cells for days
  before measurement began),
- 7 d / 30 d / 12 m figures,
- recent incidents (span, duration, probe detail).

Unknown site keys 404 before any template runs, same as
`/siteDetail`. For hostless properties (`arc42-template`) the include
renders no Availability region at all — there is no site to probe, and
an empty section would imply one; the endpoint still answers with a
"repository only — nothing to probe" note if asked directly.

### 4.4 CSS

Existing status tokens (`status-token--up/degraded/down/unknown/
unmonitored`) are reused everywhere. New small styles: the availability
strip (30 day-cells), the incident list, the tile status line, the table
status cell. No new hues for state — the existing token colours are the
palette.

## 5. Scheduling — `.github/workflows/probe.yml`

- `on: schedule: cron: '*/15 * * * *'` plus `workflow_dispatch` for
  manual runs.
- Steps: checkout → setup-go → `go run ./cmd/probe` (working dir
  `go-app`).
- Secrets: `TURSO_AUTH_TOKEN` (the Turso database URL is a compiled-in
  constant, not read from the environment). No Slack secret in this
  iteration.
- `concurrency: probe` with `cancel-in-progress: false` so runs never
  overlap.
- Known platform behaviour (accepted in ADR-0019): 5-min floor, delays
  under load, schedule disabled after 60 days of repo inactivity —
  surfaces as staleness, not silence.

## 6. Testing & local development

- `cmd/probe` classification: table-driven tests with `httptest` servers
  covering all four outcome rows and the 2-of-3 confirmation.
- `internal/availability`: pure-function tests for window math, partial
  coverage (`n/a since …`), strip cells, incident reconstruction, and
  staleness — from fixture rows, no network.
- `cmd/rendercheck`: extended with availability fixture data so the
  changed/new templates render-check like the rest.
- `make probe`: one prober run against the local dev DB, so the loop
  works end-to-end locally.

## 7. Documentation & rollout

- ADR-0019 → Accepted, with two noted deviations: `probe_run` heartbeat
  instead of bucket-timestamp freshness, Slack alerting deferred.
- Short update to the arc42 docs under `/documentation`.
- Rollout: (1) owner applies `go-app/internal/database/schema.hcl` to
  production Turso via atlas — the workflow's first run fails with
  `log.Fatal` if the tables don't exist yet; (2) set the single repo
  secret `TURSO_AUTH_TOKEN`; (3) merge → workflow starts on its schedule
  → all three surfaces show real state from the first run, with
  `n/a (since 2026-08)` for windows not yet filled.

## Out of scope (this iteration)

- Slack alerting on transitions (ADR-0019 layer 3).
- Second probe vantage; TLS-expiry checks; sub-15-min resolution.
- Availability for repo-only or planned properties.
