# HTTP API

## Conventions

- Admin prefix: `/admin/v1`.
- Organization resources: `/admin/v1/organizations/{organization_id}/...`.
- Console endpoints use the configured session cookie.
- Usage ingestion requires `Idempotency-Key`.
- Collection endpoints accept `limit` (default `25`, maximum `100`) and an opaque `cursor`. Responses include `page_info.next_cursor` and `page_info.has_more`.
- Errors use a stable envelope:

```json
{
  "error": {
    "code": "RESOURCE_ERROR_CODE",
    "message": "Human-readable message",
    "details": []
  }
}
```

## Observability listener

These endpoints are served from `METRICS_ADDRESS`, separately from business HTTP traffic:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/healthz` | Process health |
| `GET` | `/readyz` | Process readiness |
| `GET` | `/metrics` | Prometheus metrics |

Do not publish the observability port through the public ingress.

## Management listener

These endpoints are served from `MANAGEMENT_ADDRESS`, separately from business and metrics traffic:

| Process | Method | Path | Purpose |
| --- | --- | --- | --- |
| `admin-api`, `api`, `rating` | `GET` | `/debug/pprof/` | Go runtime profiles |
| `admin-api` only | `GET` | `/log/mode` | Read the active runtime log level |
| `admin-api` only | `PUT` | `/log/mode` | Set `debug`, `info`, `warn`, or `error` using `{"level":"debug"}` |

Keep this listener reachable only from trusted operational networks.

## Console authentication

| Method | Path |
| --- | --- |
| `POST` | `/admin/v1/auth/login` |
| `GET` | `/admin/v1/auth/session` |
| `POST` | `/admin/v1/auth/logout` |
| `POST` | `/admin/v1/auth/password` |
| `POST` | `/admin/v1/auth/password/skip` |
| `GET` | `/admin/v1/auth/providers` |
| `GET` | `/admin/v1/auth/providers/google/login` |
| `GET` | `/admin/v1/auth/providers/google/callback` |

## Organizations

| Method | Path |
| --- | --- |
| `GET`, `POST` | `/admin/v1/organizations` |
| `PATCH` | `/admin/v1/organizations/{organization_id}` |

## Organization resources

Meters, products, prices, customers, subscriptions, and invoices support list, create, get, and update operations below:

```text
/admin/v1/organizations/{organization_id}/{collection}
```

Invoice numbering settings:

| Method | Path |
| --- | --- |
| `GET`, `PATCH` | `/admin/v1/organizations/{organization_id}/invoice-number-settings` |

Usage endpoints:

| Method | Path |
| --- | --- |
| `GET`, `POST` | `/admin/v1/organizations/{organization_id}/usage-events` |
| `GET` | `/admin/v1/organizations/{organization_id}/usage-events/summary` |
| `GET` | `/admin/v1/organizations/{organization_id}/usage-events/{usage_event_id}` |

Invoice generation is performed by the rating worker rather than an HTTP endpoint.

## Monitoring

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/admin/v1/monitoring/resources?range=day\|week\|month` | Session-authenticated CPU, memory, disk, and network history |

The monitoring endpoint accepts only the documented range selector. PromQL,
start/end timestamps, step size, and container selectors are controlled by the
admin API. Prometheus remains a private infrastructure dependency. An invalid
range returns `MONITORING_RANGE_INVALID`; an upstream failure or timeout
returns `MONITORING_UNAVAILABLE`.

## IAM

IAM routes below `/admin/v1/iam` provide policies, permission tests, custom roles, service accounts, and API-key lifecycle operations. Authorization is default-deny and uses canonical organization resource names.

See [IAM policies](iam.md) for the complete authorization model, role and
permission catalogue, ETag behavior, and request examples.

The console Members page resolves normalized local/social identities through `GET /admin/v1/iam/users?organization={slug}` and mutates organization membership through the existing ETag-protected IAM policy endpoint.

The current admin surface serves the bundled console and is not yet a public compatibility commitment. An OpenAPI contract should be published before `cmd/api` is considered stable.
