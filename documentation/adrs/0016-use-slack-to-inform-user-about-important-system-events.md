# 16. use Slack to inform user about important system events

Date: 2024-02-02 (Updated 2026-08-06)

## Status

Accepted (Updated 2026-08-06)

## Context

Requirement F-004 requires the owners of the system to be informed about important system events.

## Decision

Routine "system startup" and "acquisition of usage and repository data" notifications are disabled.
Slack notifications are sent whenever an availability check fails (meaning a monitored domain or subdomain is not available / down).

## Consequences

- Slack app to be created and configured
- [Slack API](https://pkg.go.dev/github.com/slack-go/slack@v0.12.3#section-readme) used in `internal/slack`
- `SLACK_AUTH_TOKEN` secret passed to the availability prober workflow in GitHub Actions and set at fly.io