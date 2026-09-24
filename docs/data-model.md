# Domain data model

This document groups the PostgreSQL schema by bounded context. Every tenant
owned table carries `organization_id`; composite foreign keys keep references
inside the same organization.

## Organization and console identity

```mermaid
erDiagram
    USERS ||--o| USER_PASSWORD_CREDENTIALS : authenticates
    USERS ||--o{ USER_IDENTITIES : links
    USERS ||--o{ SESSIONS : opens
    ORGANIZATIONS ||--o{ IAM_POLICY_BINDINGS : owns
```

- `users`, `user_password_credentials`, `user_identities`, and `sessions` serve
  the administrative console only.
- Application customers are stored separately in `customers`.
- Organizations are the tenant boundary for billing and IAM data.

## IAM

```mermaid
erDiagram
    IAM_SERVICES ||--o{ IAM_RESOURCE_TYPES : defines
    IAM_SERVICES ||--o{ IAM_PERMISSIONS : publishes
    IAM_RESOURCE_TYPES ||--o{ IAM_PERMISSIONS : scopes
    ORGANIZATIONS ||--o{ IAM_ROLES : owns_custom
    IAM_ROLES ||--o{ IAM_ROLE_PERMISSIONS : grants
    IAM_PERMISSIONS ||--o{ IAM_ROLE_PERMISSIONS : included_in
    IAM_ROLES ||--o{ IAM_POLICY_BINDINGS : bound_by
    ORGANIZATIONS ||--o{ IAM_SERVICE_ACCOUNTS : owns
    IAM_SERVICE_ACCOUNTS ||--o{ IAM_API_KEYS : authenticates_with
    ORGANIZATIONS ||--|| IAM_POLICY_VERSIONS : versions
    ORGANIZATIONS ||--o{ IAM_AUDIT_LOGS : audits
```

PostgreSQL is authoritative. Policy bindings are compiled into Casbin and
atomically swapped in memory. API-key rows store hashes, never plaintext keys.

## Meter and catalogue

```mermaid
erDiagram
    MEASUREMENT_UNITS ||--o{ METERS : constrains
    ORGANIZATIONS ||--o{ METERS : owns
    ORGANIZATIONS ||--o{ PRODUCTS : owns
    PRODUCTS ||--o{ PRICES : versions
    CURRENCIES ||--o{ PRICES : denominates
    PRICES ||--|{ PRICE_CHARGES : contains
    METERS ||--o{ PRICE_CHARGES : measures
    PRICE_CHARGES ||--|{ CHARGE_TIERS : prices
```

- A product is the catalogue root.
- A price is an effective-dated version for one product and currency.
- A price charge connects one meter to `per_unit` or `graduated` pricing.
- A per-unit charge has one zero-based tier; graduated tiers have strictly
  increasing quantity boundaries.
- `currencies` and `measurement_units` are reference tables used by both the
  backend and searchable console selectors.

## Customer and subscription

```mermaid
erDiagram
    ORGANIZATIONS ||--o{ CUSTOMERS : owns
    CUSTOMERS ||--o{ SUBSCRIPTIONS : subscribes
    SUBSCRIPTIONS ||--|{ SUBSCRIPTION_ITEMS : contains
    PRICES ||--o{ SUBSCRIPTION_ITEMS : selected_by
```

Subscription items are effective-dated. Enabling another product during a
billing cycle adds an item to the existing subscription. History is ended with
`end_at`; rows are not deleted by the application update workflow.

## Usage ingestion

```mermaid
erDiagram
    ORGANIZATIONS ||--o{ IDEMPOTENCY_KEYS : scopes
    ORGANIZATIONS ||--o{ USAGE_EVENTS : owns
    METERS ||--o{ USAGE_EVENTS : measures
    CUSTOMERS ||--o{ USAGE_EVENTS : consumes
```

`idempotency_keys` protects request retries for 24 hours. The permanent unique
event identity is `(organization_id, event_id)`, so expiring an idempotency key
cannot duplicate usage.

## Rating and invoice

```mermaid
erDiagram
    CUSTOMERS ||--o{ INVOICES : billed
    INVOICES ||--|{ INVOICE_LINES : contains
    SUBSCRIPTIONS ||--o{ INVOICE_LINES : sources
    SUBSCRIPTION_ITEMS ||--o{ INVOICE_LINES : sources
    PRODUCTS ||--o{ INVOICE_LINES : snapshots
    PRICES ||--o{ INVOICE_LINES : snapshots
    PRICE_CHARGES ||--o{ INVOICE_LINES : snapshots
    METERS ||--o{ INVOICE_LINES : snapshots
    ORGANIZATIONS ||--|| INVOICE_NUMBER_SETTINGS : configures
    ORGANIZATIONS ||--o{ INVOICE_NUMBER_SEQUENCES : allocates
```

Rating intersects billing, subscription, subscription-item, and price periods,
then creates one line for every charge with billable usage. Each line persists
fixed-point quantity, unit price, amount, and tier details so the calculation
can be audited after the catalogue changes.

The MVP stops at draft invoice generation. Payment, credit notes, tax engines,
commitments, and ledger accounting remain outside the current schema.
