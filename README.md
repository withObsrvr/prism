# Prism

**Soroban-first block explorer for the Stellar network.**

Built by [Obsrvr](https://withobsrvr.com). Powered by Obsrvr Lake.

## Architecture

Prism is a server-rendered Go web application. There is no SPA, no React, no client-side router. Pages are HTML rendered on the server, with htmx handling the interactive bits.

```
Go 1.26 · Cobra/Viper CLI · net/http · Templ · Tailwind CSS · htmx
```

The application structure follows Alex Edwards' patterns from _Let's Go_ and _Let's Go Further_:

- **Dependency injection via struct** — the `Application` struct holds logger, config, and future database connections. Handlers receive dependencies through the struct receiver, never via globals or closures.
- **Timeouts on every server** — `ReadTimeout`, `WriteTimeout`, `IdleTimeout` set explicitly.
- **Graceful shutdown** — catches SIGINT/SIGTERM, drains connections with a 30s deadline.
- **Structured logging** — `log/slog` throughout, with configurable log level.
- **Middleware chain** — `recoverPanic(logRequest(router))`, applied in `Routes()`.

## Project Structure

```
prism/
├── main.go                          # Entry point — calls cmd/prism.Execute()
├── go.mod                           # Go 1.25.0 (built with Go 1.26)
├── prism.yaml                       # Default config (overridden by env vars)
├── Makefile                         # build, run, dev, generate, css, test
├── .air.toml                        # Live reload config
│
├── cmd/prism/                       # Cobra commands
│   ├── root.go                      # Root command + viper config init
│   ├── serve.go                     # `prism serve` — starts HTTP server
│   └── version.go                   # `prism version` — build info
│
├── internal/
│   ├── server/
│   │   ├── application.go           # Application struct (DI container)
│   │   ├── routes.go                # All routes in one place
│   │   └── middleware.go            # logRequest, recoverPanic
│   │
│   ├── handlers/
│   │   ├── handlers.go              # Handlers struct, isHTMX helper
│   │   ├── explorer.go              # Home, Search, LedgerDetail, TransactionReceipt
│   │   ├── network.go               # NetworkHealth, ValidatorDetail, ValidatorPreview
│   │   ├── assets.go                # AssetDirectory, AssetDetail
│   │   ├── contracts.go             # ContractList, ContractDetail, ContractEvents
│   │   ├── accounts.go              # AccountPortfolio, SmartAccountDashboard
│   │   └── devtools.go              # EventsFirehose, StateRentTracker, LiveFeed
│   │
│   └── templates/
│       ├── layouts/
│       │   └── base.templ           # HTML shell, top nav, slide-out container
│       ├── components/
│       │   └── shared.templ         # NetworkPulse, StatCard, TruncatedAddress
│       └── pages/
│           └── home.templ           # Search-first landing page
│
└── web/
    └── static/
        ├── css/
        │   └── input.css            # Tailwind input + Prism design tokens
        └── js/
            └── htmx.min.js          # htmx (vendored, no CDN in production)
```

## Route Map

Every route maps to a designed page from the Prism design system.

| Route | Handler | Template | Nav | htmx |
|---|---|---|---|---|
| `GET /` | `Home` | `pages/home` | explorer | Live search, live feed |
| `GET /search?q=` | `Search` | `pages/search` | explorer | |
| `GET /ledger/{seq}` | `LedgerDetail` | `pages/ledger_detail` | explorer | |
| `GET /tx/{hash}` | `TransactionReceipt` | `pages/transaction_receipt` | explorer | |
| `GET /network` | `NetworkHealth` | `pages/network_health` | network | |
| `GET /network/validators/{id}` | `ValidatorDetail` | `pages/validator_detail` | network | Tabs |
| `GET /assets` | `AssetDirectory` | `pages/asset_directory` | assets | |
| `GET /assets/{code}-{issuer}` | `AssetDetail` | `pages/asset_detail` | assets | |
| `GET /contracts` | `ContractList` | `pages/contract_list` | contracts | |
| `GET /contracts/{id}` | `ContractDetail` | `pages/contract_detail` | contracts | Tabs |
| `GET /contracts/{id}/events` | `ContractEvents` | partial or full | contracts | Tab swap |
| `GET /account/{id}` | `AccountPortfolio` | `pages/account_portfolio` | explorer | |
| `GET /account/{id}/smart` | `SmartAccountDashboard` | `pages/smart_account` | explorer | |
| `GET /events` | `EventsFirehose` | `pages/events_firehose` | devtools | SSE stream |
| `GET /state` | `StateRentTracker` | `pages/state_rent_tracker` | devtools | |

### htmx Partials

These routes return HTML fragments, not full pages:

| Route | Trigger | Target |
|---|---|---|
| `GET /partials/search-results?q=` | `keyup changed delay:300ms` | `#search-live-results` |
| `GET /partials/live-feed` | `every 3s` | Home page feed area |
| `GET /network/validators/{id}/preview` | Row click | `#slideout` (480px panel) |

## Getting Started

### Prerequisites

- [Nix](https://nixos.org/download/) with flakes enabled

### Setup

```bash
# Enter the development shell (provides Go, templ, Node.js, air, etc.)
nix develop

# Install npm dependencies (Tailwind CSS)
npm install

# Install Go dependencies
go mod tidy

# Generate templ files and build Tailwind
make generate
make css

# Build and run
make run

# Or use live reload for development
make dev
```

The server starts at `http://localhost:3000` by default.

### Configuration

Configuration is resolved in this order (last wins):

1. Defaults in code
2. `prism.yaml` in the working directory
3. `~/.prism/prism.yaml`
4. `/etc/prism/prism.yaml`
5. Environment variables with `PRISM_` prefix

```bash
# Override via environment
PRISM_PORT=8080 PRISM_NETWORK=testnet prism serve

# Override via flags
prism serve --port 8080 --network testnet
```

## Design System

The visual language is documented in `prism-design-system.html` — a self-contained reference covering colors, typography, spacing, every component, and interaction patterns.

Key decisions:
- **Instrument Sans** for UI, **JetBrains Mono** for blockchain data
- **Emerald** for links and success, **Violet** for Soroban/contracts
- **Light theme** for the explorer product, **dark theme** for marketing only
- **480px slide-out panels** (Ramp pattern) for entity previews
- **Progressive disclosure** — technical details collapsed by default

## SCF #42

Prism is built as part of [Stellar Community Fund #42](https://communityfund.stellar.org). It extends the Obsrvr Intelligence Console with a public-facing block explorer focused on Soroban smart contracts.
