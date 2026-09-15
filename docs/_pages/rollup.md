---
title: "All arc42 sites, counted once"
layout: single
classes: wide
author_profile: false
permalink: /rollup/
hide_title: true
ribbon: "status and statistic details"
---

{%- comment -%}
  The rollup page (ADR-0021), reached from the "Unique across sites" row under
  the traffic table's totals.

  Built like a per-site page but not from site-detail.html: the rollup is not a
  property. It has no repository, no host to probe and no record in
  _data/arc42_sites.yml, and giving it one would put it into every loop over
  the family. The numbers come from the service's /rollup fragment; the
  explanation is static, so it is readable before (and without) the service.

  data-site="rollup" matches no registered hue on purpose: the rollup is every
  member at once and wears the neutral band.
{%- endcomment -%}

<div class="site-page" data-site="rollup">

  <div class="site-page__band">
    <h1 class="site-page__name">All arc42 sites, counted once</h1>
  </div>

  <div class="site-page__body">
    <p class="site-page__tagline">The rollup is one Plausible dashboard that every member site reports
    into besides its own. A person who reads arc42.org and then the docs is two visitors in
    the table on the home page, and one visitor here.</p>
    <p class="site-page__links">
      <a class="site-page__link" href="https://plausible.io/share/rollup.arc42.com?auth=_Uc9YHaNOglp6i0arDxiE">Open the rollup dashboard &#8599;</a>
      <a class="site-page__link" href="{{ '/' | relative_url }}">&#8592; Back to the dashboard</a>
    </p>
  </div>

</div>

<h2 id="numbers">The numbers</h2>

<div id="rollup-region" class="stats-region" aria-live="polite" aria-busy="true">

  <div id="rollupFigures"
       hx-get="{{ site.stats_api }}/rollup"
       hx-trigger="load"
       hx-swap="outerHTML">
    <div class="detail-skeleton" aria-hidden="true">
      <span class="skeleton-bar" style="width:90%;height:1.1rem"></span>
      <span class="skeleton-bar" style="width:90%"></span>
      <span class="skeleton-bar" style="width:90%"></span>
      <span class="skeleton-bar" style="width:60%"></span>
    </div>
    <p class="stats-loading-note">Comparing the rollup with its members&hellip;</p>
  </div>

</div>

## How the rollup works

- **Every member reports twice.** Its snippet names two Plausible sites, its own first:
  `data-domain="docs.arc42.org,rollup.arc42.com"`. Each page view goes to both.
  The member's own dashboard, and every figure on the home page, stay exactly as they were.
- **One person, one visitor, across sites.** Plausible sets no cookies. It recognises a
  visitor by a hash of the IP address, the browser's user agent, the site the page view is
  sent to, and a salt that changes every day. Every member sends to the same site,
  `rollup.arc42.com`, so the same person gets the same hash on every member site that day.
- **One visit, across sites.** A visit ends after 30 minutes without a page view. Reading
  arc42.org/overview and then three docs pages within that time is one visit with four page
  views, in the rollup.
- **The old script, on purpose.** Plausible's newer per-site snippet cannot report into two
  dashboards, so every member uses the legacy script (meta.arc42.org ADR-0005).
- **Cost.** Every page view on a member site is billed twice.

## Reading the rollup

| Figure | Rollup compared with the members added up | What it tells you |
|---|---|---|
| Page views | Equal | Every page view is sent to both. A gap means a snippet is missing somewhere, or a member joined inside the window. |
| Unique visitors | Lower | The realistic count of people. Members added up minus the rollup = people counted on more than one site. |
| Visits | Lower | Moving from arc42.org into the docs continues the visit instead of starting a second one. |
| Bounce rate | Lower | Clicking from `/overview` into the docs is no longer a bounce. |
| Visit duration, pages per visit | Higher | Time and pages on every member site add up within one visit. |
| Top sources | Different | Where the visit *started* (a search engine, LinkedIn, a bookmark). Moving to another member site inside a visit is not a source here, while a member's own dashboard lists arc42.org as a referrer. |
| Entry and exit pages | Different | First and last page of the whole visit, across sites. An exit on arc42.org/overview now means the reader really left. |
| Outbound link clicks | Include moves between members | A click from arc42.org to the docs is movement inside the family, not a departure. Filter those URLs out to see real exits. |

## Traps

1. **Page lists show paths without host names.** `/` from every member is one row, and
   arc42.de mirrors arc42.org's paths should it ever join. Filter by *Hostname* before
   reading Top Pages, Entry Pages or Exit Pages. A useful check: *Hostname is docs.arc42.org*
   should roughly match the docs dashboard's own figures.
2. **Long windows count returning people again.** The daily salt means someone who comes
   back on three days is three visitors in a 30-day window. That is true of every Plausible
   dashboard, the members' included, so compare windows with each other, not with a head count.
3. **Joins look like growth.** A member that starts reporting adds its traffic from that day.
   The table above says "not comparable yet" for every window that reaches back before the
   latest join, and lists each member's join date.
4. **The rollup began on 2026-07-30.** Before that no page view reached it, so the 12-month
   figures cover less than a year until 2027-07-30.
5. **Some readers are never counted.** Ad blockers and Firefox's tracking protection block
   Plausible. This affects the rollup and the members' dashboards alike.

## Three questions and where to look

- **How many people use arc42's sites?** The rollup's *Unique visitors*, unfiltered.
- **Where do people who reach the trainings start?** Rollup dashboard, filter
  *Hostname is trainings.arc42.org*, then *Entry Pages*.
- **How many use more than one site?** "Counted on more than one site" in the table above.

Plausible does not record page sequences, so no report here can show "the ten most common
paths through arc42". What it can show honestly is where visits start and end, which sites
they touch, and how pages are read.

## The rollup dashboard

<p class="site-page__dashlink">
  Live from Plausible, privacy-friendly and cookie-free.
  <a href="https://plausible.io/share/rollup.arc42.com?auth=_Uc9YHaNOglp6i0arDxiE">Open this dashboard on plausible.io &#8599;</a>
</p>
<iframe class="site-plausible" plausible-embed
        title="Plausible analytics dashboard for the arc42 rollup"
        src="https://plausible.io/share/rollup.arc42.com?auth=_Uc9YHaNOglp6i0arDxiE&embed=true&theme=light"
        scrolling="no" frameborder="0" loading="lazy"
        style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
<script async src="https://plausible.io/js/embed.host.js"></script>

<script>window.ARC42_STATS_API = '{{ site.stats_api }}';</script>
<script src="{{ '/assets/js/status-regions.js' | relative_url }}"></script>
<script>
  window.arc42Status.wire('rollup-region', 'rollupFigures', '/rollup', 'rollup figures');
</script>
