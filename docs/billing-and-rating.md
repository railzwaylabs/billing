# Billing and rating

## Metering

A meter describes what an event value measures:

- `code`: stable identifier such as `api_calls`.
- `unit`: human-readable quantity such as `request`, `token`, or `GB-hour`.
- `aggregation`: `count` or `sum`.

The unit is independent from the code. `code=api_calls` and `unit=request` is valid.

## Usage ingestion

One batch endpoint accepts one or many events. Every event has a caller-defined `event_id`; PostgreSQL enforces uniqueness on `(organization_id, event_id)`.

Requests also carry `Idempotency-Key`. The same key and body return the recorded result. Reusing a key with another body is rejected. Idempotency records expire after 24 hours; event IDs remain unique independently.

```json
{
  "events": [
    {
      "event_id": "evt-001",
      "meter_id": "00000000-0000-0000-0000-000000000000",
      "customer_id": "00000000-0000-0000-0000-000000000000",
      "value_micros": 10000000,
      "event_time": "2026-09-22T10:00:00Z"
    }
  ]
}
```

## Prices and tiers

Every price has at least one tier. A non-tiered price is one tier starting at zero. Tier starts must be unique.

| Start quantity | Unit amount |
| ---: | ---: |
| 0 | USD 0.50 |
| 1,000 | USD 0.35 |
| 10,000 | USD 0.20 |

Usage is charged progressively across tiers. Invoice lines persist the pricing breakdown used by the calculation.

## Rating period

The scheduler currently rates the previous UTC calendar month as a half-open period:

```text
[first day of previous month 00:00 UTC, first day of current month 00:00 UTC)
```

An event at the exact period end belongs to the next period. Subscription end dates entered at midnight are treated inclusively through that calendar day.

## Invoice lifecycle

Generated invoices begin as `draft` and can currently be reviewed and edited. Finalized financial records should become immutable; corrections should eventually use credit notes or replacement invoices.

Payment collection is outside the MVP.

