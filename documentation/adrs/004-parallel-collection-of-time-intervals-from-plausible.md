# 4. parallel collection of time intervals from plausible

Date: 2023-12-01

## Status

Accepted

## Context

The visitor and pageview counts from plausible.io are collected for three distinct time intervals (7D, 30D, 12M).

Each of these require a single API call to plausible.io. 
These can be handled sequentially or in parallel Goroutines.

## Decision

We call plausible.io in parallel Goroutines for the three time intervals.

## Consequences

We need to implement a func to call plausible with the site and time-interval as parameter. The return value needs to include the time-intervall, therefore we define a (kind-of enum) type for these (ADR-0005).
