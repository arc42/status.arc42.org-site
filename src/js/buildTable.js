"use strict";

let body = document.getElementsByTagName("h2")[0];
let table = document.createElement("table");


const sites = ["arc42.org", "arc42.de", "docs.arc42.org", "faq.arc42.org", "quality.arc42.org"];
const headers = ["Site", "Visitors 30d",  "Pageviews 30d", "Visitors 12mo", "Pageviews 12mo", "Uptime"];
const periods = ["30d", "12mo"];
const metrics = ["Visitors", "Pageviews"];
const uptimeBadges = [];


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

    generateStatisticsColumns(row, siteName);
}

function generateStatisticsColumns(row, siteName) {
    generatePeriodStats( row, siteName, "30d");
    generatePeriodStats( row, siteName, "12mo");
}


async function generatePeriodStats(row, siteName, period) {


    let token = constructToken(42).replaceAll('!','');

    let urlString = constructURL(period, siteName);
    const response = await fetch(urlString, {
        headers: {
            "Content-Type": "application/json",
            "accept": "application/json",
            "origin": "https://status.arc42.org",
            "Authorization": "Bearer " + constructToken(42).replaceAll("!","")
        }
    });
    const body = await response.json();

    let cellVisitors = row.insertCell();
    cellVisitors.appendChild(document.createTextNode(body.results.visitors.value));
    let cellViews = row.insertCell();
    cellViews.appendChild(document.createTextNode(body.results.pageviews.value));

}

function constructURL(period, site) {
    const plausibleBaseURL = "https://plausible.io/api/v1/stats/aggregate?";
    const metrics = "metrics=pageviews,visitors";
    const periodBase = "&period=";
    return plausibleBaseURL + metrics + periodBase + period + "&site_id=" + site;
}


function constructToken(param) {
 let t1 = '!!!!' + 'CdmiPgsOg4V4ro';
 let t2 = '!!!' + 'Ua1wsDZG0nRzcE';
 let t3 = '!!!!!!' + 'owMjot2zkBNV8Ph';
 let t4 = '!!!!' + 'cEI93gZtNLrfjzBQHyXgc';

 let t5 = param.toString() + '!';

 return t1 + "!!" + t2+t3+t4;
}

generateTableHead(table, headers);
generateTableBody(table, sites);

body.appendChild(table);


