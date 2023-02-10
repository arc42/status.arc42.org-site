"use strict";

const sites = ["arc42.org", "arc42.de", "docs.arc42.org", "faq.arc42.org", "quality.arc42.org"];
const periods = ["30d", "12mo"];
const metrics = ["Visitors", "Pageviews"];

let results = {};

createResultsTable();

function createResultsTable() {
    console.log("createResultsTable");
    for (const site of sites) {
        let row = createResultsRow( site );
        results[site] = row ;
        console.log(site + "done:" + results);
    }
    console.log("finished\n");
    console.log( results);
}

// row consists of site, v30d, p30d, v12mo, p12mo, uptime
// {"site": {
// challenge: v/p values are delivered asynchronously by Promises
function createResultsRow(site) {
    console.log("creating row for " + site);
    let thisRow = new Map();
    //thisRow.set("site", site);

    console.log( thisRow );

    for (const period of periods) {
        createResultsForPeriod(thisRow, site, period);
    }

    return thisRow;
}

function createResultsForPeriod(row, site, period) {
    console.log("creating results for " + site + "/" + period);
    let key1 = metrics[0] + period;
    let key2 = metrics[1] + period;
    row.set( key1, (key1+site).length);
    row.set( key2, (key2+site).length);
}


