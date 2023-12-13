# Documentation Overview

![context diagram](documentation/3-context-status-arc42-org.drawio.png)

### Explanation

| Element  | Description |
| --- | --- |
| Plausible.io | This (commercial) service collects usage data. We access via their API |
| GitHub | Bug- and Issue-counter, plus additional repository status. Access via GraphQL |
| Fly.io | Our cloud provider (aka hyperscaler). We query the geographical region from their API.|
| Uptimerobot | in planning: Tracks availability of the site(s). API not yet tested.|
| Turso DB | in planning: A distributed, self-replicating SQLite database in the cloud. |

API calls are handled via the site-usage-statistics service, located in its own [repository](https://github.com/arc42/site-usage-statistics).

## Development Status + Planning

Upcoming features and current development are planned with a [GitHub project](https://github.com/orgs/arc42/projects/5/views/1)

