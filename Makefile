# Development tasks for status.arc42.org
#
# This site has two halves that run as two processes, so local development
# needs two terminals:
#
#     terminal 1:  make backend     the Go statistics service on :8043
#     terminal 2:  make site        the Jekyll dev server on :4000
#
# `make site` loads _config.dev.yml, which points the page at the local
# backend instead of fly.io -- so what you see is what you changed.
#
# `source` is bash, not POSIX sh, and every recipe line runs in its own
# shell, so sourcing the API keys and starting the service must stay on
# ONE line. Hence SHELL below.
SHELL := /bin/bash

.DEFAULT_GOAL := help

.PHONY: help backend site doctor stop clean build test lint \
        build-site build-image install update shell logs check-secrets

SITE_DIR  := docs
APP_DIR   := go-app
SECRETS   := $(APP_DIR)/set-api-keys.sh
TEMPLATE  := $(APP_DIR)/set-api-keys.sh.template
SITE_PORT := 4000
API_PORT  := 8043
COMPOSE   := docker compose -f $(SITE_DIR)/docker-compose.yml

help: ## Show this help
	@printf "\nstatus.arc42.org — development\n\n"
	@printf "  Two processes, two terminals:\n"
	@printf "    terminal 1:  \033[36mmake backend\033[0m   Go service   → http://localhost:$(API_PORT)\n"
	@printf "    terminal 2:  \033[36mmake site\033[0m      Jekyll site  → http://localhost:$(SITE_PORT)\n\n"
	@printf "  Unsure whether your setup is sound? \033[36mmake doctor\033[0m\n\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
	@printf "\n"

# ---------------------------------------------------------------- development

backend: check-secrets ## Run the Go statistics service on :8043 (terminal 1)
	@printf "==> statistics service on http://localhost:$(API_PORT) — Ctrl-C to stop\n"
	@printf "==> try http://localhost:$(API_PORT)/ping\n"
	cd $(APP_DIR) && source ./set-api-keys.sh && go run main.go

site: ## Start the Jekyll dev server on :4000, wired to the local backend (terminal 2)
	@printf "==> Open http://localhost:$(SITE_PORT)  (NOT http://0.0.0.0:$(SITE_PORT) — Firefox refuses to connect to 0.0.0.0)\n"
	@printf "==> The stats table is fetched from http://localhost:$(API_PORT); run 'make backend' in another terminal.\n"
	@holder=$$(docker ps --filter "publish=$(SITE_PORT)" --format '{{.Names}}'); \
	if [ -n "$$holder" ]; then \
		printf "==> Port $(SITE_PORT) is already in use by another container: %s\n" "$$holder"; \
		printf "==> That is probably a dev server from a sibling arc42 site repo. Stop it first:\n"; \
		printf "==>   docker stop %s\n" "$$holder"; \
		exit 1; \
	fi
	$(COMPOSE) up --build

stop: ## Stop and remove the running Jekyll container
	$(COMPOSE) down

# ---------------------------------------------------------------------- checks

doctor: ## Check the local dev setup and report what is missing
	@printf "\n[doctor] status.arc42.org development setup\n\n"
	@if docker info >/dev/null 2>&1; then \
		printf "  [ok]   Docker daemon reachable\n"; \
	else \
		printf "  [fail] Docker daemon not reachable — start Docker Desktop\n"; \
	fi
	@if command -v go >/dev/null 2>&1; then \
		printf "  [ok]   Go installed (%s)\n" "$$(go version | awk '{print $$3}')"; \
	else \
		printf "  [fail] Go not installed — https://go.dev/dl/\n"; \
	fi
	@if [ -f $(SECRETS) ]; then \
		printf "  [ok]   %s present\n" "$(SECRETS)"; \
	else \
		printf "  [fail] %s missing — cp %s %s and fill it in\n" "$(SECRETS)" "$(TEMPLATE)" "$(SECRETS)"; \
	fi
	@if [ -f $(SITE_DIR)/_config.dev.yml ]; then \
		printf "  [ok]   %s/_config.dev.yml present (site talks to localhost:$(API_PORT))\n" "$(SITE_DIR)"; \
	else \
		printf "  [fail] %s/_config.dev.yml missing — the local site would talk to fly.io\n" "$(SITE_DIR)"; \
	fi
	@holder=$$(docker ps --filter "publish=$(SITE_PORT)" --format '{{.Names}}'); \
	if [ -n "$$holder" ]; then \
		printf "  [warn] port $(SITE_PORT) held by container %s — 'docker stop %s'\n" "$$holder" "$$holder"; \
	else \
		printf "  [ok]   port $(SITE_PORT) free\n"; \
	fi
	@if lsof -nP -iTCP:$(API_PORT) -sTCP:LISTEN >/dev/null 2>&1; then \
		if curl -fsS --max-time 3 http://localhost:$(API_PORT)/ping >/dev/null 2>&1; then \
			printf "  [ok]   backend answering on http://localhost:$(API_PORT)/ping\n"; \
		else \
			printf "  [warn] port $(API_PORT) in use but /ping does not answer\n"; \
		fi; \
	else \
		printf "  [ok]   port $(API_PORT) free (backend not running — 'make backend')\n"; \
	fi
	@printf "\n"

check-secrets:
	@if [ ! -f $(SECRETS) ]; then \
		printf "\n  Missing %s\n\n" "$(SECRETS)"; \
		printf "  The service needs PLAUSIBLE_API_KEY, GITHUB_API_KEY, SLACK_AUTH_TOKEN,\n"; \
		printf "  TURSO_AUTH_TOKEN and ENVIRONMENT. The file holds live secrets and is\n"; \
		printf "  gitignored, so a fresh clone never has it. Create it from the template:\n\n"; \
		printf "      cp %s %s\n\n" "$(TEMPLATE)" "$(SECRETS)"; \
		printf "  then fill in the real values.\n\n"; \
		exit 1; \
	fi

# --------------------------------------------------------------------- backend

build: ## Compile the Go service to go-app/arc42-status
	cd $(APP_DIR) && go build -o arc42-status

test: ## Run the Go tests
	cd $(APP_DIR) && go test ./...

lint: ## Run golangci-lint over the Go service
	cd $(APP_DIR) && golangci-lint run

# ------------------------------------------------------------------------ site

build-site: build-image ## Generate the static site into docs/_site (production config)
	$(COMPOSE) run --rm jekyll bundle exec jekyll build

build-image: ## Build the Jekyll dev image (status-arc42-site:latest)
	$(COMPOSE) build

install: build-image ## Install/refresh gems in the dev image after editing the Gemfile
	$(COMPOSE) run --rm jekyll bundle install

update: build-image ## Update gems to their latest allowed versions (rewrites Gemfile.lock)
	$(COMPOSE) run --rm jekyll bundle update

shell: build-image ## Open a shell inside the dev container
	$(COMPOSE) run --rm jekyll bash

logs: ## Tail logs from the running dev container
	$(COMPOSE) logs -f jekyll

clean: ## Remove generated site, caches, Docker volumes and the Go binary
	rm -rf $(SITE_DIR)/_site $(SITE_DIR)/.sass-cache $(SITE_DIR)/.jekyll-cache $(SITE_DIR)/.jekyll-metadata
	rm -f $(APP_DIR)/arc42-status
	@# .jekyll-cache and .sass-cache also live in named Docker volumes,
	@# so removing them on the host alone leaves stale copies behind.
	-$(COMPOSE) down -v --remove-orphans
