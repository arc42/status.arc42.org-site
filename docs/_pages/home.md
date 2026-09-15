---
title: "Status and Statistics"
layout: splash
permalink: /
# The page title moved into the masthead band (the `ribbon:` below,
# rendered by _includes/masthead.html) so the hero itself can carry the
# live status line instead of repeating the title. hide_title suppresses
# the h1 that _includes/page__hero.html would otherwise print in the hero.
ribbon: "Status and Statistics"
hide_title: true
header:
  overlay_image: /images/statistics-splash.webp
  # slate #3a4550 at .75 = 5.32:1 for white hero text, matching the 5.37:1
  # the retired petrol overlay gave. See ADR-0007 / BRAND.md deny-list.
  overlay_filter: rgba(58, 69, 80, 0.75)
  # Gives the live status line (moved here via JS once it arrives from the
  # stats API) a place to land in the hero.
  status_slot: true
---

<!--
  Layer 1: the table. Commensurable numbers, read down a column -- it answers
  "how is the family doing". The tiles below answer "what needs me": per-site,
  heterogeneous, repo-centric. The split is by data type, so the two can never
  answer the same question.

  Table above tiles is an owner decision (2026-08-04) and reverses the earlier
  "tiles first" recommendation in the surface brief -- the brief's own §5 now
  records the reversal.

  Neither region is swapped itself: each stays put so screen readers keep a
  stable live region, and so an error panel has somewhere to live when the
  service cannot be reached. Only the inner #statsTable / #tileGrid is replaced.
-->
<div id="stats-region" class="stats-region" aria-live="polite" aria-busy="true">

<div id="statsTable"
     hx-get="{{ site.stats_api }}/statsTable"
     hx-trigger="load"
     hx-swap="outerHTML">

  <!-- Loading state: the real column count and row count, so the page does not
       jump when the data lands. -->
  <table class="stats-skeleton" aria-hidden="true">
    <tbody>
    {% comment %}
      11 rows x 8 columns: two header rows, the seven sites the table carries
      (types.Arc42properties, InTable), the totals row and the rollup row
      (ADR-0021), and the site / status / 3x(visitors, pageviews) columns the
      service returns -- the same shape, so the page barely moves when the
      real table lands.
    {% endcomment %}
    {% for row in (1..11) %}
      <tr>
        {% for col in (1..8) %}
        <td class="{% if col == 1 %}skeleton-cell--site{% elsif col == 2 %}skeleton-cell--status{% else %}skeleton-cell--number{% endif %}"><span
           class="skeleton-bar{% if row <= 2 %} skeleton-bar--head{% endif %}"></span></td>
        {% endfor %}
      </tr>
    {% endfor %}
    </tbody>
  </table>
  <p class="stats-loading-note">Sifting through the cloud (currently: fly.io) for current statistics&hellip;</p>

</div>

</div>

<!--
  Layer 2: the tiles. Identity in the band, meaning in the body -- each tile
  wears its property's registered band colour (BRAND.md) and carries all state
  neutrally below it. The hue never encodes status: on this page amber and red
  already mean "degraded" and "down".

  The paragraph below is the on-page explanation of that rule (critique
  2026-08-07: the hub/satellite/hue system was legible in code comments and
  BRAND.md, not to a first-time visitor). It sits in the static Jekyll shell,
  not the htmx fragment, so it is there before the tiles even finish loading.
-->
<p class="tiles-legend">Below, each card represents one arc42 site. 
Every site wears its own registered colour band (which does NOT represent site status).</p>

<div id="tiles-region" aria-live="polite" aria-busy="true">

<div id="tileGrid"
     hx-get="{{ site.stats_api }}/tiles"
     hx-trigger="load"
     hx-swap="outerHTML">
  <div class="tile-grid" aria-hidden="true">
    {% comment %}
      12 properties, the first two spanning two columns like the hub tiles, so
      the grid the skeleton draws is the grid that arrives. Each placeholder
      carries a band block of its own, because the real tile does.
    {% endcomment %}
    {% for tile in (1..12) %}
    <div class="tile-skeleton{% if tile <= 2 %} tile-skeleton--hub{% endif %}">
      <span class="tile-skeleton__band"></span>
      <span class="tile-skeleton__body">
        <span class="skeleton-bar" style="width:30%;height:1.4em"></span>
        <span class="skeleton-bar" style="width:85%"></span>
        <span class="skeleton-bar" style="width:70%"></span>
        <span class="skeleton-bar" style="width:55%"></span>
      </span>
    </div>
    {% endfor %}
  </div>
  <p class="stats-loading-note">Asking GitHub what nobody has looked at yet&hellip;</p>
</div>

</div>

<script>window.ARC42_STATS_API = '{{ site.stats_api }}';</script>
<script src="{{ '/assets/js/status-regions.js' | relative_url }}"></script>
<script>
  // Both fragments -- the statistics table and the tiles -- get their loading,
  // timeout, unreachable and HTTP-error behaviour from the same implementation
  // the per-site pages use, so the three can never drift apart.
  window.arc42Status.wire('stats-region', 'statsTable', '/statsTable', 'table');
  window.arc42Status.wire('tiles-region', 'tileGrid', '/tiles', 'tiles');

  // The hero states "how is arc42 doing" instead of repeating the page
  // title (which moved to the masthead badge). The verdict line itself
  // still comes from the service inside #statsTable's fragment -- this
  // just relocates that one paragraph into the hero once it lands, and
  // leaves the rest of the table where it was.
  (function () {
    var statsRegion = document.getElementById('stats-region');
    var heroStatusSlot = document.getElementById('hero-status-slot');
    if (!statsRegion || !heroStatusSlot) { return; }

    statsRegion.addEventListener('htmx:afterSwap', function () {
      var verdict = statsRegion.querySelector('.status-verdict');
      if (verdict) {
        heroStatusSlot.innerHTML = '';
        heroStatusSlot.appendChild(verdict);
      }
    });

    ['htmx:timeout', 'htmx:sendError', 'htmx:responseError'].forEach(function (name) {
      statsRegion.addEventListener(name, function () {
        heroStatusSlot.innerHTML = '<p class="page__hero-status-loading">Status unavailable right now &mdash; see below.</p>';
      });
    });
  }());
</script>

<!--
  The six embedded Plausible dashboards used to live here, six 1600px iframes
  deep. They now sit one per property at /site/<key>/, together with the full
  issue and pull-request lists and the availability detail ADR-0019 will
  produce.

  The plain-HTML link list that used to live here (critique 2026-08-07) was
  cut, not relocated: every property already gets a tile above, and every
  non-planned tile already links to this same /site/<key>/ URL from its band
  and its "...more on X" line. The list added a third appearance of the same
  name with no new information -- restore it, filtered to properties the tile
  grid doesn't cover, the day a property exists that has no tile of its own.
-->

The page was generated on {{ site.time }}.

<!-- enable table sorting -->

<script>
document.body.addEventListener('htmx:load', function(event) {
        if (event.target.id === 'sortableStatsTable') {
            $('#sortableStatsTable').DataTable({
                info:false,
                searching: false,
                paging: false,
                ordering: true,
                language: {"decimal": "-", "thousands": "." }
            });
        }
    });
</script>
