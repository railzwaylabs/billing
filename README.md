# Billing

An open-source usage-based billing engine for metering events, defining tiered prices, rating usage, and generating deterministic invoices.

Billing answers **what should be billed**. Payment collection, card storage, and payment-provider orchestration are intentionally outside this repository.

## Features

- Organization-scoped meters, products, prices, customers, and subscriptions.
- Batch usage ingestion with event deduplication and 24-hour request idempotency.
- Fixed-point quantities and money; authoritative calculations do not use floating point.
- Graduated price tiers and deterministic invoice lines.
- Monthly rating worker with idempotent invoice generation.
- GCP-style IAM roles, policies, service accounts, and `sk_live_` API keys.
- Local console authentication, optional Google OIDC, and external JWT verification.
- React console using Vite, Tailwind CSS, and shadcn components.
- Prometheus metrics, Zap logging, and PostgreSQL policy synchronization.
- Developer monitoring console with separate CPU, memory, disk, and network
  bar charts for daily, weekly, and monthly ranges.

## Components

| Component | Purpose | Default address |
| --- | --- | --- |
| `cmd/admin-api` | Console authentication and administration API | HTTP `:8080`, metrics `:9090` |
| `cmd/api` | Public API process; currently health and IAM runtime only | `:8080` |
| `cmd/rating` | Background rating and invoice-generation worker | Metrics `:9090`; no business HTTP |
| `apps/console` | Billing administration console | `:5173` |

`rating.WorkerModule` may be embedded in `admin-api` for an all-in-one deployment or run through `cmd/rating`. Do not run both concurrently without distributed locking.

## Requirements

- Go 1.25+
- PostgreSQL with `pgcrypto` and `LISTEN/NOTIFY`
- Node.js and pnpm for the console
- A migration runner such as [golang-migrate](https://github.com/golang-migrate/migrate)

## Quick start

1. Create local configuration:

   ```bash
   cp .env.example .env
   ```

2. Apply migrations:

   ```bash
   migrate -path db/migrations \
     -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" up
   ```

3. Set a development API-key hashing secret in `.env`:

   ```env
   API_KEY_SECRET=replace-with-at-least-32-random-characters
   ```

4. Start the admin backend:

   ```bash
   go run ./cmd/admin-api
   ```

5. Install dependencies and start the console:

   ```bash
   corepack enable
   pnpm install
   pnpm dev
   ```

6. Open `http://localhost:5173`. The default development bootstrap credentials are `admin` / `admin`. Never use these credentials outside local development.

7. If the worker is not embedded in `admin-api`, run it separately:

   ```bash
   go run ./cmd/rating
   ```

## Docker Compose

Start the shared infrastructure first, then the billing services:

```bash
docker compose -f infrastructure/docker-compose.yml up -d
docker compose up --build
```

The infrastructure stack starts PostgreSQL, Prometheus, and cAdvisor. The
billing stack applies migrations and then starts:

| Service | Local address |
| --- | --- |
| Console | http://localhost:5173 |
| Admin API | http://localhost:8080 |
| Public API | http://localhost:8081 |
| Admin metrics | http://localhost:9091/metrics |
| Public API metrics | http://localhost:9092/metrics |
| Rating metrics | http://localhost:9093/metrics |
| PostgreSQL | localhost:5432 |
| Rating worker | Metrics only on `localhost:9093` |
| Prometheus | http://localhost:9090 |
| cAdvisor | http://localhost:8082 |

After signing in, open **Developer → Monitor** to view resource history.
Daily shows the last 24 hours, Weekly shows the last 7 days, and Monthly shows
the last 30 days. Prometheus retains 31 days of local metrics.

Follow logs or stop the stack with:

```bash
docker compose logs -f admin-api rating
docker compose down
docker compose -f infrastructure/docker-compose.yml down
```

To also remove the local PostgreSQL volume and start with an empty database:

```bash
docker compose down -v
```

Compose defaults are for local development only. Override `DATABASE_PASSWORD`, `API_KEY_SECRET`, and `BOOTSTRAP_ADMIN_PASSWORD` through the shell or a local `.env` before using a shared environment. See [Configuration](docs/configuration.md) for production secret-management guidance.

## Billing flow

```text
Create organization
  -> create meter
  -> create product and price tiers
  -> create customer and subscription
  -> ingest usage events
  -> rate the completed period
  -> generate draft invoice and lines
```

Rating aggregates immutable usage events for a half-open period `[start, end)`, applies subscribed price tiers, and persists auditable invoice-line snapshots.

## Project structure

```text
apps/console/               React administration console
cmd/admin-api/              Admin HTTP process
cmd/api/                    Public HTTP process
cmd/rating/                 Background rating process
db/migrations/              Ordered PostgreSQL migrations
internal/<context>/domain/  Entities, rules, and repository contracts
internal/<context>/application/
                            Use cases and orchestration
internal/<context>/infrastructure/
                            PostgreSQL, Casbin, and external adapters
internal/<context>/transport/http/
                            Gin handlers
internal/platform/          Database, HTTP, logging, and metrics
pkg/clock/                  Injectable system and fixed clocks
```

## Commands

```bash
# Backend
go test ./...
go run ./cmd/admin-api
go run ./cmd/api
go run ./cmd/rating

# Console
pnpm install
pnpm dev
pnpm build
pnpm lint

# Browser tests (install browsers once)
pnpm test:e2e:install
pnpm test:e2e
pnpm test:e2e:report
```

## Documentation

- [Architecture](docs/architecture.md)
- [Configuration](docs/configuration.md)
- [Billing and rating](docs/billing-and-rating.md)
- [HTTP API](docs/http-api.md)
- [Development guide](docs/development.md)

## Current limitations

- The public API surface is not complete.
- The scheduler currently closes the previous UTC calendar month.
- Explicit IAM deny and conditional bindings are not implemented.
- Payment collection is out of scope.
- Multiple rating workers require distributed coordination for production.

## Security

- Keep `.env`, API-key secrets, OAuth secrets, and bootstrap credentials out of Git.
- Use a least-privilege PostgreSQL role in deployed environments.
- Enable secure cookies behind HTTPS.
- Replace the bootstrap password immediately or disable bootstrapping after initialization.
