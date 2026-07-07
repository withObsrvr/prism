# ──────────────────────────────────────────────
# Prism Makefile
# ──────────────────────────────────────────────

# Build metadata injected via ldflags
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS   = -s -w \
            -X github.com/withObsrvr/prism/cmd/prism.version=$(VERSION) \
            -X github.com/withObsrvr/prism/cmd/prism.commit=$(COMMIT) \
            -X github.com/withObsrvr/prism/cmd/prism.date=$(DATE) \
            -X github.com/withObsrvr/prism/internal/buildinfo.Version=$(VERSION) \
            -X github.com/withObsrvr/prism/internal/buildinfo.Commit=$(COMMIT) \
            -X github.com/withObsrvr/prism/internal/buildinfo.Date=$(DATE)

.PHONY: help build run dev generate css clean test vet

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────

build: generate css ## Build the binary
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o bin/prism .

run: build ## Build and run
	./bin/prism serve

# ──────────────────────────────────────────────
# Development
# ──────────────────────────────────────────────

dev: ## Run with live reload (requires air)
	@echo "Starting templ watch + air..."
	@templ generate --watch &
	@air

generate: ## Generate templ files
	templ generate

css: ## Build Tailwind CSS
	npx @tailwindcss/cli -i web/static/css/input.css -o web/static/css/app.css --minify

css-watch: ## Watch Tailwind CSS
	npx @tailwindcss/cli -i web/static/css/input.css -o web/static/css/app.css --watch

# ──────────────────────────────────────────────
# Quality
# ──────────────────────────────────────────────

test: ## Run tests
	go test ./... -race -count=1

vet: ## Run go vet
	go vet ./...

clean: ## Remove build artifacts
	rm -rf bin/ tmp/
	find . -name '*_templ.go' -delete
