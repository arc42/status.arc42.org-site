# 7. migrate-to-single-repo

Date: 2023-12-09

## Status

Accepted

## Context

Previously, the code for `status.arc42.org` was split into TWO different GitHub repositories:

* https://github.com/arc42/status.arc42.org-site
* https://github.com/arc42/site-usage-statistics

>That resulted from history: Between January and October 2023, only the static (Jekyll-based) site was available. 
>Only then the (dynamic, Golang-based) app was developed.

This two-repo situation suffers from some drawbacks:

It is unclear:
* where to put documentation?
* where to open/maintain bugs, issues and/or feature-requests?

Updating or releasing often needs coordinated changes in both repositories.

## Decision

Migrate both repositories, move content into the status-repo.

## Consequences

* build processes have to be updated

See [GitHub issue #72](https://github.com/arc42/status.arc42.org-site/issues/72)