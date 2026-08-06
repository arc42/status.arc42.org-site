# 19. availability monitoring with a GitHub-Actions prober

Date: 2026-08-04

## Status

Accepted (2026-08-05), implemented with three deviations:

1. **Freshness heartbeat.** A `probe_run` table (one row per run) carries
   "last checked", instead of deriving it from the newest bucket
   timestamp: a bucket row is per-site and per-day, a heartbeat is
   per-run, and staleness is a fact about the run.
2. **Slack alerting active on availability failure.** When a domain or subdomain
   availability check fails (state `down`), `cmd/probe` sends a Slack notification.
3. **meta.arc42.org is excluded from probing.** A new `types.Property.NoProbe`
   field skips it: its DNS entry does not exist (NXDOMAIN, checked
   2026-08-05), and a probe against a name that does not resolve would
   record a permanent outage for a site that is not down but simply not
   built — "not built" must never render as "outage". Separately,
   `docs.arc42.org` and `faq.arc42.org` are probed at `/home/`
   (`types.Property.ProbePath`) rather than at their root, because their
   roots serve a meta-refresh stub with no content to assert against.

## Context

status.arc42.org is named for a thing it does not do: it reports usage and repository
statistics, but no availability. The 2025 planning round assumed an external monitoring
service (BetterStack, UptimeRobot) and left `monitor_id` in the `status_snapshot` table
as a hook for it. Two facts make that assumption worth revisiting:

1. **Cost.** A paid monitoring plan is not justified by this site's traffic. The
   requirement is free or near-zero, and preferably open-source-compatible so the data
   stays ours.
2. **The Go app cannot probe.** `fly.toml` sets `auto_stop_machines = true` and
   `min_machines_running = 0`. The statistics service is asleep unless someone is
   looking at the page, so an in-process ticker would only observe the moments when a
   visitor happens to be present — precisely the moments least in need of monitoring.

A prober must therefore live outside the fly.io app. It must also live outside the
monitored sites themselves, which are heterogeneously hosted (GitHub Pages for most
subdomains, Netlify for arc42.org), so no single platform's own health signal covers
the family.

Constraints verified 2026-08-04:

- GitHub Actions enforces a **5-minute minimum** `schedule.cron` interval, gives
  **unlimited free minutes to public repositories** (this repo is public), and
  **silently disables schedules after 60 days without repository activity**.
- Turso's free plan allows 5 GB storage, 500 M row reads and **10 M row writes per
  month**. Even a naive design writing every sample for 9 sites every 5 minutes
  (~2.3 M writes/month) stays inside it; the transition-based design below writes
  roughly a thousand rows a month.

## Decision

**A small Go program in this repository, `cmd/probe`, run by a scheduled GitHub Actions
workflow every 15 minutes, writes availability transitions into the existing Turso
tables. The statistics service only reads and renders them.**

### Architecture

```
GitHub Actions (cron */15)          the 9 arc42 sites
        │                            (GitHub Pages, Netlify)
        │  go run ./cmd/probe  ──────────► HTTP GET
        │            │                        │
        │            └── compares to last known state
        │                         │
        │                         ├──► Turso: status_snapshot (only on change)
        │                         ├──► Turso: status_bucket   (daily rollup)
        │                         ├──► Turso: probe_run       (heartbeat, every run)
        │                         └──► Slack: on availability failure (site down)
        ▼
   arc42-stats (fly.io, asleep by default)
        └── on request: reads Turso, renders the status column into the htmx fragment
```

The prober is deliberately *not* a service. It is a batch job that wakes, measures,
records, and exits — which is why it costs nothing and has nothing to keep running.

### Probe semantics

Per site, one GET with a 10-second timeout:

| Outcome | State | `detail` |
|---|---|---|
| 2xx, latency ≤ 2000 ms, expected content present | `up` | — |
| 2xx, latency > 2000 ms | `degraded` | `slow: 3480ms` |
| 2xx but expected content missing | `degraded` | `content` |
| non-2xx, TLS error, DNS failure, or timeout | `down` | `502` / `timeout` / … |

The content assertion is a per-site expected substring (e.g. arc42.org must contain
`arc42`), which catches the "serves 200 but the build broke" case that pure ping
monitoring misses. TLS expiry is **not** checked: all domains use Let's Encrypt with
automated renewal.

A failure is confirmed before it is recorded: **2 of 3 attempts**, 5 seconds apart, must
fail before a site is declared `down`. This keeps a single dropped packet from writing
an incident.

### Storage

Uses the tables already declared in `internal/database/schema.hcl`:

- **`status_snapshot`** — one row per *state change*, not per sample: `site`, `status`,
  `changed_at`, `response_ms`, `detail`. The `monitor_id` column, added for an external
  provider, is repurposed to identify the probe vantage point (`gha-<runner-region>`),
  which keeps the door open for a second vantage later.
- **`status_bucket`** — one row per site per day: `downtime_minutes`, `outage_count`.
  Written by the same job, upserting the current day. This is what the history bars read,
  so rendering never scans the event log.
- **`probe_run`** — one row per *run*, not per site: `run_at`, vantage, sites checked,
  duration. Added per Status deviation 1, as the freshness heartbeat (see below).

All three tables are append-mostly and tiny, but `status_bucket` is the dominant one:
every run upserts each site's daily bucket, so it is 9 sites × 96 runs ≈ 864 bucket
upserts a day, ~26,000 a month — a per-run cost, not a per-day one. `status_snapshot`
adds only a handful of transition rows on top. `probe_run` adds one row every run
regardless of outcome: 96 a day, ~2900 a month. The combined total, on the order of
30,000 writes a month, is still roughly 0.3% of the 10 M allowance — negligible.

### Freshness is derived, never asserted

The page computes staleness from the newest `probe_run` row (`run_at`), not from the
newest bucket timestamp — see Status deviation 1. A bucket is a per-site daily rollup,
so on a quiet day with no transitions its timestamp would go stale even while the
prober is running fine; `probe_run` gets one row per run regardless of outcome, so it
is the true heartbeat. If the workflow stops running — GitHub disabled the schedule, a
secret expired, the job broke — `probe_run` stops advancing and the page says so
("last checked 4 hours ago") instead of showing a stale green. **No separate heartbeat
_service_ is needed: the heartbeat is a row the batch job already writes, not a second
process to keep alive; absence of new rows is itself the signal.**

This is designed as a three-layer honesty chain, in which nothing can fail silently.
As implemented (see Status), only layers 1–2 are built; layer 3 is deferred:

1. The static Jekyll page renders an error panel when the fly.io app is unreachable
   (ADR-relevant: this is why the shell stays static — a page served by the app could
   not report the app being down).
2. The app renders `stale` when the probe data has stopped advancing.
3. Slack alerts when an availability check fails (`down`), so the maintainer
   learns of an incident without visiting the page.

### Cadence

`*/15 * * * *` — 96 runs per day, ~30 s each, free on a public repo. Fifteen minutes is
a deliberate choice, not a limit: GitHub's floor is 5 minutes, but scheduled runs are
delayed under load, and a status page for documentation sites gains nothing from
minute-level resolution. The published copy states the cadence, so no reader infers more
precision than exists.

## Consequences

- **€0/month.** No new provider, no new machine, no volume, no plan to outgrow.
- **The prober is independent of everything it watches** — outside fly.io, outside
  GitHub Pages, outside Netlify.
- **The stack stays the showpiece stack.** Go, goroutines for concurrent probing, Turso,
  GitHub Actions, and — once Status deviation 2 closes — Slack: the probe is a compact,
  readable example of exactly the architecture this project exists to demonstrate.
- **Single vantage point.** All probes originate from one GitHub-hosted runner region,
  so a network problem between that region and a site reads as a site outage. Mitigated
  by 2-of-3 confirmation; a second vantage can be added later without a schema change.
- **Resolution is 15 minutes, and delays are possible.** Outages shorter than one cycle
  can be missed entirely. The page must never claim more than "checked every ~15 min".
- **The 60-day inactivity rule applies.** If the repository goes quiet for two months,
  GitHub disables the schedule — which surfaces as growing staleness on the page rather
  than as silence.
- Secrets configured on the workflow: `TURSO_AUTH_TOKEN` and `SLACK_AUTH_TOKEN`.
- `schema.hcl` and the Go code must stay in sync manually (see ADR-0014).

## Alternatives considered

**Always-on fly.io machine** (`min_machines_running = 1`, goroutine ticker, 60-second
resolution). The best resolution and the simplest code, at roughly €2/month — but it
puts the monitor inside the system being monitored, and `status.arc42.org` is itself one
of the nine monitored sites. Rejected for independence, not for cost. If minute-level
resolution is ever wanted, this becomes the natural upgrade and the schema does not
change.

**Upptime** (GitHub-Actions-based, open source). Does essentially this decision as a
finished product, with incident issues and history in git. Rejected because it brings its
own status page and its own data layout, duplicating the site and the Turso database this
project already has; adopting it would mean maintaining an integration instead of ~150
lines of Go.

**Gatus** (open source, YAML-declared checks, richer semantics including TLS expiry and
conditional assertions). The most capable option, and philosophically close to this
project. Rejected because it needs its own always-on host and volume — the same cost as
the fly option, plus a second service to operate, for capability beyond what nine
documentation sites require.

**Uptime Kuma** (open source, click-to-configure). Rejected: configuration lives in its
own database rather than in this repository, and it needs a persistent host.

**UptimeRobot / BetterStack free tiers.** Genuinely free at this scale and zero
maintenance, but proprietary, with the availability history living in someone else's
account and leaving on their terms. Rejected on ownership, which is the same reason the
statistics themselves are collected here rather than embedded.
