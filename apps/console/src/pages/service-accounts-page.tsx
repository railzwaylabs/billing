import { type FormEvent, useEffect, useState } from "react";
import { ArrowLeft, Plus } from "lucide-react";
import {
  Link,
  useNavigate,
  useOutletContext,
  useParams,
} from "react-router-dom";
import { iamApi, type Organization, type ServiceAccount } from "@/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { DataTable } from "@/components/data-table";
import { useCursorPagination } from "@/hooks/use-cursor-pagination";

export function ServiceAccountsPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { serviceAccountId } = useParams();
  const navigate = useNavigate();
  const client = iamApi(organization);
  const listPath = `/organizations/${organization.id}/developer/service-accounts`;
  const editor =
    location.pathname.endsWith("/new") || Boolean(serviceAccountId);
  const [accounts, setAccounts] = useState<ServiceAccount[]>([]);
  const [current, setCurrent] = useState<ServiceAccount | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const pagination = useCursorPagination();
  useEffect(() => {
    if (!editor)
      void client
        .serviceAccounts(pagination.request)
        .then((result) => {
          setAccounts(result.service_accounts);
          pagination.setPageInfo(result.page_info);
        })
        .catch((cause) => setError(cause.message));
  }, [organization.id, editor, pagination.cursor]);
  useEffect(() => {
    if (serviceAccountId)
      void client
        .serviceAccount(serviceAccountId)
        .then((account) => {
          setCurrent(account);
          setName(account.display_name);
          setDescription(account.description);
        })
        .catch((cause) => setError(cause.message));
  }, [serviceAccountId]);
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      if (current)
        await client.updateServiceAccount({
          id: current.id,
          display_name: name,
          description,
        });
      else
        await client.createServiceAccount({
          display_name: name,
          description,
          issuer: "billing-api-key",
          subject: crypto.randomUUID(),
        });
      navigate(listPath, { replace: true });
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Unable to save service account",
      );
    }
  }
  if (editor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild>
          <Link to={listPath}>
            <ArrowLeft />
            Service accounts
          </Link>
        </Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">DEVELOPER / SERVICE ACCOUNTS</p>
            <h1>{current ? current.display_name : "Create service account"}</h1>
            <p className="muted">
              Machine identity used for organization API access.
            </p>
          </div>
          {current && !current.disabled && (
            <Button
              variant="destructive"
              onClick={async () => {
                await client.disableServiceAccount(current.id);
                navigate(listPath, { replace: true });
              }}
            >
              Disable
            </Button>
          )}
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
                <Field>
                  <FieldLabel hint="Human-readable name for the workload identity.">
                    Display name
                  </FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    required
                    placeholder="Production ingest"
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                  />
                </Field>
                <Field>
                  <FieldLabel hint="Describe which application or environment uses this account.">
                    Description
                  </FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    placeholder="Used by the production application"
                    value={description}
                    onChange={(event) => setDescription(event.target.value)}
                  />
                </Field>
                {current && (
                  <>
                    <Field>
                      <FieldLabel hint="Trusted identity-provider issuer recorded for this account.">
                        Issuer
                      </FieldLabel>
                      <Input
                        type="url"
                        inputMode="url"
                        disabled
                        value={current.issuer}
                      />
                    </Field>
                    <Field>
                      <FieldLabel hint="Unique identity-provider subject mapped to this account.">
                        Subject
                      </FieldLabel>
                      <Input type="text" disabled value={current.subject} />
                    </Field>
                  </>
                )}
                {error && <p className="form-error">{error}</p>}
                <div className="form-actions">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => navigate(listPath)}
                  >
                    Cancel
                  </Button>
                  <Button>Save service account</Button>
                </div>
              </FieldGroup>
            </form>
          </CardContent>
        </Card>
      </main>
    );
  return (
    <main className="content">
      <div className="page-head">
        <div>
          <p className="eyebrow">DEVELOPER</p>
          <h1>Service accounts</h1>
          <p className="muted">
            Machine identities that own API credentials and IAM roles.
          </p>
        </div>
        <Button asChild>
          <Link to="new">
            <Plus />
            Create service account
          </Link>
        </Button>
      </div>
      <Card className="resource-list">
        <CardContent>
          <DataTable
            data={accounts}
            cursorPagination={{
              cursor: pagination.cursor,
              pageInfo: pagination.pageInfo,
              onCursorChange: pagination.setCursor,
            }}
            searchKey="display_name"
            searchPlaceholder="Search service accounts…"
            onRowClick={(account) => navigate(account.id)}
            columns={[
              {
                accessorKey: "display_name",
                header: "Name",
                cell: ({ row }) => (
                  <div>
                    <div className="font-medium">
                      {row.original.display_name}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {row.original.description}
                    </div>
                  </div>
                ),
              },
              { accessorKey: "subject", header: "Subject" },
              {
                id: "status",
                header: "Status",
                cell: ({ row }) =>
                  row.original.disabled ? "Disabled" : "Active",
              },
            ]}
          />
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
