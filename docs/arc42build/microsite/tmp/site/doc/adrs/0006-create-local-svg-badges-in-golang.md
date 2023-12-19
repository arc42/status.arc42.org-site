status=published
jbake-order=6
type=page
jbake-menu=adrs
jbake-title=6. create-local-svg-badges-in-golang
~~~~~~


# 6. create-local-svg-badges-in-golang

Date: 2023-12-08

## Status

Accepted

## Context

The _badges_ showing the number of open issues and bugs for each repository have to be loaded.

Sending requests to the external service (_shields.io_) adds a runtime dependency, and potential runtime and energy overhead.

## Decision

* Pre-load at least 20 such badges for open-issues and bugs to local storage
* Make these badges available to static Jekyll site under `images/badges/`
* Add function to create appropriate path/filename combination, so these can be added to HTML output

## Consequences

* New `golang` app in /cmd