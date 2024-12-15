# 18. use env-vars for API secrets and tokens

Date: 2024-04-18

## Status

Accepted

## Context

(External) APIs require secrets (similar to passwords) for access. 

## Decision

Two options:
* environment variables, set during deployment. For local development, these are kept in a non-versioned shell script that needs to be source'd once per session.
* A server based system like keycloak.

Decision: Use environment variables, as it's simpler and creates less external dependencies.

GitHub detects if the file is accidentally commited into the central repo.

## Consequences

* make sure the file `set-api-keys.sh` is run prior to local deployment
* keep the names (id's) of the secrets in sync between deployment platforms and source code.
