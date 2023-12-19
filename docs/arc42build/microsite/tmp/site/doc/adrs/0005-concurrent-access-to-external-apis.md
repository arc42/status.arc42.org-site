status=published
jbake-order=5
type=page
jbake-menu=adrs
jbake-title=5. concurrent access to external APIs
~~~~~~


# 5. concurrent access to external APIs

Date: 2023-12-04

## Status

Accepted

## Context

For the 7+ arc42 websites we need 3 queries each to plausible.io (for the three time periods) plus one graphql query to GitHub.

Performing these 28 queries sequentially takes approx. 4 seconds on average, which seems too slow for the website.

Therefor, we evaluated concurrent access to these APIs.

## Decision

Goroutines are an established way of performing concurrent processing in Golang. Their approach is well-documented and they are fairly easy to implement.

But we don't want the additional complexity of channels, but stick to the easiest programming model possible:

* refactor the funcs calling external APIs to get a distinct pointer to a struct
* have a `sync.WaitGroup` in the func surrounding these calls
* do not use a mutex.


## Consequences


