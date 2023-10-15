---
title: "arc42 Status and Statistics"
layout: splash
permalink: /
header:
  overlay_image: /images/statistics-splash.webp
  overlay_color: "#d72f2f"
  overlay_filter: rgba(10, 80, 250, 0.4)
  
  actions: 
    - label: "&#8594; arc42.org"
      url: "https://www.arc42.org"
    - label: "&#8594; arc42.de"
      url: "https://www.arc42.de"

   
summary:
- title: "Summary of all arc42 sites"
  excerpt: '
  <iframe plausible-embed src="https://plausible.io/share/rollup.arc42.com?auth=H_2ArEfjjP25OdRumQluH&embed=true&theme=light" scrolling="no" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%;"></iframe>
  '

de-org-canvas:
- title: "arc42.de"
  excerpt: '
  <iframe plausible-embed src="https://plausible.io/share/arc42.de?auth=IYzUmMI8s2PYKgggJhO7q&embed=true&theme=light" height="600" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%;" ></iframe>
  '
- title: "arc42.org"
  excerpt: '
<iframe plausible-embed src="https://plausible.io/share/arc42.org?auth=tNNpNN0VqPh9xbjkaEPrx&embed=true&theme=light" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%; height: 600px;"></iframe>
'
- title: "canvas.arc42.org"
  excerpt: '
<iframe plausible-embed src="https://plausible.io/share/canvas.arc42.org?auth=sAJkIzBTeFg-a5ndJenA4&embed=true&theme=light" scrolling="no" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
'

  

doc-faq-quality:
- title: "docs.arc42.org"
  excerpt: '
  <iframe plausible-embed src="https://plausible.io/share/docs.arc42.org?auth=D_6pSvlKkq_hTlttpTOtz&embed=true&theme=light" heigth="600" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%;"></iframe>
  '
- title: "faq.arc42.org"
  excerpt: '<iframe plausible-embed src="https://plausible.io/share/faq.arc42.org?auth=wc065ryr-3YNoYFluaqGh&embed=true&theme=light" scrolling="no" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
  '
- title: "quality.arc42.org"
  excerpt: '<iframe plausible-embed src="https://plausible.io/share/quality.arc42.org?auth=cjoKlapPdw3czFugGy6jM&embed=true&theme=light" scrolling="no" frameborder="0" loading="lazy" style="width: 1px; min-width: 100%; height: 1600px;"></iframe>
'

---

<script async src="https://plausible.io/js/embed.host.js"></script>
<script src="https://unpkg.com/htmx.org@1.9.6"
            integrity="sha384-FhXw7b6AlE/jyjlZH5iHa/tTe9EpJ1Y55RjcgPbjeWMskSxZt1v9qkxLJWNJaGni"
            crossorigin="anonymous"></script>
 

## Details

<div id="version"
     hx-get="https://arc42-stats.fly.dev/statsTable"
     hx-trigger="load delay"
     hx-swap="outerHTML">
    <table>
        <tr>
            <th>Site</th>
            <th>Visitors-7d</th>
            <th>PageViews-7d</th>
            <th>Visitors-30d</th>
            <th>PageViews-30d</th>
            <th>Visitors-12m</th>
            <th>PageViews-12m</th>
        </tr>

        <tr>
            <td>arc42.org</td>
            <td>2149</td>
            <td>2614</td>
            <td>7967</td>
            <td>708</td>
            <td>5741</td>
            <td>520</td>
        </tr>
    </table>
</div>
<p></p>
<button class='btn' hx-get="https://arc42-stats.fly.dev/statsTable"
        hx-target="#version"
        hx-swap="outerHTML">
    Reload
</button>

## Breakdown for Sites
<div style="font-size: 14px; padding-bottom: 14px;">All stats powered by <a target="_blank" style="color: #4F46E5; text-decoration: underline;" href="https://plausible.io">Plausible Analytics</a></div>



Detailed statistics for:

* all [arc42 sites combined](#combined)
* [arc42.org](#de-org-canvas)
* [arc42.de](#de-org-canvas)
* [docs.arc42.org](#doc-faq-quality)
* [faq.arc42.org](#doc-faq-quality)
* [quality.arc42.org](#doc-faq-quality)
* [canvas.arc42.org](#de-org-canvas)



<a id="combined">
{% include feature_row id="summary" %}


<a id="de-org-canvas">
## German and International Site, canvas

{% include feature_row id="de-org-canvas" %}


<a id="doc-faq-quality"/>
## Subdomain Sites
{% include feature_row id="doc-faq-quality" %}


<!--
the script that is currently not working :-(
<script src="https://status.arc42.org/assets/js/buildTable.min.js"></script>
-->


The page was generated on {{ site.time }}.
