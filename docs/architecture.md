# Architecture

## Design goals

Billing is a financial truth system. Calculations must be deterministic, tenant boundaries explicit, and retries safe.

The code follows domain-driven and hexagonal boundaries:

```text
HTTP or worker trigger
        |
application use case
        |
domain contracts and rules
        |
PostgreSQL, Casbin, and identity adapters
```

Domain packages do not depend on Gin, GORM, Casbin, or Uber Fx. Application packages coordinate domain contracts. Infrastructure implements those contracts, and transport translates HTTP requests and responses.

## Bounded contexts

| Context | Responsibility |
| --- | --- |
| `organization` | Tenant lifecycle and initial owner binding |
| `meter` | Usage measurement and aggregation definition |
| `catalogue` | Products, prices, and graduated price tiers |
| `customer` | Billable customer records |
| `subscription` | Customer-to-price assignments and active periods |
| `usage` | Idempotent event ingestion and usage queries |
| `rating` | Aggregation, tier calculation, and invoice generation |
| `invoice` | Financial invoice and line snapshots |
| `consoleauth` | Local console sessions and Google sign-in |
| `authn` | External JWT and API-key authentication |
| `iam` | Roles, permissions, bindings, service accounts, and authorization |

Every business record is scoped by `organization_id`. Composite foreign keys prevent references to another organization's resources.

## Runtime topology

### Admin API

`cmd/admin-api` serves `/admin/v1`, console authentication, IAM management, and resource administration. It uses session-cookie authentication.

### Public API

`cmd/api` is reserved for application-facing endpoints authenticated with service-account API keys or external JWTs. Its business endpoint surface is still under development.

### Rating worker

`cmd/rating` has no business HTTP server. It exposes a private metrics and health listener on port `9090`, while Uber Fx starts the scheduler through lifecycle hooks. It:

1. Resolves the previous completed UTC calendar month.
2. Finds organizations with active subscriptions in that period.
3. Loads subscriptions, prices, products, meters, and usage.
4. Aggregates events and applies graduated tiers.
5. Creates draft invoices and line snapshots.

The invoice unique constraint on organization, customer, and period makes repeated runs idempotent. Multiple worker instances still require distributed locking to avoid races and excess load.

## Financial representation

Money and quantities use fixed-point integers in the domain:

- Money: nanos, scale 9.
- Quantity: micros, scale 6.

PostgreSQL stores corresponding `DECIMAL` values. `float64` is not authoritative for billing calculations.

## IAM evaluation

PostgreSQL is the source of truth. Bindings are compiled into an in-memory Casbin evaluator:

```text
Principal -> organization -> canonical resource -> permission
```

Policy mutations increment `iam_policy_versions` and publish `billing_iam_policy_changed`. Instances use `LISTEN/NOTIFY` for prompt reload and poll versions as recovery. A complete evaluator is swapped atomically; failed reloads preserve the previous snapshot.

## Time

Business code receives `pkg/clock.Clock`. Production uses `clock.System`; tests can use `clock.Fixed`. HTTP and Prometheus duration measurements intentionally use the system monotonic clock.

## Resource monitoring

The console resource monitor is an operational view and is separate from the
billing domain and its HTTP API:

```text
Docker containers
      |
   cAdvisor
      |
  Prometheus
      |
Nginx / Vite proxy
      |
Developer -> Monitor
```

cAdvisor exports container CPU, memory, filesystem, and network metrics.
Prometheus scrapes those metrics every five seconds. The console queries the
Prometheus range API and renders separate shadcn bar charts for:

- CPU cores used and allocated.
- Memory used and allocated, in GiB.
- Disk usage and capacity, in GiB.
- Network receive and transmit throughput, in MiB/s.

The monitor selects containers whose Docker Compose project label is
`billing`. Network is presented as throughput because Docker Compose does not
configure a bandwidth allocation limit. The Prometheus endpoint is an
observability dependency, not part of the supported public or admin billing
API contract.
