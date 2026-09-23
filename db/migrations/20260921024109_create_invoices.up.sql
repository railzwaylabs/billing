
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    billing_period_start TIMESTAMPTZ NOT NULL,
    billing_period_end TIMESTAMPTZ NOT NULL,
    issued_at TIMESTAMPTZ,

    subtotal DECIMAL(29, 9) NOT NULL DEFAULT 0,
    tax DECIMAL(29, 9) NOT NULL DEFAULT 0,
    total DECIMAL(29, 9) NOT NULL DEFAULT 0,

    currency VARCHAR(3) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (status IN ('draft', 'open', 'past_due', 'paid')),
    CHECK (subtotal >= 0),
    CHECK (tax >= 0),
    CHECK (total = subtotal + tax),
    CHECK (currency ~ '^[A-Z]{3}$'),
    CHECK (billing_period_end > billing_period_start),
    UNIQUE (organization_id, customer_id, billing_period_start, billing_period_end),
    UNIQUE (id, organization_id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
	FOREIGN KEY (currency) REFERENCES currencies(code) ON DELETE RESTRICT,
    FOREIGN KEY (customer_id, organization_id)
        REFERENCES customers(id, organization_id) ON DELETE CASCADE
);

CREATE INDEX idx_invoices_organization_id ON invoices (organization_id);
CREATE INDEX idx_invoices_customer_id ON invoices (customer_id);
CREATE INDEX idx_invoices_status ON invoices (status);

CREATE TABLE invoice_lines (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL,
  invoice_id UUID NOT NULL,
  subscription_id UUID,
  subscription_item_id UUID,
  product_id UUID NOT NULL,
  price_id UUID NOT NULL,
  price_charge_id UUID NOT NULL,
  meter_id UUID NOT NULL,
  description TEXT NOT NULL,
  usage_quantity DECIMAL(19, 6) NOT NULL,
  unit VARCHAR(100) NOT NULL,
  pricing_unit_quantity DECIMAL(19, 6) NOT NULL,
  unit_amount DECIMAL(29, 9) NOT NULL,
  amount DECIMAL(29, 9) NOT NULL,
  pricing_details JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (id, organization_id),
  CHECK(usage_quantity >= 0),
  CHECK(pricing_unit_quantity > 0),
  CHECK(unit_amount >= 0),
  CHECK(amount >= 0),
  CHECK(NULLIF(BTRIM(unit), '') IS NOT NULL),
  CHECK(jsonb_typeof(pricing_details) = 'object'),
  CHECK(subscription_id IS NOT NULL OR subscription_item_id IS NOT NULL),
  FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  FOREIGN KEY (invoice_id, organization_id)
      REFERENCES invoices(id, organization_id) ON DELETE CASCADE,
  FOREIGN KEY (subscription_id, organization_id)
      REFERENCES subscriptions(id, organization_id) ON DELETE RESTRICT,
  FOREIGN KEY (subscription_item_id, organization_id)
      REFERENCES subscription_items(id, organization_id) ON DELETE RESTRICT,
  FOREIGN KEY (product_id, organization_id)
      REFERENCES products(id, organization_id) ON DELETE RESTRICT,
  FOREIGN KEY (price_id, organization_id)
      REFERENCES prices(id, organization_id) ON DELETE RESTRICT,
  FOREIGN KEY (price_charge_id, organization_id)
      REFERENCES price_charges(id, organization_id) ON DELETE RESTRICT,
  FOREIGN KEY (meter_id, organization_id)
      REFERENCES meters(id, organization_id) ON DELETE RESTRICT
);

CREATE INDEX idx_invoice_lines_organization_id ON invoice_lines (organization_id);
CREATE INDEX idx_invoice_lines_invoice_id ON invoice_lines (invoice_id);
CREATE INDEX idx_invoice_lines_subscription_id ON invoice_lines (subscription_id);
CREATE INDEX idx_invoice_lines_subscription_item_id ON invoice_lines (subscription_item_id);
CREATE INDEX idx_invoice_lines_product_id ON invoice_lines (product_id);
CREATE INDEX idx_invoice_lines_price_id ON invoice_lines (price_id);
CREATE INDEX idx_invoice_lines_price_charge_id ON invoice_lines (price_charge_id);
CREATE INDEX idx_invoice_lines_meter_id ON invoice_lines (meter_id);
