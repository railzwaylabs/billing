DROP INDEX IF EXISTS idx_invoices_invoice_number;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS uq_invoices_organization_invoice_number;
ALTER TABLE invoices DROP COLUMN IF EXISTS invoice_number;
DROP TABLE IF EXISTS invoice_number_sequences;
DROP TABLE IF EXISTS invoice_number_settings;
