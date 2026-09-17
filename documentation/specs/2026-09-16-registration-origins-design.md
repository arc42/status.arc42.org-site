# Registration origins — design

Date: 2026-09-16
Status: draft, awaiting owner review
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
  "date_range": "30d",
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
start**. For one window (30 days by default) three short tables, one per
dimension above, each showing the dimension value with visitors and visits,
highest first, limited to the top rows.

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
| Window predates the join | rows shown, with the not-whole label above them |

## Risk to retire first

`has_done` and `has_not_done` arrived in plausible/analytics **v3.0.0**. Whether
arc42's account serves them cannot be determined from the repository or from
the public docs — only by asking the API. **Task 1 of the implementation is a
throwaway probe** that sends the query above and reports the status code and
body. If it is refused, the feature stops there and the fallback is documented
recipes for the Plausible UI, which was the third option the owner weighed.

## Code shape

- `internal/plausible/queryV2.go` — new. A small typed client: build request,
  set the bearer header, POST, decode `results`/`meta`, map errors. It knows
  nothing about registrations.
- `internal/plausible/registrationOrigins.go` — new. Builds the three queries,
  calls the client, returns a typed result per dimension.
- `internal/types` — the result types, and the window's not-whole flag derived
  from the existing `RollupSince` of `trainings.arc42.org` rather than a
  second hard-coded date.
- `internal/api/rollupPage.gohtml` — the new section, using the page's existing
  class vocabulary; no new stylesheet, since the rollup page loads the site's.
- `cmd/rendercheck` — fixture rows for the new section, so the absolute-link
  and rendering checks cover it.

The collection cadence is the existing one: these queries run in the same
cached collection run as the rollup figures, not per request.

## Testing

- The query builder: exact JSON for each of the three dimensions, asserted
  against the literal bodies in this document.
- The decoder: a recorded response mapped to typed rows, including the
  positional dimension/metric mapping and an empty `results` array.
- Error paths: non-200, malformed body, refused `has_done`.
- The not-whole rule: a window starting before 2026-09-15 is flagged; one
  starting after is not.
- No test calls the live API.

## Out of scope

Exit pages, the family-wide entry report, funnels (Plausible exposes none via
this API), and anything resembling a per-person path.
