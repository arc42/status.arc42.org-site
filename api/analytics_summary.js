"use strict";

// example of plausible url:
// "https://plausible.io/api/v1/stats/aggregate?metrics=pageviews,visitors&period=30d&site_id=docs.arc42.org"
const sites = ["arc42.org", "arc42.de", "docs.arc42.org", "faq.arc42.org", "quality.arc42.org"];
const PLTOKEN = 'CdmiPgsOg4V4roUa1wsDZG0nRzcEowMjot2zkBNV8PhcEI93gZtNLrfjzBQHyXgc';


function constructURL(period, site) {
    const plausibleBaseURL = "https://plausible.io/api/v1/stats/aggregate?";
    const metrics = "metrics=pageviews,visitors";
    const periodBase = "&period=";
    return plausibleBaseURL + metrics + periodBase + period + "&site_id=" + site;
}

function iterateSites() {
    for (let site of sites) {
        console.log(site);
        let URL30d = constructURL("30d", site);
        let URL12m = constructURL("12mo", site);
        callPlausible(URL30d, site);
        callPlausible(URL12m, site);
    }
}



async function callPlausible(plausibleURL, site, period) {

    const response = await fetch(plausibleURL, {
        headers: {
//            "method": "GET",
            "Content-Type": "application/json",
            "accept": "application/json",
            "origin": "https://status.arc42.org",
            "Authorization": "Bearer " + PLTOKEN
        }
    });

    const body = await response.json();
    //console.log(body);

    const pageview = body.results.pageviews.value;
    const visitors = body.results.visitors.value;

    console.log(site + ":" + period + ": pageviews: " + pageview + "\tvisitors: " + visitors);


};


iterateSites();
