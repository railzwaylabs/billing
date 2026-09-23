# Development guide

## Backend

```bash
gofmt -w ./path/to/changed/files.go
go test ./...
```

Keep domain behavior independent from GORM, Gin, Fx, and wall-clock time. Add dependencies through domain interfaces and inject `clock.Clock` for business time.

## Migrations

Migration pairs live in `db/migrations`:

```text
<timestamp>_<description>.up.sql
<timestamp>_<description>.down.sql
```

```bash
migrate -path db/migrations \
  -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" up

migrate -path db/migrations \
  -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" down 1
```

Do not pass the GORM-only `timezone` URI parameter to `psql`.

## Console

```bash
corepack enable
pnpm install
pnpm dev
```

Vite proxies `/admin/v1` to the backend configured in `apps/console/vite.config.ts`. Keep list, create, and detail/edit routes separate. Use checked-in shadcn components from `src/components/ui` rather than local primitive replacements.

### Resource monitor

The **Developer → Monitor** page requires Prometheus and cAdvisor. Start the
infrastructure before running the console:

```bash
docker compose -f infrastructure/docker-compose.yml up -d
pnpm dev
```

Vite proxies `/prometheus` to `http://localhost:9090`. In the containerized
console, Nginx proxies only the Prometheus instant and range query endpoints to
the `prometheus` service on `billing-net`.

The period selector controls both the queried history and chart resolution:

| Selection | History | Query step |
| --- | --- | --- |
| Daily | Last 24 hours | 1 hour |
| Weekly | Last 7 days | 6 hours |
| Monthly | Last 30 days | 1 day |

If the page shows `Unavailable`, verify that both services are healthy and
that cAdvisor exposes metrics with
`container_label_com_docker_compose_project="billing"`.

## End-to-end tests

Install browsers once, then run tests:

```bash
pnpm test:e2e:install
pnpm test:e2e
pnpm test:e2e:report
```

Generated reports are ignored by Git.

## Adding a feature

1. Put invariants and value behavior in `domain`.
2. Define repository contracts in the domain package.
3. Implement orchestration in `application`.
4. Implement PostgreSQL adapters in `infrastructure/repository`.
5. Translate HTTP only in `transport/http`.
6. Wire constructors in the context module.
7. Add deterministic domain tests and risk-appropriate integration coverage.

## Local seed data

Developer-only seeds use `.local.sql` under `db/seeds` and are ignored. Seeds should discover an existing organization instead of hardcoding organization UUIDs.
