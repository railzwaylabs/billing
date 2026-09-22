# Billing

## Metrics

`admin-api`, `api`, and `rating` expose Prometheus metrics at `GET /metrics` on
their HTTP address. The endpoint includes Go runtime, process CPU/memory, and
HTTP request metrics.

Configure deployment identity through:

- `BILLING_ORGANIZATION_ID` (defaults to `unknown`)
- `BILLING_PROJECT_ID` (defaults to `unknown`)
- `BILLING_ADMIN_ADDRESS` (defaults to `:8081`)
- `BILLING_API_ADDRESS` (defaults to `:8080`)
- `BILLING_RATING_ADDRESS` (defaults to `:8082`)

`organization_id`, `project_id`, and `service` are constant labels on every
application metric. Set them from trusted IaC/deployment metadata, not from an
HTTP request.

Disk and network usage are host/container metrics rather than application
metrics. Collect them with the platform exporter (for example Nomad allocation
metrics, cAdvisor, or node_exporter) and attach the same organization/project
labels in the Prometheus scrape or service-discovery configuration. This lets
dashboards join and aggregate all four resource dimensions consistently.
