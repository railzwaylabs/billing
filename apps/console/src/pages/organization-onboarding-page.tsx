import { type FormEvent, useState } from "react";
import { api, type Organization } from "@/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";

export function OrganizationOnboardingPage({
  onCreated,
}: {
  onCreated: (organization: Organization) => void;
}) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugEdited, setSlugEdited] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const result = await api.createOrganization({ name, slug });
      onCreated(result.organization);
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Unable to create organization",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="content onboarding-page">
      <div className="page-head">
        <div>
          <p className="eyebrow">GET STARTED</p>
          <h1>Create your organization</h1>
          <p className="muted">
            This workspace contains your catalog, customers, usage, and
            invoices.
          </p>
        </div>
      </div>
      <Card className="onboarding-card">
        <CardHeader>
          <h2>Organization details</h2>
          <p className="muted">
            The slug is permanent and becomes part of IAM resource names.
          </p>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit}>
            <FieldGroup>
            <Field>
              <FieldLabel>Organization name</FieldLabel>
              <Input
                type="text"
                inputMode="text"
                autoComplete="organization"
                autoFocus
                required
                value={name}
                placeholder="Acme, Inc."
                onChange={(event) => {
                  const nextName = event.target.value;
                  setName(nextName);
                  if (!slugEdited) {
                    setSlug(
                      nextName
                        .toLowerCase()
                        .replace(/[^a-z0-9]+/g, "-")
                        .replace(/^-|-$/g, ""),
                    );
                  }
                }}
              />
            </Field>
            <Field>
              <FieldLabel>Organization slug</FieldLabel>
              <Input
                type="text"
                inputMode="text"
                pattern="[a-z0-9]+(?:-[a-z0-9]+)*"
                spellCheck={false}
                required
                value={slug}
                placeholder="acme"
                onChange={(event) => {
                  setSlugEdited(true);
                  setSlug(event.target.value);
                }}
              />
            </Field>
            {error && <p className="form-error">{error}</p>}
            <div className="onboarding-actions">
              <Button disabled={submitting}>
                {submitting ? "Creating…" : "Create organization"}
              </Button>
            </div>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
