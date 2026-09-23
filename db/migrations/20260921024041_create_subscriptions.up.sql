
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK(status IN ('active', 'inactive', 'archived')),
    CHECK(end_date IS NULL OR end_date >= start_date),
    UNIQUE (id, organization_id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (customer_id, organization_id)
        REFERENCES customers(id, organization_id) ON DELETE CASCADE
);

CREATE INDEX idx_subscriptions_organization_id ON subscriptions (organization_id);
CREATE INDEX idx_subscriptions_customer_id ON subscriptions (customer_id);

CREATE TABLE subscription_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    subscription_id UUID NOT NULL,
    price_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (id, organization_id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (subscription_id, organization_id)
        REFERENCES subscriptions(id, organization_id) ON DELETE CASCADE,
    FOREIGN KEY (price_id, organization_id)
        REFERENCES prices(id, organization_id) ON DELETE RESTRICT
);

CREATE INDEX idx_subscription_items_organization_id ON subscription_items (organization_id);
CREATE INDEX idx_subscription_items_subscription_id ON subscription_items (subscription_id);
CREATE INDEX idx_subscription_items_price_id ON subscription_items (price_id);
