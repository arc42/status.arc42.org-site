# Registration origins — design

Date: 2026-09-16
Status: implemented (2026-09-18, ADR-0023); extended to the German courses on arc42.de by ADR-0024 (2026-09-19)
Supersedes nothing. Implements the "journeys page" left open by ADR-0021.

## The question

> From which entry page and which domain do the people who reach the trainings
> registration come?

Concretely: of the visits that opened `trainings.arc42.org/registration/`, where
did the visit *start* — which host, which page, which source.

## What cannot be answered, and why

Plausible records no page sequences and never identifies a person across visits.
There is no report, here or on plausible.io, that can show "the ten most common
paths through arc42", and none that can show what an individual did before
registering. The rollup page already says this; this design does not change it.

What the rollup *can* answer is where a visit began, because entry page, entry
hostname and source are properties of the visit as a whole. Since every member
site reports into `rollup.arc42.com`, a visit that starts on docs.arc42.org and
ends on the registration form is **one** visit there, with docs as its entry
page. That is the whole reason this question is answerable at all, and it is
answerable only in the rollup — never in the trainings site's own dashboard,
where the same journey looks like a fresh visit referred by docs.

## Precondition, already met

`trainings.arc42.org` reports into the rollup, registration page included:

```
data-domain="trainings.arc42.org,rollup.arc42.com"
```

It joined on **2026-09-15** (commit `a429902`). Windows reaching back before
that date see a registration page that was not yet in the rollup, so they
under-report. The feature must not present such a window as if it were whole.

## Data source

Plausible **Stats API v2**, which the vendored client cannot reach:
`go-plausible v0.3.1` exposes only `event:page`, `visit:source`,
`visit:referrer`, `utm_*`, `device`, `browser`, `os`, `country`. It has no
`entry_page`, no `exit_page` and no `hostname`. So this feature talks to the
API directly over HTTP; the library stays where it is for everything else.

- Endpoint: `POST https://plausible.io/api/v2/query`
- Auth: `Authorization: Bearer $PLAUSIBLE_API_KEY` — the token the collector
  already uses, no new secret
- `site_id`: `rollup.arc42.com`

### The filter that makes it correct

An event-level filter selects *matching events only*, so filtering
`event:page == /registration/` and grouping by `visit:entry_page` would return
the entry pages of the registration pageviews themselves — nonsense. Selecting
whole visits that contained a registration pageview needs the behavioural
operator `has_done`, whose shape is fixed by Plausible's own query schema
(`priv/json-schemas/query-api-schema.json`): a two-item array of the operator
and a nested filter tree.

```json
{
  "site_id": "rollup.arc42.com",
  "metrics": ["visitors", "visits"],
  "date_range": "all",
  "dimensions": ["visit:entry_page_hostname"],
  "filters": [["has_done", ["is", "event:page", ["/registration/"]]]],
  "order_by": [["visitors", "desc"]],
  "pagination": {"limit": 25, "offset": 0}
}
```

Three queries per window, differing only in `dimensions`:

| Dimension | Answers |
|---|---|
| `visit:entry_page_hostname` | which arc42 site the visit came in through |
| `visit:entry_page` | which page it came in through (path only — see traps) |
| `visit:source` | where the visit came from before arc42 at all |

Response rows map positionally: `results[].dimensions` follows the query's
`dimensions`, `results[].metrics` follows its `metrics`.

## What is shown

A new section on `/rollup`, below "The numbers", headed **Where registrations
start**. For one window (all time in the rollup) three short tables, one per
dimension above, each showing the dimension value with visitors and visits,
highest first, limited to the top rows.

Since ADR-0024 (2026-09-19) the section holds one block per course site. This
spec describes the English block; the German block reports arc42.de's
`/anmeldung/` from arc42.de's own dashboard over the last 12 months, with two
cuts (entry page and source), because arc42.de is not in the rollup.

Above them, one sentence naming the filtered page and the window, so no figure
is readable without knowing what it counts.

## Honest reporting rules

These follow ADR-0002 (report what was measured) and match how the rollup
table already behaves:

1. **A window that reaches back before 2026-09-15** is labelled as not whole,
   naming the join date, exactly as the comparison table labels joiners.
2. **`visit:entry_page` is a path without a host.** `/` from four member sites
   is one row. The entry-page table therefore always sits *below* the
   entry-hostname table, and carries that warning inline.
3. **A query that fails reports as failed** — no zeroes standing in for
   unknown. An empty result set says "no visit in this window reached
   /registration/", which is a different statement from a failure.
4. **Nothing is cross-multiplied.** Visitors in one dimension's table are not
   added to another's; they are the same people counted three ways.

## Failure modes

| Case | Behaviour |
|---|---|
| `PLAUSIBLE_API_KEY` unset | section omitted; the rest of the page renders |
| API returns non-200 | section shows the failure and the status code |
| `has_done` unsupported by the account's plan | section shows that the query was refused, with the API's message — **see risk below** |
| No rows | "no visit in this window reached the registration page" |
| Every cut failed | each cut shows its failure; no visit count and no small-sample sentence, because a zero is never shown for unknown |

## Risk retired (probed 2026-09-17)

`has_done` arrived in plausible/analytics **v3.0.0**, and whether arc42's
account served it could only be settled by asking. It was asked: the query
above returned **HTTP 200**, so the operator works and this design stands. The
same probe confirmed the filtered path is exactly `/registration/` — Plausible
strips the `?kurs=…` query string, so one filter covers every course.

## What the data looks like today, and what follows from it

The probe also measured the sample, and it is very small. All-time in the
rollup, `/registration/` has **3 page views from 2 visitors**, and each of the
three dimensions returns a single row: entry hostname `arc42.org`, entry page
`/`, source `Google` — 2 visitors in each.

That is not a defect. `trainings.arc42.org` joined the rollup on 2026-09-15, so
the rollup has known about the registration page for two days. The figures will
fill out over the coming weeks without anyone touching the code.

It does bind the design, though. A page that prints three tables of one row
each, under a 30-day heading, invites the reader to believe a month was
measured when two days were. Therefore:

- The default window is **all time in the rollup**, not 30 days. A 30-day label
  over two days of membership is the misreading to avoid.
- Every table states the sample it rests on — visitors and visits behind it —
  and the section names trainings' join date, so a single-row table reads as
  "two visits so far", never as "everyone arrives via Google".
- Below a threshold of **20 visits**, the section leads with a plain sentence
  saying the sample is too small to generalise from, and the tables follow it.
  This is the same instinct as the comparison table's "not comparable yet".

## Code shape

- `internal/statsv2` — new package, not `internal/plausible`. Package
  `plausible`'s pre-existing `init()` calls `os.Exit(13)` when
  `PLAUSIBLE_API_KEY` is unset, so any test placed there would make
  `go test ./...` require a live secret; the v2 code needs nothing else from
  the v1 package, so it stands apart.
  - `queryV2.go` — a small typed client for the Stats API v2
    (`V2Query`, `V2Row`, `V2Pagination`, `RunV2Query`): builds the request, sets
    the bearer header, POSTs, decodes `results`, maps errors. It knows nothing
    about registrations.
  - `registrationOrigins.go` — `RegistrationOriginsFor(joinedOn string)` builds
    the three queries against site `rollup.arc42.com`, filter
    `[["has_done", ["is", "event:page", ["/registration/"]]]]`, date_range
    `"all"`, dimensions `visit:entry_page_hostname`, `visit:entry_page`,
    `visit:source`; calls the client; returns a typed result per dimension. An
    empty `PLAUSIBLE_API_KEY` sends no request and returns nil `Cuts`, so the
    section is omitted.
- `internal/types` — `OriginRow`, `OriginCut`, `RegistrationOrigins`, and the
  constant `SmallSampleVisits = 20`; fields on `Arc42Statistics` and
  `RollupPageData`.
- `internal/domain/domain.go` — the three queries run in the same cached
  collection pass as the rollup figures, not per request; the join date comes
  from `trainings.arc42.org`'s `RollupSince` in the roster
  (`"2026-09-15"`) rather than a second hard-coded date.
- `internal/api/rollupPage.gohtml` — the section "Where registrations start",
  placed after "The numbers"; omitted entirely when there are no cuts; the
  small-sample sentence below 20 visits; a failed cut and an empty cut read
  differently, using the page's existing class vocabulary.
- `cmd/rendercheck` — a fixture that mirrors production (one row per
  dimension), plus a second variant, `rollupPage-noregistrations.html`, that
  exercises every cut present but empty: no visit in the rollup has reached
  the registration page yet.

Measured on 2026-09-17: `has_done` returned HTTP 200; `/registration/` had 3
page views from 2 visitors all-time in the rollup, and each cut returned one
row (`arc42.org`, `/`, `Google`).

## Testing

- The query builder: the exact set of dimensions sent - one query per
  dimension, matching the literal bodies in this document - and metrics
  `["visitors","visits"]` in that order.
- The decoder: a recorded response mapped to typed rows, including the
  positional dimension/metric mapping (checked with asymmetric metrics, so
  visitors and visits cannot be swapped unnoticed), an empty `results` array,
  and a response with no `results` key at all - the latter is an error, not
  an empty result.
- Error paths: non-200, malformed body, refused `has_done`, the token never
  appearing in an error.
- A cut that fails while its siblings succeed: the surviving cuts keep their
  rows and drive TotalVisits; the failed cut carries no rows and stays
  separate from an empty result.
- No zero for unknown: when every cut fails, TotalVisits is 0 for lack of an
  answer, and SmallSample must not fire on it - "too small to generalise
  from" only ever qualifies a total that came from real data.
- The not-whole rule has no test of its own: the window is fixed at all time
  (see "Precondition, already met" above), so there is no window boundary
  left to flag - the join-date sentence shown beside the tables carries that
  statement instead.
- No test calls the live API.

## Out of scope

Exit pages, the family-wide entry report, funnels (Plausible exposes none via
this API), and anything resembling a per-person path.
