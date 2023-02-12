"ust strict";

const sites = ["a.de", "a.org", "d.a.org"];
const periods = ["30d", "12mo"]

let results = {};

function createKey( site, period){
    const key = site+period;
    return key
}

function createMapEntry(site, period, value){
    results[createKey(site, period)] = value;
}

function createMap(){
    for (const site of sites){
        for (const period of periods){
            const value = site.length * period.length
            createMapEntry(site, period, value)
        }
    }
}
function main(){
    createMap();
    console.log(results);
    deStructureResults();
}

function deStructureResults(){
    for (const site of sites) {
        for (const period of periods) {
            const key = createKey(site, period)
            console.log( "value of ", key, " = ", results[key]);
        }
    }
}

main();