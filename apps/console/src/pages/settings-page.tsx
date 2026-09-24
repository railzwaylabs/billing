import { type FormEvent, useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { api, organizationApi, type Organization } from "@/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
export function SettingsPage() {
  const { organization, onOrganizationUpdated } = useOutletContext<{
    organization: Organization;
    onOrganizationUpdated: (organization: Organization) => void;
  }>();
  const [name, setName] = useState(organization.name);
  const [invoiceNumberFormat, setInvoiceNumberFormat] = useState(
    "INV-{YYYY}-{SEQ:06}",
  );
  const [organizationError, setOrganizationError] = useState("");
  const [organizationSaved, setOrganizationSaved] = useState(false);
  const [invoiceError, setInvoiceError] = useState("");
  const [invoiceSaved, setInvoiceSaved] = useState(false);
  useEffect(() => {
    void organizationApi(organization.id)
      .invoiceNumberSettings()
      .then(({ settings }) => setInvoiceNumberFormat(settings.number_format))
      .catch((cause) => setInvoiceError(cause.message));
  }, [organization.id]);
  async function submitOrganization(e: FormEvent) {
    e.preventDefault();
    try {
      setOrganizationError("");
      const result = await api.updateOrganization(organization.id, { name });
      onOrganizationUpdated(result.organization);
      setOrganizationSaved(true);
    } catch (c) {
      setOrganizationError(
        c instanceof Error ? c.message : "Unable to update organization",
      );
    }
  }
  async function submitInvoiceNumbering(e: FormEvent) {
    e.preventDefault();
    try {
      setInvoiceError("");
      await organizationApi(organization.id).updateInvoiceNumberSettings(
        invoiceNumberFormat,
      );
      setInvoiceSaved(true);
    } catch (c) {
      setInvoiceError(
        c instanceof Error ? c.message : "Unable to update invoice numbering",
      );
    }
  }
  return (
    <main className="content">
      <div className="page-head">
        <div>
          <p className="eyebrow">SETTINGS</p>
          <h1>Workspace settings</h1>
          <p className="muted">Manage the active organization.</p>
        </div>
      </div>
      <Card className="onboarding-card">
        <CardHeader>
          <h2>Organization</h2>
        </CardHeader>
        <CardContent>
          <form onSubmit={submitOrganization}>
            <FieldGroup>
              <Field>
                <FieldLabel hint="Organization name displayed in the console and invoices.">
                  Name
                </FieldLabel>
                <Input
                  type="text"
                  inputMode="text"
                  autoComplete="organization"
                  required
                  placeholder="Acme, Inc."
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value);
                    setOrganizationSaved(false);
                  }}
                />
              </Field>
              <Field>
                <FieldLabel hint="Immutable identifier used by canonical IAM resource names.">
                  Slug
                </FieldLabel>
                <Input type="text" value={organization.slug} disabled />
                <FieldDescription>
                  The slug is immutable because it is part of canonical IAM
                  resource names.
                </FieldDescription>
              </Field>
              {organizationError && (
                <p className="form-error">{organizationError}</p>
              )}
              {organizationSaved && (
                <p className="success-message">Organization updated.</p>
              )}
              <div className="onboarding-actions">
                <Button>Save changes</Button>
              </div>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
      <Card className="onboarding-card">
        <CardHeader>
          <h2>Invoice numbering</h2>
        </CardHeader>
        <CardContent>
          <form onSubmit={submitInvoiceNumbering}>
            <FieldGroup>
              <Field>
                <FieldLabel hint="Template used when allocating the next invoice number for this organization.">
                  Invoice number format
                </FieldLabel>
                <Input
                  type="text"
                  inputMode="text"
                  required
                  spellCheck={false}
                  placeholder="INV-{YYYY}-{SEQ:06}"
                  value={invoiceNumberFormat}
                  onChange={(event) => {
                    setInvoiceNumberFormat(event.target.value);
                    setInvoiceSaved(false);
                  }}
                />
                <FieldDescription>
                  Tokens: {`{YYYY}`}, {`{YY}`}, {`{MM}`}, and one required{" "}
                  {`{SEQ:n}`}. Example: INV-{`{YYYY}`}-{`{SEQ:06}`}.
                </FieldDescription>
              </Field>
              {invoiceError && <p className="form-error">{invoiceError}</p>}
              {invoiceSaved && (
                <p className="success-message">Invoice numbering updated.</p>
              )}
              <div className="onboarding-actions">
                <Button>Save invoice numbering</Button>
              </div>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
