---
title: "arc42 Status and Statistics"
layout: splash
permalink: /
header:
  overlay_image: /images/statistics-splash.webp
  # slate #3a4550 at .75 = 5.32:1 for white hero text, matching the 5.37:1
  # the retired petrol overlay gave. See ADR-0007 / BRAND.md deny-list.
  overlay_filter: rgba(58, 69, 80, 0.75)

  actions:
    - label: "&#8594; arc42.org"
      url: "https://www.arc42.org"
    - label: "&#8594; arc42-Docu"
      url: "https://docs.arc42.org"
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
      10 rows x 8 columns: two header rows, the seven sites the table carries
      (types.Arc42properties, InTable), one totals row, and the
      site / status / 3x(visitors, pageviews) columns the service returns --
      the same shape, so the page barely moves when the real table lands.
    {% endcomment %}
    {% for row in (1..10) %}
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
-->
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
</script>

<!--
  The six embedded Plausible dashboards used to live here, six 1600px iframes
  deep. They now sit one per property at /site/<key>/, together with the full
  issue and pull-request lists and the availability detail ADR-0019 will
  produce. This index is the plain-HTML way in, so every subpage stays
  reachable even when the tile fragment never arrives.
-->
<h2 id="per-site-detail">Per-site detail</h2>

<p class="site-index__note">Open issues and pull requests, traffic and availability &mdash; one page per property.</p>

<ul class="site-index">
{% for s in site.data.arc42_sites %}
  <li class="site-index__item" data-site="{{ s.key }}">
    {%- comment -%}
      A planned property has no page to link to. Linking it anyway would be a
      404 dressed as a destination.
    {%- endcomment -%}
    {% if s.planned %}<span class="site-index__planned">{{ s.key }}</span>
    {% else %}<a href="{{ '/site/' | append: s.key | append: '/' | relative_url }}">{{ s.key }}</a>{% endif %}
    <span class="site-index__meta">{% if s.planned %}not built yet{% elsif s.plausible_embed %}traffic dashboard{% else %}no dashboard{% endif %}</span>
  </li>
{% endfor %}
</ul>

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
