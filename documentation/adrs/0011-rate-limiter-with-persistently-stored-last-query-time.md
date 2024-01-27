# 11. rate-limiter with persistently stored last-query-time

Date: 2023-12-29
modified: 2024-01-25

## Status

Accepted.

## Context

* The number of visitors and pageviews does not change quickly 
* Requests to our API might come frequently

* It would be resource-friendly if we kept results of our external APIs (Plausible, GitHub) for appropriate times:
  * Plausible statistics for at least 10 min, maybe even 30 or 60 min is good enough
  * GitHub: Issues, bugs and PR counts might change frequently, therefore, cache retention needs to be lower here

* This is only valid if our service wasn't shutdown by our cloud provider between calls to our API.
If our current server is a "fresh instance", we have to ignore the last-time-called values.
* 
* As we run in the cloud, we need to store the last-query-time for our external APIs in a persistent store (DB).


## Decision

1. Create a table that keeps the invocation times of **our** API
2. Create tables that keep the last-query time for both Plausible and GitHub requests.
2. Call the Plausible.io API only once every `plausible_Rate_Limit_Minutes` (defaulting to 20)
3. Call the GitHub API only once every `gitHub_Rate_Limit_Minutes` (defaulting to 3)

The tables shall have the following format:
 
#### TimeOfInvocation

TimeOfInvocation stores the DateTime of invocations, plus request IP and route.

| invocation_time | request_ip | route |
| --- | --- | --- |
| A DATETIME entry, denoting at what date/time our API was called  | caller IP | Route that was called (e.g. statsTable or ping |


#### TimeOfPlausibleCall

TimeOfPlausibleCall stores the DateTime of the calls to plausible.io API.

| plausible_invocation_time |
| --- | 
| A DATETIME entry, denoting at what date/time the plausible API was called |

#### TimeOfGitHubCall

TimeOfGitHubCall stores the DateTime of the calls to GitHub API.

| github_invocation_time | 
| --- | 
| A DATETIME entry, denoting at what date/time the GitHub API was called | 

## Consequences

* we use turso.tech as our (cloud) database
* create a script/app to create that table

```
CREATE TABLE TimeOfStatusRequest IF NOT EXISTS (
TimeCalled DATETIME  PRIMARY KEY,
ServiceVersion STRING,
RequestIP STRING
);
```
