CREATE TABLE invoice_number_settings (
    organization_id UUID PRIMARY KEY,
    number_format VARCHAR(100) NOT NULL DEFAULT 'INV-{YYYY}-{SEQ:06}',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (NULLIF(BTRIM(number_format), '') IS NOT NULL),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE TABLE invoice_number_sequences (
    organization_id UUID NOT NULL,
    sequence_year INTEGER NOT NULL,
    last_value BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (organization_id, sequence_year),
    CHECK (sequence_year BETWEEN 2000 AND 9999),
    CHECK (last_value >= 0),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

ALTER TABLE invoices ADD COLUMN invoice_number VARCHAR(100);

WITH numbered AS (
    SELECT id,
           'INV-' || EXTRACT(YEAR FROM created_at)::INTEGER || '-LEGACY-' ||
           LPAD(ROW_NUMBER() OVER (PARTITION BY organization_id, EXTRACT(YEAR FROM created_at) ORDER BY created_at, id)::TEXT, 6, '0') AS generated_number
    FROM invoices
)
UPDATE invoices
SET invoice_number = numbered.generated_number
FROM numbered
WHERE invoices.id = numbered.id;

ALTER TABLE invoices ALTER COLUMN invoice_number SET NOT NULL;
ALTER TABLE invoices ADD CONSTRAINT uq_invoices_organization_invoice_number UNIQUE (organization_id, invoice_number);
CREATE INDEX idx_invoices_invoice_number ON invoices (invoice_number);

INSERT INTO invoice_number_sequences (organization_id, sequence_year, last_value, updated_at)
SELECT organization_id, EXTRACT(YEAR FROM created_at)::INTEGER, COUNT(*), NOW()
FROM invoices
GROUP BY organization_id, EXTRACT(YEAR FROM created_at)
ON CONFLICT (organization_id, sequence_year)
DO UPDATE SET last_value = GREATEST(invoice_number_sequences.last_value, EXCLUDED.last_value), updated_at = EXCLUDED.updated_at;
