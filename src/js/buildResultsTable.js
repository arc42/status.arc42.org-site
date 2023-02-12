"use strict";

// operating mode: if debug (modes[0]) then mock-API is called,
// and output sent to console.log

const modes = ["debug", "live"];
const mode = modes[0];


function debug() {
    return (mode == modes[0]);
}


const sites = ["arc42.org", "arc42.de", "docs.arc42.org", "faq.arc42.org", "quality.arc42.org"];
const periods = ["30d", "12mo"];
const metrics = ["Visitors", "Pageviews"];
const keyVisitors = metrics[0];
const keyViews = metrics[1];

let results = {};

collectResultsForAllSites();
displayResultsOnConsole();

createTableFromResults();


function createKey(site, period, metric) {
    return site + period + metric
}

function collectResultsForAllSites() {
    for (const site of sites) {
        createResultsForSite(site);
    }
}

// challenge: v/p values are delivered asynchronously by Promises
function createResultsForSite(site) {

    for (const period of periods) {
        createSiteResultsForPeriod(site, period);
    }
}

function createSiteResultsForPeriod(site, period) {

    let periodStats = {};

    // in debug mode, return simple integers
    if (debug()) {
        periodStats = generateDummyStats(site, period);
    }
    // if in normal (non-debug) mode, return results from plausible API
    else {
        periodStats = generatePeriodStats(site, period);
    }

    results[createKey(site, period, metrics[0])] = periodStats.keyViews;
    results[createKey(site, period, metrics[1])] = periodStats.keyVisitors;

//    console.log("getStatisticsForSiteAndPeriod ", site, ":", period, ":", periodStats);

}


function generateDummyStats(site, period) {

    let vsts = metrics[0].length * period.length + site.length;
    let pagvs = metrics[1].length * period.length + site.length;

    return {
        keyVisitors: vsts,
        keyViews: pagvs
    };

}

async function generatePeriodStats(siteName, period) {

    let token = constructToken(42).replaceAll('!', '');

    let urlString = constructURL(period, siteName);
    const response = await fetch(urlString, {
        headers: {
            "Content-Type": "application/json",
            "accept": "application/json",
            "origin": "https://status.arc42.org",
            "Authorization": "Bearer " + token
        }
    });
    const body = await response.json();

    const cellVisitors = body.results.visitors.value;
    const cellViews = body.results.pageviews.value;

    return {keyVisitors: cellVisitors, keyViews: cellViews};

    function constructToken(param) {
        let t1 = '!!!!' + 'CdmiPgsOg4V4ro';
        let t2 = '!!!' + 'Ua1wsDZG0nRzcE';
        let t3 = '!!!!!!' + 'owMjot2zkBNV8Ph';
        let t4 = '!!!!' + 'cEI93gZtNLrfjzBQHyXgc';

        let t5 = param.toString() + '!';

        return t1 + "!!" + t2 + t3 + t4;
    }
}

function constructURL(period, site) {
    const plausibleBaseURL = "https://plausible.io/api/v1/stats/aggregate?";
    const metrics = "metrics=pageviews,visitors";
    const periodBase = "&period=";
    return plausibleBaseURL + metrics + periodBase + period + "&site_id=" + site;
}

function getValueFromResults(siteName, period, metric){
    let key = createKey(siteName, period, metric);
    return results[key];

}
function displayResultsOnConsole() {
    for (const siteName of sites) {

        for (const period of periods) {
            for (const metric of metrics) {

                console.log(siteName, period, metric + ":" + getValueFromResults(siteName, period, metric) + ",");
            }
        }
    }
}

function createTableFromResults() {
    const headers = ["Site", "Visitors 30d", "Pageviews 30d", "Visitors 12mo", "Pageviews 12mo", "Uptime"];

    let body = document.getElementsByTagName("h2")[0];
    let table = document.createElement("table");

    generateTableHead(table, headers);
    generateTableBody(table, sites);

    if (body != null) {
        body.appendChild(table);
    } else {
        console.log("no h2-tag found. Results:\n", results);
    }

    function generateTableHead(table, headers) {
        let thead = table.createTHead();
        let row = thead.insertRow();

        for (const element of headers) {
            let th = document.createElement("th");
            let text = document.createTextNode(element);
            th.appendChild(text);
            row.appendChild(th);
        }
    }

    function generateTableBody(table) {
        for (let siteName of sites) {
            let row = table.insertRow();
            generateSingleRow(row, siteName);
        }
    }

    function generateSingleRow(row, siteName) {
        let cell1 = row.insertCell();
        cell1.appendChild(document.createTextNode(siteName));

        for (const period of periods) {
            for (const metric of metrics) {
                let cell = row.insertCell();

                let value = getValueFromResults(siteName, period, metric);
                cell.appendChild(document.createTextNode(value));
            }
        }
        let cellUptime = row.insertCell();
        cellUptime.appendChild(document.createTextNode("t.b.d."));

    }

}
