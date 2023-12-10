# 8. Deploy on fly.io

Date: 2023-12-10

## Status

Accepted

## Context

We want to make the statistic-service available online, so we need to either host a service on-premise or in the cloud.

## Decision

Deploy the (PRODUCTION) service on [fly.io](https://fly.io), an affordable cloud service provider with a nice developer experience.

## Consequences

* some secrets (API-tokens) need to be configured via the fly.io command line tool.
* for development, the flyctl utility needs to be installed. See their [documentation](https://fly.io/docs/speedrun/) for details.
