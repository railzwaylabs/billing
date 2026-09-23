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
| `METRICS_ADDRESS` | `0.0.0.0:9090` | Private observability listener for metrics and probes |
| `MANAGEMENT_ADDRESS` | `127.0.0.1:7070` | Private management listener for pprof and admin runtime controls |
| `RATING_INTERVAL` | `1m` | Scheduler check interval |

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

Budget PostgreSQL connections across all processes. Three processes configured for 20 connections can consume up to 60 pooled connections, plus dedicated listener connections.

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
