
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    meter_id UUID NOT NULL,
    code VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK(status IN ('active', 'inactive', 'archived')),
    UNIQUE (organization_id, code),
    UNIQUE (id, organization_id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (meter_id, organization_id)
        REFERENCES meters(id, organization_id) ON DELETE RESTRICT
);

CREATE INDEX idx_products_organization_id ON products (organization_id);
CREATE INDEX idx_products_meter_id ON products (meter_id);

CREATE TABLE prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    product_id UUID NOT NULL,
    currency VARCHAR(3) NOT NULL,
    unit_quantity DECIMAL(19, 6) NOT NULL DEFAULT 1,
    aggregation_interval VARCHAR(20) NOT NULL DEFAULT 'month',
    interval_type VARCHAR(50),
    interval_count INTEGER,
    effective_at TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK(currency ~ '^[A-Z]{3}$'),
    CHECK(unit_quantity > 0),
    CHECK(aggregation_interval IN ('day', 'month')),
    CHECK(interval_type IN ('month', 'year', 'day', 'week')),
    CHECK(
        (interval_type IS NULL AND interval_count IS NULL)
        OR (interval_type IS NOT NULL AND interval_count > 0)
    ),
    CHECK(effective_until IS NULL OR effective_until > effective_at),
    CHECK(status IN ('active', 'inactive', 'archived')),
    UNIQUE (id, organization_id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id, organization_id)
        REFERENCES products(id, organization_id) ON DELETE CASCADE
);

CREATE INDEX idx_prices_organization_id ON prices (organization_id);
CREATE INDEX idx_prices_product_id ON prices (product_id);
CREATE INDEX idx_prices_currency ON prices (currency);
CREATE INDEX idx_prices_effective_at ON prices (effective_at);
CREATE INDEX idx_prices_effective_until ON prices (effective_until);

CREATE TABLE price_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    price_id UUID NOT NULL,
    start_quantity DECIMAL(19, 6) NOT NULL,
    unit_amount DECIMAL(29, 9) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (price_id, start_quantity),
    UNIQUE (id, organization_id),
    CHECK(start_quantity >= 0),
    CHECK(unit_amount >= 0),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (price_id, organization_id)
        REFERENCES prices(id, organization_id) ON DELETE CASCADE
);

CREATE INDEX idx_price_tiers_organization_id
    ON price_tiers (organization_id);
