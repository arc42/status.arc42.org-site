# 23. report where registrations start

Date: 2026-09-18

## Status

Accepted. Takes up the "journeys page" that ADR-0021 left open, for the question of
where registrations start.

## Context

ADR-0021 asked whether the arc42 sites could be described as one traffic family rather
than as separate dashboards, because `rollup.arc42.com` gives every member's visitors a
single shared identity. It left one question unanswered: from which entry page, and
which domain, do the people who reach the trainings registration come?

Plausible records no page sequences and never identifies a person across visits. There
is no report, here or on plausible.io, that shows the ten most common paths through
arc42, or what an individual did before registering, and none is being built now. What
Plausible does record is properties of a visit as a whole: its entry page, its entry
hostname, its source. Since every member site reports into the rollup besides its own,
a visit that starts on docs.arc42.org and ends on the registration form is **one**
visit there, with docs as its entry page. That is the only reason the question is
answerable at all, and it is answerable only in the rollup: in the trainings site's own
dashboard the same visit looks like a fresh one, referred by docs.

Selecting the right visits is not a plain page filter. An event-level filter
(`event:page == /registration/`) selects matching *events*, so grouping by entry page
would return the entry pages of the registration pageviews themselves - always
`/registration/` - which answers nothing. The visits that matter are the ones that
*contain* a registration pageview anywhere in them, which needs Plausible's behavioural
operator `has_done`. `has_done` reached the account's plan in analytics v3.0.0, and
whether arc42's own account served it could only be settled by asking: probed on
2026-09-17, the query returned HTTP 200. The same probe showed Plausible strips the
`?kurs=...` query string, so filtering the one path `/registration/` covers every
course.

`has_done` and the dimensions this needs - `visit:entry_page_hostname`,
`visit:entry_page`, `visit:source` - are Stats API v2 only. The vendored `go-plausible`
client speaks v1, which has no entry-page or hostname dimension at all.

trainings.arc42.org joined the rollup on 2026-09-15. All-time in the rollup, on
2026-09-17, `/registration/` had 3 page views from 2 visitors, one row per dimension.
A page that prints one-row tables under a 30-day heading invites the reader to believe
a month was measured when two days were.

## Decision

- The rollup page grows a section, "Where registrations start", below "The numbers",
  shown only to maintainers (ADR-0022) since it lives on `/rollup`.
- Three cuts of the same visits, differing only in the grouping dimension:
  `visit:entry_page_hostname` (which site), `visit:entry_page` (which page, path only -
  no host, so `/` from four member sites is one row), `visit:source` (where the visit
  came from before arc42). Nothing is cross-multiplied: the three tables count the same
  people three ways, not three populations.
- The window is **all time in the rollup**, not a rolling 30 days. A short label over a
  young join date is the misreading to avoid, and the section names trainings' join
  date so a reader can see how young the data still is.
- Below **20 visits** total, the section leads with a plain sentence saying the sample
  is too small to generalise from; the tables still follow, because they are what was
  measured.
- A query that fails is reported as failed, with the status or error, never as a zero
  standing in for unknown. An empty result set says "no visit reached the registration
  page yet", which is a different statement from a failure. With
  `PLAUSIBLE_API_KEY` unset, no request is sent and the section is omitted entirely
  rather than shown empty.
- The v2 client lives in its own package, `internal/statsv2`, not in package
  `plausible`. Package `plausible`'s `init()` calls `os.Exit(13)` when
  `PLAUSIBLE_API_KEY` is unset, so any test placed there would make `go test ./...`
  require a live secret; the v2 code needs nothing else from the v1 package.
  `queryV2.go` is a small typed client (`RunV2Query`) that knows nothing about
  registrations; `registrationOrigins.go` builds the three queries and shapes the
  result.
- The join date is read from the roster's existing `RollupSince` for
  `trainings.arc42.org` (ADR-0021), not a second hard-coded date.
- The three queries run in the same cached collection pass as the rollup figures, not
  per page request.

## Consequences

- Three extra Plausible API calls per collection run, on top of ADR-0021's three.
- The report stays thin until the window fills: today it is three one-row tables, and
  the small-sample sentence says so plainly rather than letting a single row read as a
  finding.
- `visit:entry_page` carries no host name, so the entry-page table alone cannot say
  which site a path belongs to; it is read together with the entry-hostname table
  above it, not on its own.
- The Stats API v2 client is a second, separate package from the vendored v1 library,
  so the service now has two ways of talking to Plausible. Package `plausible`'s
  exit-on-missing-key behaviour was the reason, not a preference; a future v2 need can
  extend `internal/statsv2` rather than adding a third package.
