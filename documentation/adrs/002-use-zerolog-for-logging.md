# 2. use zerolog for logging

Date: 2023-11-27

## Status

Accepted

## Context

Previously several `fmt.Print*` function calls were scattered around the code.
Especially when run in the fly.io cloud, these normal print statements didn't work properly.

Therefore, [issue #58](https://github.com/arc42/status.arc42.org-site/issues/58) was created to track this problem.

## Decision

We will use [zerolog](https://github.com/rs/zerolog) for logging.


## Consequences

* a global logger ('log') is made available 
* zerolog has to be imported by all packages
