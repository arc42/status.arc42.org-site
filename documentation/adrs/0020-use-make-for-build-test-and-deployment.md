# 20. Use Make as central automation tool for build, test, database, and deployment tasks

Date: 2026-08-05

## Status

Accepted

## Context

The repository consists of multiple interconnected components:
* A Go backend service (`go-app/`)
* A static Jekyll shell website (`docs/`)
* Database schema management via Atlas (`go-app/internal/database/`) targeting both local SQLite and remote TursoDB (libSQL)
* Cloud deployment on Fly.io (`flyctl`)

Without a central workflow runner, local development, database migrations, and deployments require remembering multiple complex multi-line shell commands, directory changes, and secret-sourcing incantations.

## Decision

We use **GNU Make** (`Makefile` in the repository root) as the single, standardized entry point for all development, build, test, database management, and deployment tasks.

Key capabilities provided via `Makefile`:
* **Environment Diagnostics**: `make doctor` verifies Docker, Go, `flyctl`, Atlas CLI, and API secret configuration.
* **Development Servers**: `make backend` and `make site` to run the two local dev processes.
* **Backend Build & Quality**: `make build`, `make test`, `make lint`, `make probe`.
* **Fly.io Deployment**: `make fly-deploy`, `make fly-status`, `make fly-logs`, `make fly-ssh`, `make fly-secrets`.
* **Database & Schema (Atlas)**: `make db-apply-dev`, `make db-apply-prod`, `make db-diff-dev`, `make db-diff-prod`, `make db-validate`, `make db-shell-dev`.

## Consequences

* Developers and AI agents have a single, self-documenting interface (`make help`).
* Sourcing live environment secrets (`go-app/set-api-keys.sh`) is automatically handled by relevant targets (`make backend`, `make probe`, `make db-apply-prod`).
* `make doctor` catches missing tools (`go`, `docker`, `flyctl`, `atlas`) early before failures occur.
