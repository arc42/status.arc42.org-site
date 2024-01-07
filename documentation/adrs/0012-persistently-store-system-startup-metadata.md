# 12. persistently store system startup metadata

Date: 2024-01-03

## Status

Accepted

## Context

We're running this service in the cloud (mostly), and our cloud provider [fly.io](https://fly.io) might shutdown instances when they need to.

To get a feel for fly.ios behaviour in that sense, we need to know when the service is started or re-started.

## Decision

We write the current time to a database table during system startup, prior to starting the API server.

Therefore we introduce a new table:


#### TimeOfSystemStartup

TimeOfSystemStartup stores the DateTime when the whole system was started.

| startup | app_version | environment |
| --- | --- | ---|
| A DATETIME entry, denoting at what date/time our system was started  | The version of our service, kept in the global variable `AppVersion` | PROD or DEV or 

## Consequences

- enhance the scripts for creating, dropping and dumping tables, accordingly.
- create function to be called *once* immeadiately after system startup