# 21. report the rollup beside the summed totals

Date: 2026-09-14

## Status

Accepted. The public page `/rollup/` and the table's "Unique across sites" row are
superseded by ADR-0022: the rollup is shown to maintainers only.

## Context

The traffic table adds up each site's own Plausible dashboard. A person who reads
arc42.org and then docs.arc42.org is counted twice in that total, so it overstates how
many people use the arc42 sites.

meta.arc42.org ADR-0005 introduced `rollup.arc42.com`: a receive-only Plausible site
that every member reports into besides its own
(`data-domain="<own-host>,rollup.arc42.com"`). Plausible identifies a visitor by a
daily-salted hash that includes the site ID, so all members share one visitor identity
there and a person on two member sites is one visitor. Until now the rollup was visible
only on plausible.io.

Two facts make the rollup awkward to show next to the table:

- Its roster differs from the table's rows. status.arc42.org has a row and does not
  report; pdfminion.arc42.org reports its own dashboard only and has no row; arc42.de has
  a row and does not report either.
- Members join on different days, and the rollup has no data before 2026-07-30. A window
  reaching back before a join compares a member's full own count with a partial rollup.

## Decision

- The collector asks Plausible for `rollup.arc42.com`'s visitors and page views with the
  same three aggregate queries every site gets (7 days, 30 days, 12 months).
- The table gains a **second footer row, "Unique across sites"**, with the rollup's own
  figures. It is not a total and is not styled as one; its label links to `/rollup/`.
  It is a row rather than a tile because the tiles are repository-centric and the rollup
  has no repository.
- Membership is declared per property in `types.Arc42properties`: `RollupSince` (the day
  copies started to arrive) or `RollupPending` (snippet changed, not deployed).
- `types.BuildRollup` compares the rollup with the sum of its members' dashboards and
  derives "counted on more than one site". That figure is `n/a` whenever the two sides did
  not count the same traffic: a member joined inside the window, a value was not
  measured, or the rollup counts more than its members (which signals a stale roster).
- `/rollup/` is a static Jekyll page explaining how to read the rollup, with the numbers
  from the service's `/rollup` fragment and the embedded shared rollup dashboard.

## Consequences

- Three extra Plausible API calls per collection run.
- Deploying a member means a one-line roster edit here: drop `RollupPending`, set
  `RollupSince` to the deploy date. Forgetting it turns the comparison `n/a` rather than
  wrong, and `TestDeclaredRollupRoster` guards the date format.
- The 12-month comparison stays `n/a` until a year after the latest join.
- A journeys page (entry hosts, entry and exit pairs, content health) needs Plausible's
  Stats API v2, which the current client library does not speak. It is a separate step.
