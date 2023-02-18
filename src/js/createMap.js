"ust strict";

const sites = ["a.de", "a.org", "d.a.org"];
const periods = ["30d", "12mo"]

let results = {};

function createKey( site, period){
    const key = site+period;
    return key
}

async function createValue( site, period){
    const rndInt = Math.floor(Math.random() * 1000) + 1
    console.log( "sleeping for ", rndInt);
    await sleep( rndInt);
    const val = await site.length * period.length;
    return val;
}
 function createMapEntry(site, period, value){

    results[createKey(site, period)] = value;
}

async function createMap(){
    for (const site of sites){
        for (const period of periods){
            const value = await createValue(site, period);
            console.log( "created value for ",site, ":", period, "=", value);
            await createMapEntry(site, period, value)
        }
    }
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}


async function main(){
    await createMap();
    console.log(results);
    await deStructureResults();
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