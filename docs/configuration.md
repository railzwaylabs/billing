# Configuration

Viper loads configuration from `.env` and process environment variables. Environment variables override file values. A missing `.env` is valid in deployed environments.

Start from `.env.example`. `config.example.yaml` documents the same settings for deployment tooling, but the current application loader reads `.env` and environment variables.

## Secrets and secure configuration

`.env` is intended for local development only. In staging and production, inject sensitive values at runtime from a secret-management system such as:

- Google Cloud Secret Manager
- AWS Secrets Manager or Systems Manager Parameter Store
- HashiCorp Vault or Nomad Variables
- Kubernetes Secrets backed by an external secret manager

Do not commit real secrets, place them in `config.example.yaml`, bake them into a container image, or expose them through frontend environment variables. Restrict secret access to the service identity that needs it, enable audit logging, and rotate secrets periodically.

### Values that must be secret

| Variable | Why it is sensitive | Rotation impact |
| --- | --- | --- |
| `DATABASE_PASSWORD` | Grants access to billing and identity data | Rotate the database credential and restart every process using it |
| `API_KEY_SECRET` | HMAC secret used to verify every `sk_live_` key | Changing it invalidates existing API keys; plan a staged rotation or reissue keys |
| `BOOTSTRAP_ADMIN_PASSWORD` | Grants initial administrator access | Use only during bootstrap, then disable bootstrap and remove the secret |
| `AUTH_GOOGLE_CLIENT_SECRET` | OAuth client credential accepted by Google | Rotate in Google and update the runtime secret before revoking the old value |
| `LOGS_PASSWORD` | Credential used by admin-api to query the configured log provider | Rotate it in the provider and restart admin-api |

Use independently generated values for each environment. Never reuse development secrets in staging or production.

### Sensitive but usually not secret

The following values may reveal infrastructure or identity topology, but they are not authentication credentials by themselves:

- `DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_NAME`, and `DATABASE_USER`
- `IAM_ISSUER`, `IAM_AUDIENCE`, and `IAM_JWKS_URL`
- `AUTH_GOOGLE_CLIENT_ID`, discovery URL, redirect URL, and allowed domains
- deployment metric labels

They may remain in deployment configuration, subject to the organization's infrastructure-disclosure policy.

### Runtime handling

- Deliver secrets directly to the process environment or a protected runtime file supplied by the orchestrator.
- Do not print configuration structs or DSNs in logs.
- Avoid exposing secrets in command-line arguments because process listings and CI logs may capture them.
- Grant the PostgreSQL account only the privileges required by the selected process.
- Prefer short-lived workload identity credentials where the deployment platform supports them.
- Restart processes after rotating a secret because the current application reads configuration at startup.

## Runtime

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_ADDRESS` | `0.0.0.0:8080` | Business HTTP listener for `admin-api` or `api` |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | Comma-separated browser origins allowed with credentials |
| `METRICS_ADDRESS` | `0.0.0.0:9090` | Private observability listener for metrics and probes |
| `MANAGEMENT_ADDRESS` | `127.0.0.1:7070` | Private management listener for pprof and admin runtime controls |
| `RATING_INTERVAL` | `1m` | Scheduler check interval |
| `PROMETHEUS_URL` | `http://localhost:9090` | Private Prometheus origin used by admin-api monitoring |
| `PROMETHEUS_TIMEOUT` | `10s` | Timeout for each Prometheus range query |
| `LOGS_PROVIDER` | `disabled` | Admin log-query provider; supported values are `disabled` and `loki` |
| `LOGS_URL` | `http://localhost:3100` | Private Loki origin used only by admin-api |
| `LOGS_TENANT_ID` | empty | Optional Loki `X-Scope-OrgID` tenant |
| `LOGS_USERNAME` | empty | Optional Loki basic-auth username |
| `LOGS_PASSWORD` | empty | Optional Loki basic-auth password |
| `LOGS_TIMEOUT` | `10s` | Timeout for each Loki query |
| `LOGS_SERVICE_LABEL` | `billing_service` | Loki label identifying `admin-api`, `public-api`, or `rating` |

The rating command ignores the business HTTP address. In production, route port `8080` through ingress, restrict port `9090` to the monitoring plane, and keep the management listener private. Pprof may expose memory contents and runtime details, so never publish `MANAGEMENT_ADDRESS` through public ingress.

## PostgreSQL

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE_HOST` | `localhost` | PostgreSQL host |
| `DATABASE_PORT` | `5432` | PostgreSQL port |
| `DATABASE_NAME` | `postgres` | Database name |
| `DATABASE_USER` | `postgres` | Database user |
| `DATABASE_PASSWORD` | Development value | Database password; inject from secret management in production |
| `DATABASE_SSL_MODE` | `disable` | PostgreSQL SSL mode |
| `DATABASE_MAX_OPEN_CONNS` | `20` | Maximum open connections per process |
| `DATABASE_MAX_IDLE_CONNS` | `5` | Retained idle connections |
| `DATABASE_CONN_MAX_LIFETIME` | `30m` | Maximum connection lifetime |
| `DATABASE_CONN_MAX_IDLE_TIME` | `5m` | Maximum connection idle duration |
| `DATABASE_LOG_LEVEL` | `warn` | GORM JSON verbosity: `silent`, `error`, `warn`, or `info` |
| `DATABASE_SLOW_QUERY_THRESHOLD` | `200ms` | Duration after which a query is logged as slow at `warn` level |

Budget PostgreSQL connections across all processes. Three processes configured for 20 connections can consume up to 60 pooled connections, plus dedicated listener connections.

Fx lifecycle events and GORM events use the same Zap JSON stream as application logs. GORM statements are logged with bind placeholders and never include parameter values. The default `warn` level records failed queries and queries slower than `DATABASE_SLOW_QUERY_THRESHOLD`; use `info` only when full query visibility is needed.

The application builds one canonical PostgreSQL DSN from these values. GORM uses it for the main connection pool, while IAM uses the same DSN for a dedicated `LISTEN/NOTIFY` connection. A separate `DATABASE_URI` is not required.

## Console sessions

| Variable | Default | Description |
| --- | --- | --- |
| `SESSION_COOKIE_NAME` | `_billing_session` | Session cookie name |
| `SESSION_TTL` | `24h` | Session validity |
| `SESSION_COOKIE_SECURE` | `false` | Require HTTPS for cookies |

## Bootstrap administrator

`BOOTSTRAP_ADMIN_ENABLED` creates the first user only when no user exists. The example username and password are both `admin`; change them for development and never use them in production. Disable bootstrapping after production initialization.

## Google authentication

Set `AUTH_GOOGLE_ENABLED=true`, provide client credentials, and register `AUTH_GOOGLE_REDIRECT_URL` with Google. The discovery document supplies authorization, token, issuer, and JWKS endpoints. All related variables are listed in `.env.example`.

## IAM and API keys

| Variable | Description |
| --- | --- |
| `IAM_ISSUER` | Expected external JWT issuer |
| `IAM_AUDIENCE` | Expected JWT audience |
| `IAM_JWKS_URL` | JWKS used for signature verification |
| `API_KEY_SECRET` | HMAC secret used to hash `sk_live_` API keys |

Use at least 32 random characters for `API_KEY_SECRET` and a different value per environment. Only API-key hashes are persisted; plaintext is returned once.

## Metrics labels

`BILLING_ORGANIZATION_ID` and `BILLING_PROJECT_ID` are deployment metadata attached as Prometheus labels. They do not select a billing tenant and do not affect authorization.

## Console resource monitoring

The local monitoring stack is configured in
`infrastructure/docker-compose.yml` and
`infrastructure/prometheus/prometheus.yml`:

| Setting | Local value | Purpose |
| --- | --- | --- |
| Prometheus scrape interval | `5s` | Collect cAdvisor and billing-process metrics |
| Prometheus retention | `31d` | Support the Monthly console range |
| Prometheus address | `localhost:9090` | Local Prometheus UI and query API |
| cAdvisor address | `localhost:8082` | Local container metrics endpoint |
| Loki address | `localhost:3100` | Local log ingestion and query API |
| Alloy address | `localhost:12345` | Local collector diagnostics UI |

The browser calls session-authenticated `/admin/v1/monitoring/services`
endpoints with the active organization slug. The admin API additionally
requires `billing.monitoring.get` through the organization's IAM policy. It
owns the service registry and PromQL, and connects to `PROMETHEUS_URL`; it does
not accept arbitrary PromQL from clients. Keep Prometheus on the private
network and do not publish it through ingress.

Monthly charts require at least 30 days of retained samples. If retention is
reduced, the console remains usable but can only display the history still
available in Prometheus.

## Console log queries

Log queries are disabled by default. Set `LOGS_PROVIDER=loki` on admin-api and
configure the private `LOGS_URL` to enable them. Workloads continue writing
structured JSON to stdout; the deployment collector is responsible for adding
the configured service label and forwarding records to Loki. This keeps the
admin API contract unchanged across Docker Compose, Kubernetes, and Nomad.

The browser never receives Loki credentials or submits arbitrary LogQL. It
calls session-authenticated `/admin/v1/logs` endpoints with the active
organization slug. The admin API requires `billing.logs.list`, then builds
allow-listed queries with a maximum 30-day range and 500 records per request.
Keep Loki private and store its password in secret management.

The root Docker Compose file enables Loki for admin-api with
`LOGS_URL=http://loki:3100`. The infrastructure stack runs single-process Loki
with 31-day filesystem retention and Grafana Alloy as its Docker collector.
Alloy selects Compose project `billing`, keeps `admin-api`, `api`, and `rating`,
and normalizes the `api` service label to `public-api`.

The local collector mounts `/var/run/docker.sock` read-only. Access to the
Docker socket is still highly privileged, so this topology is intended only
for development. Kubernetes should run Alloy as a DaemonSet using pod-log
discovery. Nomad should run Alloy or Vector as a system job reading allocation
logs. In either environment, emit the stable
`billing_service=admin-api|public-api|rating` label so the backend and Console
remain independent from the scheduler.

For a separately hosted console, `VITE_BACKEND_URL` is embedded into the
frontend at build time. Set it to an HTTPS origin such as
`https://admin-api.example.com`, without a trailing slash. When it is empty,
the console uses same-origin `/admin/v1` requests, suitable for an ingress or
Vercel rewrite. `BACKEND_URL` configures only the local Vite development proxy.
`VITE_APP_NAME` controls the browser title and defaults to `Billing Console`.
Both `VITE_*` values are build-time configuration; rebuild the console image
after changing them.
