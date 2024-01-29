# 15. caching of results

Date: 2024-01-28

## Status

Accepted

## Context

Using the external APIs from Plausible and GitHub is resource intensive, and their results don't change too often.

## Decision

Introduce caching, related to [ADR 0011 (rate limit)](./0011-rate-limiter-with-persistently-stored-last-query-time.md)

For Golang, a few caching libraries/packages exist, most targeting large volumes of data and/or high-throughput applications.

We tested the simple packages
* [go-cache](https://github.com/patrickmn/go-cache) and
* [zcache](https://github.com/arc242/zcache)

as both hav both global and entry-specific expiration times and is simple to use.

`zcache` is an updated fork of `go-cache`, and go-cache is no longer actively maintained. Therefore we use `zcache`.

A small example can be found in /cmd/cache/try-caching.go.

## Consequences

* cache needs to be typed
* expiration needs to be set when pushing data into the cache
