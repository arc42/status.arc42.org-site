# 24. report where German registrations start

Date: 2026-09-19

## Status

Accepted. Extends ADR-0023 from the English courses to the German ones.

## Context

ADR-0023 answers where the visits that reach trainings.arc42.org's `/registration/`
begin. The German courses register elsewhere: on arc42.de, at `/anmeldung/?kurs=...`,
linked from `/termine/`. The owner wants the same question answered for them.

arc42.de does not report into the rollup. Its snippet names only its own dashboard
(`data-domain="arc42.de"`), which keeps the separate domain out of the family's shared
count; whether it should join is an open decision in meta.arc42.org (ADR-0005). So the
rollup query of ADR-0023 cannot see a single German registration.

arc42.de's own dashboard can still answer most of the question. `has_done` works there
as it does in the rollup, and Plausible strips the query string, so the one path
`/anmeldung/` covers every course. Probed on 2026-09-19 over twelve months: 235 page
views of `/anmeldung/` from 181 visitors; the visits that reached it began mostly on
`/termine/`, `/` and the course info pages, and came from direct traffic, Google,
arc42.org, isaqb.org and docs.arc42.org.

What the own dashboard cannot give is the page on another arc42 site where such a visit
began. A reader who moves from docs.arc42.org to arc42.de starts a new visit there, with
docs.arc42.org as its source. The domain they came from is visible; the page is not.

Unlike trainings, arc42.de's dashboard holds years of history, so ADR-0023's all-time
window no longer guards against anything - it would mix in registrations from older
versions of the site.

## Decision

- The "Where registrations start" section on `/rollup` shows one block per course site:
  English courses (trainings.arc42.org, as in ADR-0023) and German courses (arc42.de).
- The German block asks arc42.de's own dashboard (`site_id: arc42.de`) with
  `has_done` on `/anmeldung/`, over **the last 12 months** (`date_range: 12mo`), long
  enough for a full course season and recent enough to reflect today's site.
- Two cuts: `visit:entry_page` and `visit:source`. There is no entry-hostname cut, since
  every visit in that dashboard enters on arc42.de. The source cut carries a note that
  arrivals from other arc42 sites appear there, because arc42.de is not in the rollup.
- Each block states where its figures come from: the rollup since the join date, or the
  site's own dashboard over its window.
- ADR-0023's rules apply to both blocks unchanged: failed is never shown as empty, no
  zero stands in for unknown, the small-sample sentence below 20 visits, and no section
  at all without `PLAUSIBLE_API_KEY`.
- The two blocks are queried concurrently inside the collection goroutine, so the German
  block does not lengthen the collection pass.

## Consequences

- Two more Plausible API calls per collection run, five for the section in all.
- The two blocks do not compare like with like: the English figures follow one visit
  across sites, the German ones restart at arc42.de's door, and their windows differ.
  Each block says which it is; comparing their numbers directly would mislead.
- If arc42.de ever joins the rollup, its block can switch to the rollup query, with an
  entry-hostname cut and a join date, in the target definition in
  `internal/statsv2/registrationOrigins.go`.
