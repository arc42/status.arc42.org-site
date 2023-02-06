"use strict";

// example of plausible url:
// "https://plausible.io/api/v1/stats/aggregate?metrics=pageviews,visitors&period=30d&site_id=docs.arc42.org"


const PLTOKEN = 'CdmiPgsOg4V4roUa1wsDZG0nRzcEowMjot2zkBNV8PhcEI93gZtNLrfjzBQHyXgc';


function constructURL(period, site) {
    const plausibleBaseURL = "https://plausible.io/api/v1/stats/aggregate?";
    const metrics = "metrics=pageviews,visitors";
    const periodBase = "&period=";
    return plausibleBaseURL + metrics + periodBase + period + "&site_id=" + site;
}

function iterateSites() {
    const sites = ["arc42.org", "arc42.de", "docs.arc42.org", "faq.arc42.org", "quality.arc42.org"];
    for (let i = 0; i < sites.length; i++) {
        console.log(sites[i]);
        let URL30d = constructURL("30d", sites[i]);
        let URL12m = constructURL("12mo", sites[i]);
        callPlausible(URL30d);
        callPlausible(URL12m);
    }
}

iterateSites();


async function callPlausible(plausibleURL) {

    const response = await fetch(plausibleURL, {
        headers: {
            "Content-Type": "application/json",
            "Access-Control-Allow-Headers": "access-control-allow-origin, Authorization, Access-Control-Allow-Credentials,X-Requested-With, Access-Control-Request-Method, Access-Control-Request-Headers",
            "Access-Control-Request-Headers": "access-control-allow-origin, Authorization, Access-Control-Allow-Credentials, X-Requested-With, Access-Control-Request-Method, Access-Control-Request-Headers",
            "Access-Control-Allow-Credentials": "true",
            "Access-Control-Allow-Origin": "https://plausible.io,localhost",
            "Authorization": "Bearer " + PLTOKEN
        }
    });

    const body = await response.json();
    console.log(body);

    const pageview = body.results.pageviews.value;

    console.log(pageview)


};


