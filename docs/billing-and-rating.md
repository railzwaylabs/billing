# Billing and rating

## Metering

A meter describes what an event value measures:

- `code`: stable identifier such as `api_calls`.
- `unit`: a code selected from the `measurement_units` reference table, such as
  `request`, `token`, or `gib_hour`.
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

## Catalog and prices

A product is the catalog root. A versioned price belongs to one product and
contains one or more price charges. Each charge selects its own meter, pricing
model, pricing unit, and tiers. For example, one Compute Engine price can carry
separate vCPU, memory, and persistent-disk charges.

Currency is selected from the `currencies` reference table. It is not accepted
as an arbitrary console value. The MVP supports `per_unit` and `graduated`:

- `per_unit` requires exactly one tier starting at zero.
- `graduated` requires one or more strictly increasing tiers starting at zero.

| Start quantity | Unit amount |
| ---: | ---: |
| 0 | USD 0.50 |
| 1,000 | USD 0.35 |
| 10,000 | USD 0.20 |

Graduated usage is charged progressively across tiers. Rating creates one
invoice line for every charge with billable usage and persists the charge,
meter, pricing unit, effective period, and tier breakdown used by the
calculation.

## Subscriptions

A customer has a subscription containing effective-dated items. Each item
points to a price and has its own `[start_at, end_at)` period. Enabling Cloud
Storage halfway through a cycle adds an item to the existing subscription; it
does not create a second subscription. An item is stopped by setting `end_at`,
not by deleting its history. All items in this MVP subscription use the same
currency.

## Rating period

The scheduler currently rates the previous UTC calendar month as a half-open period:

```text
[first day of previous month 00:00 UTC, first day of current month 00:00 UTC)
```

An event at the exact period end belongs to the next period. Subscription end dates entered at midnight are treated inclusively through that calendar day. Price and subscription-item effective periods are intersected with the billing period before usage is queried.

## Invoice lifecycle

Generated invoices begin as `draft` and can currently be reviewed and edited. Finalized financial records should become immutable; corrections should eventually use credit notes or replacement invoices.

Payment collection is outside the MVP.
