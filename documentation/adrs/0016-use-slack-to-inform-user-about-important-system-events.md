# 16. use Slack to inform user about important system events

Date: 2024-02-02

## Status

Accepted

## Context

Requirement F-004 requires the owners of the system to be informed about important system events.

## Decision

Such important events are "system startup" and "acquisition of usage and repository data".
These shall be sent to a Slack channel to notify the owners.

## Consequences

- Slack app to be created
- [Slack API](https://pkg.go.dev/github.com/slack-go/slack@v0.12.3#section-readme) to be used
- Slack OAuth Token to be set at fly.io