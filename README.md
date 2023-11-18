# Status and Statistics Overview
like all cool websites we need a status-subdomain with some statistics.

## Overview

![context diagram](docs/3-context-status-arc42-org.drawio.png)

### Explanation

| Element  | Description |
| --- | --- |
| Plausible.io | This (commercial) service collects usage data. We access via their API |
| GitHub | Bug- and Issue-counter, plus additional repository status. Access via GraphQL |
| Fly.io | Our cloud provider (aka hyperscaler). We query the geographical region from their API.|
| Uptimerobot | in planning: Tracks availability of the site(s). API not yet tested.|
| Turso DB | in planning: A distributed, self-replicating SQLite database in the cloud. |

API calls are handled via the site-usage-statistics service, located in its own [repository](https://github.com/arc42/site-usage-statistics).