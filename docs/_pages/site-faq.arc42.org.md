---
title: "faq.arc42.org"
layout: single
classes: wide
author_profile: false
permalink: /site/faq.arc42.org/
host: faq.arc42.org
---

{%- comment -%}
  Per-site subpage. Generated shape, data-driven content: everything below is
  looked up from _data/arc42_sites.yml by this page's `host` front-matter key,
  so the ten stubs never disagree about a URL, a repo or a Plausible token.

  Why ten files and not one loop: a Jekyll page is one output file, and turning
  one source into ten permalinks needs either a generator plugin (not on the
  GitHub Pages whitelist in _config.yml) or a shared _includes fragment. Both
  are outside this change's file ownership, so the duplication is confined to
  this markup and none of it is data.
{%- endcomment -%}
{%- assign s = site.data.arc42_sites | where: "host", page.host | first -%}

<div class="site-page" data-site="{{ s.host }}">

  <div class="site-page__band">
    <span class="site-page__name">{{ s.name }}</span>
    <span class="site-page__host">{{ s.host }}</span>
  </div>

  <div class="site-page__body">
    <p class="site-page__tagline">{{ s.tagline }}</p>
    <p class="site-page__links">
      <a class="site-page__link" href="{{ s.url }}">Visit the site &#8599;</a>
      <a class="site-page__link" href="{{ s.repo }}">GitHub repository &#8599;</a>
      <a class="site-page__link" href="{{ '/' | relative_url }}">&#8592; Back to the dashboard</a>
    </p>
  </div>

</div>

<h2 id="availability">Availability</h2>

<!-- ADR-0019 placeholder. Deliberately empty of numbers: nothing has been
     measured for this property yet, and a zero or a green tick here would be
     an invention. The prober fills this region; until it does, the region
     says what it does not know. -->
<div class="site-availability" id="availability-detail" data-site="{{ s.host }}" data-adr="0019">
  <p class="status-verdict">
    <span class="status-token status-token--unmonitored" role="img" aria-label="not monitored"></span>
    <span><b>Not monitored yet.</b> No uptime has been collected for {{ s.host }}.
    <a href="https://github.com/arc42/status.arc42.org-site/blob/main/documentation/adrs/0019-availability-monitoring-with-github-actions-prober.md">ADR&nbsp;19</a>
    describes the GitHub-Actions prober that will fill this region &mdash; current state, a 30-day
    daily strip, and the precise 7d / 30d / 12m figures.</span>
  </p>
</div>

<h2 id="traffic">Traffic</h2>

{% if s.plausible_embed %}
{%- assign share = s.plausible_embed | split: "&embed=" | first -%}
<p class="site-page__dashlink">
  Live from Plausible, privacy-friendly and cookie-free.
  <a href="{{ share }}">Open this dashboard on plausible.io &#8599;</a>
</p>
<iframe class="site-plausible" plausible-embed
        title="Plausible analytics dashboard for {{ s.host }}"
        src="{{ s.plausible_embed }}"
        scrolling="no" frameborder="0" loading="lazy"
        style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
<script async src="https://plausible.io/js/embed.host.js"></script>
{% else %}
<!-- No share link exists for this property. An empty iframe would look like a
     broken page; saying so costs one sentence and is true. -->
<p class="site-page__nodata">
  <span class="status-token status-token--unmonitored" role="img" aria-label="no data"></span>
  <span><b>No analytics dashboard for this property.</b> No Plausible share link has been
  issued for {{ s.host }}, so there is nothing to embed here &mdash; not an outage, and not a
  zero. If traffic for this property should be public, a share link has to be created in
  Plausible first; this page will pick it up from
  <code>_data/arc42_sites.yml</code>.</span>
</p>
{% endif %}
