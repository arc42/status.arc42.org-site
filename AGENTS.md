# Agent Guidelines for arc42 Status Site

## Build Commands
- **Build**: `cd go-app && go build -o arc42-status` (or `make build`)
- **Run**: `cd go-app && go run main.go` (or `make backend`)
- **Test**: `cd go-app && go test ./...` (or `make test`)
- **Lint**: `cd go-app && golangci-lint run` (or `make lint`)
- **Docker Build**: `cd go-app && docker build -t arc42-status .`
- **Fly Deploy**: `make fly-deploy` (or `flyctl deploy --remote-only`)
- **Fly Status & Logs**: `make fly-status`, `make fly-logs`, `make fly-ssh`
- **Database (Atlas)**: `make db-apply-dev`, `make db-apply-prod`, `make db-diff-dev`, `make db-diff-prod`, `make db-validate`

## Code Style Guidelines
- **Language**: Go 1.21+, module name: `arc42-status`
- **Imports**: Group stdlib, external, internal packages separately
- **Logging**: Use `github.com/rs/zerolog` with structured logging
- **Error Handling**: Return errors explicitly, use `log.Error()` for logging
- **Naming**: Use camelCase for variables, PascalCase for exported types
- **Constants**: Use ALL_CAPS for package-level constants
- **Comments**: Document exported functions and types, include version history in main.go

## Project Structure
- Main app in `go-app/` directory
- Internal packages in `go-app/internal/`
- Templates embedded using `//go:embed *.gohtml`
- Environment variables for configuration (LOGLEVEL, API keys)
- Database: TursoDB (libSQL)
- Deployment: Fly.io with Docker