---
title: "arc42 Status and Statistics"
layout: splash
permalink: /
header:
  overlay_image: /images/statistics-splash.webp
  overlay_filter: rgba(0, 65, 83, 0.7)

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

<p class="status-verdict">
  <span class="status-token status-token--unmonitored" role="img" aria-label="not monitored"></span>
  <span><b>Availability is not measured yet.</b> This page reports usage and repository health only.
  The dash in the <i>Status</i> column means no uptime data has been collected for that site &mdash;
  see <a href="https://github.com/arc42/status.arc42.org-site/blob/main/documentation/adrs/0019-availability-monitoring-with-github-actions-prober.md">ADR&nbsp;19</a>
  for how that is going to work.</span>
</p>

<div id="statsTable"
     hx-get="{{ site.stats_api }}/statsTable"
     hx-trigger="load"
     hx-swap="outerHTML">

  <!-- Loading state: the real column count and row count, so the page does not
       jump when the data lands. -->
  <table class="stats-skeleton" aria-hidden="true">
    <tbody>
    {% comment %}
      13 rows x 8 columns: two header rows, ten sites, one totals row, and the
      site / status / 3x(visitors, pageviews) columns the service returns --
      the same shape, so the page barely moves when the real table lands.
    {% endcomment %}
    {% for row in (1..13) %}
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
      10 properties, the first two spanning two columns like the hub tiles, so
      the grid the skeleton draws is the grid that arrives. Each placeholder
      carries a band block of its own, because the real tile does.
    {% endcomment %}
    {% for tile in (1..10) %}
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

<script>
(function () {
    var API = '{{ site.stats_api }}';

    if (!window.htmx) { return; }

    // htmx 1.x has no hx-timeout attribute; without this a hanging service
    // would leave the loading state on screen forever.
    htmx.config.timeout = 15000;

    function icon() {
        return '<svg class="stats-panel__icon" viewBox="0 0 24 24" fill="none" ' +
               'stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">' +
               '<path d="M12 3 1.8 20.5h20.4L12 3Z"/><path d="M12 9.5v5"/><path d="M12 18h.01"/></svg>';
    }

    // name the host the page is actually talking to -- fly.io in production,
    // localhost when served by `make site` against a local backend
    var HOST = API.replace(/^https?:\/\//, '').replace(/\/.*$/, '');

    var SITES_FINE = 'The arc42 sites themselves are unaffected — only this is missing.';
    var SLEEPS = 'The statistics service sleeps when nobody is looking at this page, so the first request of the day can need a moment.';

    // Both fragments -- the statistics table and the tiles -- get the same
    // three states from one implementation, so they can never drift apart.
    function wire(regionId, slotId, path, noun) {
        var region = document.getElementById(regionId);
        if (!region) { return; }

        var endpoint = API + path;

        function showPanel(title, text, detail) {
            var slot = document.getElementById(slotId);
            if (!slot) { return; }
            slot.innerHTML =
                '<div class="stats-panel" role="alert">' + icon() +
                '<div class="stats-panel__body">' +
                '<p class="stats-panel__title">' + title + '</p>' +
                '<p class="stats-panel__text">' + text + '</p>' +
                '<p class="stats-panel__detail">' + detail + '</p>' +
                '<button type="button" class="stats-panel__retry">Try again</button>' +
                '</div></div>';
            region.setAttribute('aria-busy', 'false');
        }

        region.addEventListener('htmx:timeout', function () {
            showPanel('The statistics service did not answer',
                      HOST + ' took longer than 15 seconds to send the ' + noun + '. ' + SITES_FINE,
                      SLEEPS);
        });

        region.addEventListener('htmx:sendError', function () {
            showPanel('The statistics service is unreachable',
                      'Your browser could not reach ' + HOST + ' at all. ' + SITES_FINE,
                      'This is either a network problem on your side, or the service is down.');
        });

        region.addEventListener('htmx:responseError', function (event) {
            var status = (event.detail && event.detail.xhr) ? event.detail.xhr.status : 0;
            showPanel('The statistics service reported a problem',
                      HOST + ' answered with HTTP ' + status + '. ' + SITES_FINE,
                      'Nothing you can do from here; the service needs a look.');
        });

        region.addEventListener('htmx:afterSwap', function () {
            region.setAttribute('aria-busy', 'false');
        });

        region.addEventListener('click', function (event) {
            var button = event.target.closest('.stats-panel__retry');
            if (!button) { return; }
            button.disabled = true;
            button.textContent = 'Trying…';
            region.setAttribute('aria-busy', 'true');
            htmx.ajax('GET', endpoint, { target: '#' + slotId, swap: 'outerHTML' });
        });
    }

    wire('stats-region', 'statsTable', '/statsTable', 'table');
    wire('tiles-region', 'tileGrid', '/tiles', 'tiles');
})();
</script>

<!--
  The six embedded Plausible dashboards used to live here, six 1600px iframes
  deep. They now sit one per property at /site/<host>/, together with the
  availability detail ADR-0019 will produce. This index keeps every subpage
  reachable without the tiles -- the tile fragment covers eight properties,
  this list covers all ten.
-->
<h2 id="per-site-detail">Per-site detail</h2>

<p class="site-index__note">Traffic dashboards and availability, one page per property.</p>

<ul class="site-index">
{% for s in site.data.arc42_sites %}
  <li class="site-index__item" data-site="{{ s.host }}">
    <a href="{{ '/site/' | append: s.host | append: '/' | relative_url }}">{{ s.host }}</a>
    <span class="site-index__meta">{% if s.plausible_embed %}traffic dashboard{% else %}no dashboard{% endif %}</span>
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
                language: {"decimal": "-", "thousands": "." },
               // column 1 is Status: identical in every row until ADR-19 lands
               columnDefs: [{ orderable: false, targets: [1] }] 
            });
        }
    });
</script>
