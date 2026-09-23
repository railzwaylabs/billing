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
