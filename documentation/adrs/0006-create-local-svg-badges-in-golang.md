# 6. create-local-svg-badges-in-golang

Date: 2023-12-08

## Status

Deprecated.

Update: 23-12-25: The badges shall be replaced by pure numbers to improve visual consistency of the table.

## Context

The _badges_ showing the number of open issues and bugs for each repository have to be loaded.

Sending requests to the external service (_shields.io_) adds a runtime dependency, and potential runtime and energy overhead.

## Decision

* Pre-load at least 20 such badges for open-issues and bugs to local storage
* Make these badges available to static Jekyll site under `images/badges/`
* Add function to create appropriate path/filename combination, so these can be added to HTML output

## Consequences

* New `golang` app in /cmd