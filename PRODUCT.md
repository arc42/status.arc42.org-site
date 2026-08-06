# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

**Primary: Gernot Starke and the arc42 maintainers.** They open status.arc42.org to answer maintenance questions about the arc42 site family — which site is growing or fading, which repo has an issue backlog, whether anything is down. This is a working check-in, not a browse; scanning density and truthful signal outrank persuasion.

**Secondary: the arc42 public and sponsors (INNOQ).** The page is public and should read as credible and alive to a stranger who lands on it, but no public need may compromise the maintainer view.

## Product Purpose

A single public page that shows the operational and usage health of every arc42 web property in one look: visitor and pageview statistics from Plausible, repository health from GitHub, and — not yet built — real availability/uptime.

Success is that a maintainer gets the answer they came for without leaving the first screen, and that anything wrong is visible before they go looking for it.

## Positioning

Unlike a generic hosted status page, this one is *owned*: it merges three signals no vendor combines — visitor analytics (Plausible), repository maintenance load (GitHub), and site availability — across a family of related sites, and it renders them from an architecture that is itself a demonstration piece (see Brand Commitments).

## Operating Context

- Public URL: `https://status.arc42.org`. Statistics fragment served from `https://arc42-stats.fly.dev/statsTable`.
- Current implementation is two-part: a Jekyll site (GitHub Pages, minimal-mistakes remote theme, `splash` layout) under `docs/`, and a Go service under `go-app/` deployed to fly.io that renders `internal/api/arc42statistics.gohtml` and is swapped into the page by htmx (`hx-get`, `hx-trigger="load delay"`, `hx-swap="outerHTML"`).
- Because of that swap, the page is visibly empty-then-full on every visit; the placeholder text is part of the real experience, not an edge case.
- Data sources: Plausible API (visitors, pageviews at 7d/30d/12m), GitHub GraphQL (open issues, bugs, PRs), TursoDB (libSQL) for storage, Slack for notifications, fly.io for hosting.
- Sorting of the statistics table is done client-side by DataTables (jQuery).
- Technical documentation lives in `documentation/` as an arc42 document built with docToolchain.

## Capabilities and Constraints

**Today**

- One sortable table: 8 sites × (visitors, pageviews) × (7d, 30d, 12m), plus open issues and bugs per repo, plus a totals row.
- A provenance footer: app version, collection duration, timestamp, fly.io region.
- Six embedded Plausible dashboards (1600px iframes) in two grouped sections, reachable from an anchor list.
- Sites currently tracked (`go-app/internal/types/types.go`): arc42.org, arc42.de, docs.arc42.org, faq.arc42.org, canvas.arc42.org, quality.arc42.org, status.arc42.org, pdfminion.arc42.org.

**Confirmed scope going forward**

- **Real availability/uptime.** Current up/down per site plus condensed history for 7d/30d/12m (planned as red/yellow/green bars). Planning notes exist in `claude-plan.md`, `updated-claude-plan.md`, and the sibling ChatGPT/Gemini plans (Aug 2025); an external monitor such as BetterStack or UptimeRobot was the assumed source. Not implemented.
- **Trends, not only levels.** Direction over time per site, not three static number pairs.
- **Repo/project health.** Deeper GitHub signal per site: issues, bugs, PRs, staleness, last activity.
- **Better presentation of existing data**, independent of any new source.
- **A second "dashboard" page**: one tile per property, each tile carrying a "New Issues and PRs" section and a short "repo health + visitor trend" summary. Per-repo detail subpages may follow later; the tile must be designed so a repo *without* visitor statistics is a first-class variant.
- **Coverage:** live web properties first — the 8 above plus `trainings.arc42.org` (repo `trainings.arc42.org-site`). Tooling repos with maintenance load but no visitor stats (arc42-template, arc42-generator, PDFminion, publishToConfluence) come next.

**Constraints**

- The embedded Plausible dashboards must remain reachable on the site; confirmed 2026-08-04 that "reachable" means **one full-width dashboard per site on its own subpage, linked from that site's row/tile** — not six embeds expanded on the primary surface.
- Running cost must stay free-tier or near-zero. The page has low traffic and does not justify spend; this gates the choice of uptime-monitoring service.
- Plausible values can be unavailable; the data model already carries `"n/a"` strings rather than numbers for that case, and totals count such values as 0. Missing data is a normal state, not an error state.

**Open decisions**

- The delivery shell is undecided: keep Jekyll + minimal-mistakes, keep Jekyll but drop the theme for own layouts, or serve the whole page from the Go app. To be decided when the design direction is clearer. The trade-off recorded at interview time: dropping the theme buys design control at equal deploy cost; serving from Go simplifies the model but makes the entire page depend on fly.io uptime, which is awkward for a status page.
- Which uptime-monitoring provider to use.

## Brand Commitments

- Name and logo: arc42. The page belongs to the arc42 family and must be recognizable as such (logo, colors), but is explicitly allowed its own, more instrumented character than the content sites — it is not required to match arc42.org or docs.arc42.org closely.
- **The Go + htmx backend is a deliberate showpiece.** Gernot values it as a working example of a backend/frontend architecture combination and as a place to demonstrate cloud building blocks (fly.io, TursoDB, Slack integration). It is a strong preference rather than an absolute must, and alternatives may be proposed — but replacing it costs something real that a replacement has to earn.
- Supported by INNOQ; content is CC-BY-SA 4.0. Author: Dr. Gernot Starke, Cologne.

## Evidence on Hand

- Live production site and live statistics API (URLs above) — real numbers, no need to invent data.
- Real per-site Plausible share links with auth tokens, already embedded in `docs/_pages/home.md`.
- Real repository inventory via the `arc42` GitHub org.
- `documentation/images/` holds the repo-overview diagram, the INNOQ support badge, and the CC-BY-SA badge.
- No testimonials, benchmarks, SLA figures, pricing, or uptime history exist. Uptime history in particular must not be fabricated or mocked as if real — none is being collected yet.

## Product Principles

1. **Maintainer answer first.** Every layout decision is judged by whether a maintainer gets their answer in the first screen; public credibility is a by-product of that, never a trade against it.
2. **Signal over inventory.** The page should surface what changed or what needs attention, not merely list every number it can fetch.
3. **Honest about absence.** Unavailable stats, unmonitored sites, and unbuilt uptime history are shown as what they are; nothing is presented as measured that is not measured.
4. **The architecture is part of the product.** Server-rendered fragments over htmx are a demonstrated position, not an implementation detail — solutions that abandon it must justify themselves.
5. **Cheap to run, cheap to keep.** Low traffic means near-zero cost and low maintenance burden are permanent requirements, not startup constraints.

## Accessibility & Inclusion

No product-specific standard was established. General web accessibility applies; note that the current data table is dense and its group headers ("7 Days" / "30 Days" / "12 Month" over paired columns) rely on visual structure alone.
