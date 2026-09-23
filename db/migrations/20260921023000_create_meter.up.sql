CREATE TABLE meters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    code VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    aggregation VARCHAR(50) NOT NULL,
    unit VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (organization_id, code),
    UNIQUE (id, organization_id),
    CONSTRAINT fk_meters_organization
        FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT chk_meters_aggregation
        CHECK (aggregation IN ('count', 'sum')),
    CONSTRAINT chk_meters_unit_not_blank
        CHECK (NULLIF(BTRIM(unit), '') IS NOT NULL)
);

CREATE INDEX idx_meters_organization_id ON meters (organization_id);
CREATE INDEX idx_meters_organization_created_at
    ON meters (organization_id, created_at, id);
