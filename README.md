# Repository Overview
Like all cool websites, arc42 has a status-subdomain, showing usage- and repo statistics for its domains and subdomains.

This is a _multi-repo_, and the directories are organized as follows:

| **Static Jekyll Site** | **Documentation**  | **Statistic service (Go app)** |
|------|--------|----------------|
| [**/docs**](/docs) | [**/documentation**](/documentation) |[**/go-app**](/go-app)  |
|  Jekyll/Github-pages based static website.|arc42-based technical  documentation, created with docToolchain. | Cloud-based statistics service written in Golang |
| [![build+deploy jekyll](https://github.com/arc42/status.arc42.org-site/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/arc42/status.arc42.org-site/actions/workflows/pages/pages-build-deployment) |    | [![build+deploy go-app](https://github.com/arc42/status.arc42.org-site/actions/workflows/fly.yml/badge.svg?branch=main)](https://github.com/arc42/status.arc42.org-site/actions/workflows/fly.yml) <br>[![golang-lint](https://github.com/arc42/status.arc42.org-site/actions/workflows/golang-lint.yml/badge.svg)](https://github.com/arc42/status.arc42.org-site/actions/workflows/golang-lint.yml) |

![Repo overview with logos](documentation/images/0-repo-overview.drawio.png)

## Development and Build Targets

We use `make` (`Makefile`, see [ADR-0020](documentation/adrs/0020-use-make-for-build-test-and-deployment.md)) as the single entry point for development, database schema management, testing, and deployment:

### Local Development

This site runs as two processes during local development:

```bash
make backend    # terminal 1: run the Go statistics service on :8043
make site       # terminal 2: run the Jekyll dev server on :4000
make doctor     # verify setup health (Docker, Go, flyctl, Atlas, secrets, ports)
make help       # show all available make targets
```

`make site` loads `docs/_config.dev.yml`, pointing the static site to the local backend. The deployed site uses `_config.yml` (pointing to Fly.io).

The backend requires `go-app/set-api-keys.sh`, created from template:

```bash
cp go-app/set-api-keys.sh.template go-app/set-api-keys.sh
```

### Database & Schema Management (Atlas)

Database schemas for both local SQLite (`~/arc42-stats-dev.db`) and production TursoDB (`libsql+ws://...`) are declaratively managed with Atlas ([ADR-0013](documentation/adrs/0013-use-atlas-for-declarative-database-schema-management.md)):

```bash
make db-apply-dev   # Apply schema.hcl to local development SQLite DB
make db-apply-prod  # Apply schema.hcl to production TursoDB
make db-diff-dev   # Dry-run / diff schema against local dev DB
make db-diff-prod  # Dry-run / diff schema against production TursoDB
make db-validate   # Validate schema syntax against an in-memory DB
make db-shell-dev  # Open interactive sqlite3 shell on local dev DB
```

Availability is measured by a scheduled GitHub Actions workflow (`.github/workflows/probe.yml`, every ~15 minutes, [ADR-0019](documentation/adrs/0019-availability-monitoring-with-github-actions-prober.md)) that runs `go-app/cmd/probe` and writes into TursoDB. Locally: `make db-apply-dev` once, then `make probe`.

### Fly.io Deployment & Diagnostics

The backend Go app is hosted on Fly.io ([ADR-0008](documentation/adrs/0008-deploy-on-fly-io.md)). Manage deployments directly via `make`:

```bash
make fly-deploy   # Deploy backend Go service to Fly.io
make fly-status   # Show status of Fly.io app and machines
make fly-logs     # Tail live production logs from Fly.io
make fly-ssh      # Open SSH console session on running Fly.io instance
make fly-secrets  # List secrets configured on Fly.io
```

### Build, Test & Lint

```bash
make build    # Compile the Go service to go-app/arc42-status
make test     # Run Go unit tests
make lint     # Run golangci-lint over the Go code
```

## Technologies used

![Plausible Analytics Badge](https://img.shields.io/badge/Plausible%20Analytics-5850EC?logo=plausibleanalytics&logoColor=fff&style=plastic)
![Turso Badge](https://img.shields.io/badge/Turso-4FF8D2?logo=turso&logoColor=000&style=plastic)
![Jekyll Badge](https://img.shields.io/badge/Jekyll-C00?logo=jekyll&logoColor=fff&style=plastic)
![GitHub Badge](https://img.shields.io/badge/GitHub-181717?logo=github&logoColor=fff&style=plastic)
![GitHub Actions Badge](https://img.shields.io/badge/GitHub%20Actions-2088FF?logo=githubactions&logoColor=fff&style=plastic)
![GoLand Badge](https://img.shields.io/badge/GoLand-000?logo=goland&logoColor=fff&style=plastic)

![Go Badge](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=fff&style=plastic)
![Markdown Badge](https://img.shields.io/badge/Markdown-000?logo=markdown&logoColor=fff&style=plastic)
![GraphQL Badge](https://img.shields.io/badge/GraphQL-E10098?logo=graphql&logoColor=fff&style=plastic)
![JSON Badge](https://img.shields.io/badge/JSON-000?logo=json&logoColor=fff&style=plastic)
![Asciidoctor Badge](https://img.shields.io/badge/Asciidoctor-E40046?logo=asciidoctor&logoColor=fff&style=plastic)

## Development and Feature Planning


Upcoming features and are planned with a [GitHub project](https://github.com/orgs/arc42/projects/5/views/1)

## Supported by INNOQ

This work is actively supported by [INNOQ Deutschland GmbH](https://innoq.com).

![Supported by INNOQ](documentation/images/supported-by-innoq.svg)

## Licence

![CC-BY-SA](documentation/images/by-sa.png)

This content is provided _as-is_, without any guarantees. 
It is open-source under the [Creative-Commons-ShareAlike 4.0](https://creativecommons.org/licenses/by-sa/4.0/deed.en) licence:

* see the [original text](https://creativecommons.org/licenses/by-sa/4.0/) and the
* [legal details](https://creativecommons.org/licenses/by-sa/4.0/legalcode.en)


Created 2023 by [Dr. Gernot Starke](https://gernotstarke.de) and contributors. 

