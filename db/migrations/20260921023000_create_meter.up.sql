CREATE TABLE currencies (
    code VARCHAR(3) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10) NOT NULL,
    minor_unit SMALLINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK (code ~ '^[A-Z]{3}$'),
    CHECK (minor_unit BETWEEN 0 AND 4)
);

INSERT INTO currencies (code, name, symbol, minor_unit) VALUES
    ('USD', 'US Dollar', '$', 2),
    ('IDR', 'Indonesian Rupiah', 'Rp', 0),
    ('EUR', 'Euro', '€', 2),
    ('SGD', 'Singapore Dollar', 'S$', 2);

CREATE TABLE measurement_units (
    code VARCHAR(100) PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    category VARCHAR(50) NOT NULL,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK (NULLIF(BTRIM(code), '') IS NOT NULL),
    CHECK (NULLIF(BTRIM(category), '') IS NOT NULL)
);

INSERT INTO measurement_units (code, name, symbol, category, description) VALUES
    ('request', 'Request', 'request', 'count', 'Number of requests or operations.'),
    ('token', 'Token', 'token', 'count', 'Number of processed tokens.'),
    ('sample', 'Metric sample', 'sample', 'count', 'Number of metric samples.'),
    ('seat', 'Seat', 'seat', 'count', 'Number of licensed seats.'),
    ('second', 'Second', 's', 'time', 'Elapsed seconds.'),
    ('minute', 'Minute', 'min', 'time', 'Elapsed minutes.'),
    ('byte', 'Byte', 'B', 'data', 'Bytes transferred or stored.'),
    ('gib', 'Gibibyte', 'GiB', 'data', 'Binary gibibytes transferred or stored.'),
    ('vcpu_second', 'vCPU second', 'vCPU-second', 'compute', 'One virtual CPU used for one second.'),
    ('gib_second', 'GiB second', 'GiB-second', 'compute', 'One GiB of memory used for one second.'),
    ('gib_hour', 'GiB hour', 'GiB-hour', 'capacity', 'One GiB provisioned for one hour.');

CREATE TABLE meters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    code VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    aggregation VARCHAR(50) NOT NULL,
    unit VARCHAR(100) NOT NULL REFERENCES measurement_units(code) ON DELETE RESTRICT,
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
