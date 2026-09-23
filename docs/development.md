# Development guide

## Backend

```bash
gofmt -w ./path/to/changed/files.go
golangci-lint config verify
golangci-lint run ./...
go test ./...
```

Keep domain behavior independent from GORM, Gin, Fx, and wall-clock time. Add dependencies through domain interfaces and inject `clock.Clock` for business time.

Application constructors receive an exported `Params` struct embedding
`fx.In`. This keeps dependency wiring declarative and prevents constructors
from growing long positional argument lists. Domain packages remain unaware of
Fx.

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
pnpm format:apps
pnpm format:apps:check
```

Vite proxies `/admin/v1` to `BACKEND_URL`, defaulting to
`http://localhost:8080`. Keep list, create, and detail/edit routes separate.
Use checked-in shadcn components from `src/components/ui` rather than local
primitive replacements.

### Resource monitor

The **Developer → Monitor** page requires Prometheus and cAdvisor. Start the
infrastructure before running the console:

```bash
docker compose -f infrastructure/docker-compose.yml up -d
pnpm dev
```

The overview calls `GET /admin/v1/monitoring/services`. Selecting a service
calls `GET /admin/v1/monitoring/services/{service}/resources?range=day|week|month`.
The admin API validates the session, selects predefined PromQL, queries
`PROMETHEUS_URL`, and returns normalized health and resource data. Prometheus
is never called directly by browser code.

The period selector controls both the queried history and chart resolution:

| Selection | History | Query step |
| --- | --- | --- |
| Daily | Last 24 hours | 1 hour |
| Weekly | Last 7 days | 6 hours |
| Monthly | Last 30 days | 1 day |

If the page shows `Unavailable`, verify that the Admin API, Prometheus, and
cAdvisor are reachable, and that cAdvisor exposes metrics with
`container_label_com_docker_compose_project="billing"`.

### Local service logs

Loki and Grafana Alloy are part of the infrastructure stack:

```bash
docker compose -f infrastructure/docker-compose.yml up -d loki alloy
docker compose -f infrastructure/docker-compose.yml logs -f loki alloy
```

The root Compose stack configures admin-api with `LOGS_PROVIDER=loki` and
`LOGS_URL=http://loki:3100`. Alloy discovers the root Compose project through
the Docker socket, collects JSON stdout from `admin-api`, `api`, and `rating`,
and forwards it to Loki. Open **Developer → Logs** after applying the IAM
migration.

For infrastructure diagnosis only:

```bash
curl http://localhost:3100/ready
curl -G http://localhost:3100/loki/api/v1/labels
```

Normal Console traffic uses the authenticated and IAM-protected
`/admin/v1/logs` API rather than connecting to Loki directly.

For a Vercel deployment, set the project root to `apps/console`. Either keep
`VITE_BACKEND_URL` empty and rewrite `/admin/v1/*` to admin-api, or set it to
the public HTTPS origin of admin-api. Direct cross-origin requests require the
backend to allow the exact console origin and credentials; do not use `*`.

## End-to-end tests

Install browsers once, then run tests:

```bash
pnpm test:e2e:install
pnpm test:e2e
pnpm test:e2e:report
```

Generated reports are ignored by Git.

## Testing authorization changes

IAM behavior should be tested at both the domain boundary and the compiled
Casbin evaluator. Include cases for default-deny, parent-resource inheritance,
organization isolation, issuer isolation, stale ETags, duplicate bindings, and
last-owner protection. Repository tests should also verify policy-version
increments and reload behavior.

Do not bypass authorization in handlers to simplify a test. Construct a
principal with an explicit policy binding or assert the expected
`PERMISSION_DENIED` response. See [IAM policies](iam.md) for the model and error
codes.

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
