---
title: "arc42 Statistics"
layout: splash
permalink: /statsTable
header:
  overlay_image: /images/statistics-splash.webp
  overlay_color: "#d12f2f"
  overlay_filter: rgba(10, 120, 240, 0.2)
  
  actions: 
    - label: "&#8594; arc42.org"
      url: "https://www.arc42.org"
    - label: "&#8594; arc42.de"
      url: "https://www.arc42.de"

 

---


## Welcome to the compressed version of arc42 sites statistics

<div hx-get="http://localhost:8043/statsTable"
     hx-trigger="load delay"
     hx-swap="outerHTML"
     hx-target="#statsTable">
</div>

<!-->
<div hx-get="https://arc42-stats.fly.dev/statsTable"
<-->

<!-- the following div will be swapped with the HTML generated by the backend API -->
<div id="statsTable">
  <table border="1">
    <tr>
        <th rowspan="2"><img src="./images/minion-logo-100px.png"></th>
        <th colspan="2" style="border-left: 2px solid black;">7 Days</th>
        <th colspan="2" style="border-left: 2px solid black;">30 Days</th>
        <th colspan="2" style="border-left: 2px solid black;">12 Month</th>
        <th rowspan="2" style="border-left: 2px solid black;">Issues</th>
    </tr>
    <tr>
        <th style="border-left: 2px solid black;">Visitors</th>
        <th>PageViews</th>
        <th style="border-left: 2px solid black;">Visitors</th>
        <th>PageViews</th>
        <th style="border-left: 2px solid black;">Visitors</th>
        <th>PageViews</th>
    </tr>
    <tr>
        <td colspan="8"> <img src="./images/spinner.gif"> collecting data...</td>
      
   </tr>
  </table>
</div>
