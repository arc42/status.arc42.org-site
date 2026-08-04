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



de-org-canvas:
  - title: "arc42.de"
    excerpt: '
  <iframe plausible-embed src="https://plausible.io/share/arc42.de?auth=IYzUmMI8s2PYKgggJhO7q&embed=true&theme=light" height="1600" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
  '
  - title: "arc42.org"
    excerpt: '
<iframe plausible-embed src="https://plausible.io/share/arc42.org?auth=tNNpNN0VqPh9xbjkaEPrx&embed=true&theme=light" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
'
  - title: "canvas.arc42.org"
    excerpt: '
<iframe plausible-embed src="https://plausible.io/share/canvas.arc42.org?auth=sAJkIzBTeFg-a5ndJenA4&embed=true&theme=light" scrolling="no" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
'



doc-faq-quality:
  - title: "docs.arc42.org"
    excerpt: '
  <iframe plausible-embed src="https://plausible.io/share/docs.arc42.org?auth=D_6pSvlKkq_hTlttpTOtz&embed=true&theme=light" heigth="1600" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%;height: 1600px;"></iframe>
  '
  - title: "faq.arc42.org"
    excerpt: '<iframe plausible-embed src="https://plausible.io/share/faq.arc42.org?auth=wc065ryr-3YNoYFluaqGh&embed=true&theme=light" scrolling="no" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
  '
  - title: "quality.arc42.org"
    excerpt: '<iframe plausible-embed src="https://plausible.io/share/quality.arc42.org?auth=cjoKlapPdw3czFugGy6jM&embed=true&theme=light" scrolling="no" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
'

---


<!--
  Layer 1: the tiles. Per-site, heterogeneous, repo-centric — they answer
  "what needs me". The table below answers "how is the family doing" with
  numbers you read down a column. Split by data type, so the two never
  answer the same question.
-->
<div id="tiles-region" aria-live="polite" aria-busy="true">

<div id="tileGrid"
     hx-get="{{ site.stats_api }}/tiles"
     hx-trigger="load"
     hx-swap="outerHTML">
  <div class="tile-grid" aria-hidden="true">
    {% comment %}
      8 properties, the first two spanning two columns like the hub tiles, so
      the grid the skeleton draws is the grid that arrives.
    {% endcomment %}
    {% for tile in (1..8) %}
    <div class="tile-skeleton{% if tile <= 2 %} tile-skeleton--hub{% endif %}">
      <span class="skeleton-bar" style="width:55%"></span>
      <span class="skeleton-bar" style="width:30%;height:1.4em"></span>
      <span class="skeleton-bar" style="width:85%"></span>
      <span class="skeleton-bar" style="width:70%"></span>
    </div>
    {% endfor %}
  </div>
  <p class="stats-loading-note">Asking GitHub what nobody has looked at yet&hellip;</p>
</div>

</div>

<!--
  Layer 2: the table. Commensurable numbers, read down a column.
  Neither region is swapped itself: each stays put so screen readers keep a
  stable live region, and so an error panel has somewhere to live when the
  service cannot be reached. Only the inner #tileGrid / #statsTable is replaced.
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
      11 rows x 11 columns: two header rows, eight sites, one totals row --
      the same shape the service returns, so the page barely moves when it lands.
    {% endcomment %}
    {% for row in (1..11) %}
      <tr>
        {% for col in (1..11) %}
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

    // Both fragments -- the tiles and the statistics table -- get the same
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

    wire('tiles-region', 'tileGrid', '/tiles', 'tiles');
    wire('stats-region', 'statsTable', '/statsTable', 'table');
})();
</script>

## Breakdown for Sites

* [arc42.org](#de-org-canvas)
* [arc42.de](#de-org-canvas)
* [docs.arc42.org](#doc-faq-quality)
* [faq.arc42.org](#doc-faq-quality)
* [quality.arc42.org](#doc-faq-quality)
* [canvas.arc42.org](#de-org-canvas)

<a id="de-org-canvas"/>
## German and International Site, canvas

{% include feature_row id="de-org-canvas" %}


<a id="doc-faq-quality"/>
## Subdomain Sites
{% include feature_row id="doc-faq-quality" %}



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
