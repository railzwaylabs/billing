CREATE TABLE idempotency_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    key VARCHAR(255) NOT NULL,
    request_hash BYTEA NOT NULL,
    response_status INTEGER,
    response_body JSONB,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organization_id, key),
    CONSTRAINT fk_idempotency_keys_organization
        FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT chk_idempotency_keys_key_not_blank
        CHECK (NULLIF(BTRIM(key), '') IS NOT NULL),
    CONSTRAINT chk_idempotency_keys_response_status
        CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599),
    CONSTRAINT chk_idempotency_keys_response_body
        CHECK (response_body IS NULL OR jsonb_typeof(response_body) = 'object'),
    CONSTRAINT chk_idempotency_keys_expiry
        CHECK (expires_at > created_at)
);

CREATE INDEX idx_idempotency_keys_expires_at
    ON idempotency_keys (expires_at);

CREATE TABLE usage_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    event_id VARCHAR(255) NOT NULL,
    meter_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    value DECIMAL(19, 6) NOT NULL DEFAULT 1,
    event_time TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organization_id, event_id),
    UNIQUE (id, organization_id),
    CONSTRAINT fk_usage_events_organization
        FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_usage_events_meter
        FOREIGN KEY (meter_id, organization_id)
        REFERENCES meters(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT fk_usage_events_customer
        FOREIGN KEY (customer_id, organization_id)
        REFERENCES customers(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT chk_usage_events_event_id_not_blank
        CHECK (NULLIF(BTRIM(event_id), '') IS NOT NULL),
    CONSTRAINT chk_usage_events_value_non_negative
        CHECK (value >= 0)
);

CREATE INDEX idx_usage_events_meter_time
    ON usage_events (organization_id, meter_id, event_time, id);

CREATE INDEX idx_usage_events_customer_time
    ON usage_events (organization_id, customer_id, event_time, id);

CREATE INDEX idx_usage_events_event_time
    ON usage_events (event_time);
