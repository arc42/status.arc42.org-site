# 14. keep atlas hcl and go code in sync manually

Date: 2024-01-07

## Status

Accepted

## Context

Database schema with table and column names are maintained with Atlas (see ADR-0013).

That involves a central `schema.hcl` file defining table and column names.

In Go code, we use those names (as constants) for database operations.

How to keep hcl and go code in sync?

## Decision

We decided to manually keep both in sync, and refrain from automating this process.

The file `internal/database/schema.hcl` defines the db schema.

## Consequences

What becomes easier or more difficult to do and any risks introduced by the change that will need to be mitigated.
