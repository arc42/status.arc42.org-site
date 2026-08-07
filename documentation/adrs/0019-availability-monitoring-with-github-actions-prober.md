# 19. availability monitoring with cron-job.org prober endpoint

Date: 2026-08-04 (Updated 2026-08-06)

## Status

Accepted (2026-08-05), updated to **Option A** (2026-08-06):

1. **Deterministic Cron Trigger via cron-job.org:** Replaced GitHub Actions cron with an external cron-job.org schedule calling `POST /api/probe` every 15 minutes. This eliminates GitHub Actions runner queue delays and prevents false "stale" warnings.
2. **Visitor Cold-Start Elimination:** Every 15 minutes, the incoming `cron-job.org` HTTP ping automatically wakes/warms the Fly.io machine, providing instant responses for human visitors.
3. **Freshness heartbeat.** A `probe_run` table (one row per run) carries "last checked", maintaining the honesty chain.
4. **Slack alerting active on availability failure.** When a domain or subdomain availability check fails (state `down`), the prober sends a Slack notification.
5. **meta.arc42.org is excluded from probing.** Skipped via `types.Property.NoProbe`.

## Decision

**An external cron schedule on cron-job.org calls `POST /api/probe` on the Go backend every 15 minutes with a Bearer secret key. The Go endpoint runs `internal/probe`, writes availability transitions into Turso, and alerts Slack on outage.**

### Architecture

```
cron-job.org (cron */15)              the 9 arc42 sites
        │                              (GitHub Pages, Netlify)
        │  POST /api/probe  ──────────────► HTTP GET
        │  (Bearer token)                      │
        ▼                                      └── compares to last known state
   arc42-stats (fly.io)                             │
        ├── wakes & stays warm                      ├──► Turso: status_snapshot (only on change)
        ├── runs internal/probe                     ├──► Turso: status_bucket   (daily rollup)
        └── returns 200 OK                          ├──► Turso: probe_run       (heartbeat, every run)
                                                    └──► Slack: on availability failure (site down)
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
