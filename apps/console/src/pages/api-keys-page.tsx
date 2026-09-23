import { type FormEvent, useEffect, useState } from "react";
import { ArrowLeft, Copy, Plus } from "lucide-react";
import { Link, useOutletContext } from "react-router-dom";
import {
  iamApi,
  type APIKey,
  type Organization,
  type ServiceAccount,
} from "@/api";
import { RelationCombobox } from "@/components/relation-combobox";
import { DatePicker } from "@/components/date-picker";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { DataTable } from "@/components/data-table";
import { useCursorPagination } from "@/hooks/use-cursor-pagination";

export function APIKeysPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const client = iamApi(organization);
  const listPath = `/organizations/${organization.id}/developer/api-keys`;
  const create = location.pathname.endsWith("/new");
  const [accounts, setAccounts] = useState<ServiceAccount[]>([]);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [accountID, setAccountID] = useState("");
  const [name, setName] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [secret, setSecret] = useState("");
  const [error, setError] = useState("");
  const pagination = useCursorPagination();
  useEffect(() => {
    void client
      .serviceAccounts({ limit: 100 })
      .then((result) => setAccounts(result.service_accounts))
      .catch((cause) => setError(cause.message));
  }, [organization.id]);
  useEffect(() => {
    if (accountID)
      void client
        .apiKeys(accountID, pagination.request)
        .then((result) => {
          setKeys(result.api_keys);
          pagination.setPageInfo(result.page_info);
        })
        .catch((cause) => setError(cause.message));
    else setKeys([]);
  }, [accountID, pagination.cursor]);
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      const key = await client.createAPIKey(accountID, {
        display_name: name,
        expires_at: expiresAt
          ? new Date(`${expiresAt}T23:59:59Z`).toISOString()
          : undefined,
      });
      setSecret(key.key ?? "");
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to create API key",
      );
    }
  }
  const options = accounts
    .filter((account) => !account.disabled)
    .map((account) => ({
      value: account.id,
      label: account.display_name,
      description: account.description,
    }));
  const selectAccount = (value: string) => {
    pagination.reset();
    setAccountID(value);
  };
  if (create)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild>
          <Link to={listPath}>
            <ArrowLeft />
            API keys
          </Link>
        </Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">DEVELOPER / API KEYS</p>
            <h1>Create API key</h1>
            <p className="muted">
              The secret is displayed exactly once after creation.
            </p>
          </div>
        </div>
        {secret ? (
          <Card className="secret-card">
            <CardContent className="pt-6">
              <p className="eyebrow">NEW SECRET</p>
              <code>{secret}</code>
              <div className="form-actions">
                <Button
                  variant="outline"
                  onClick={() => void navigator.clipboard.writeText(secret)}
                >
                  <Copy size={16} />
                  Copy secret
                </Button>
                <Button asChild>
                  <Link to={listPath}>Done</Link>
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="pt-6">
              <form onSubmit={submit}>
                <FieldGroup>
                  <Field>
                    <FieldLabel hint="Service account that will own and authenticate with this key.">
                      Service account
                    </FieldLabel>
                    <RelationCombobox
                      value={accountID}
                      onValueChange={selectAccount}
                      options={options}
                      placeholder="Select service account"
                      searchPlaceholder="Search service accounts…"
                    />
                  </Field>
                  <Field>
                    <FieldLabel hint="Descriptive name used to identify this key later; the secret is shown only once.">
                      Key name
                    </FieldLabel>
                    <Input
                      type="text"
                      inputMode="text"
                      required
                      placeholder="Production usage ingest"
                      value={name}
                      onChange={(event) => setName(event.target.value)}
                    />
                  </Field>
                  <Field>
                    <FieldLabel hint="After this date the API key is rejected automatically. Leave empty for no expiry.">
                      Expires on (optional)
                    </FieldLabel>
                    <DatePicker
                      value={expiresAt}
                      onValueChange={setExpiresAt}
                      placeholder="No expiration date"
                    />
                  </Field>
                  {error && <p className="form-error">{error}</p>}
                  <div className="form-actions">
                    <Button variant="outline" asChild>
                      <Link to={listPath}>Cancel</Link>
                    </Button>
                    <Button disabled={!accountID}>Create API key</Button>
                  </div>
                </FieldGroup>
              </form>
            </CardContent>
          </Card>
        )}
      </main>
    );
  return (
    <main className="content">
      <div className="page-head">
        <div>
          <p className="eyebrow">DEVELOPER</p>
          <h1>API keys</h1>
          <p className="muted">
            Organization credentials grouped by service account.
          </p>
        </div>
        <Button asChild>
          <Link to="new">
            <Plus />
            Create API key
          </Link>
        </Button>
      </div>
      <div className="list-toolbar">
        <RelationCombobox
          value={accountID}
          onValueChange={selectAccount}
          options={options}
          placeholder="Select service account"
          searchPlaceholder="Search service accounts…"
        />
      </div>
      <Card className="resource-list">
        <CardContent>
          {!accountID ? (
            <div className="empty-state">
              <h2>Select a service account</h2>
              <p className="muted">
                Choose a service account to inspect its keys.
              </p>
            </div>
          ) : (
            <DataTable
              data={keys}
              cursorPagination={{
                cursor: pagination.cursor,
                pageInfo: pagination.pageInfo,
                onCursorChange: pagination.setCursor,
                resetKey: accountID,
              }}
              searchKey="display_name"
              searchPlaceholder="Search API keys…"
              columns={[
                {
                  accessorKey: "display_name",
                  header: "Name",
                  cell: ({ row }) => (
                    <span className="font-medium">
                      {row.original.display_name}
                    </span>
                  ),
                },
                { accessorKey: "key_id", header: "Key ID" },
                {
                  accessorKey: "created_at",
                  header: "Created",
                  cell: ({ row }) =>
                    new Date(row.original.created_at).toLocaleDateString(),
                },
                {
                  accessorKey: "expires_at",
                  header: "Expires",
                  cell: ({ row }) =>
                    row.original.expires_at
                      ? new Date(row.original.expires_at).toLocaleDateString()
                      : "Never",
                },
                {
                  id: "status",
                  header: "Status",
                  cell: ({ row }) =>
                    row.original.revoked_at ? "Revoked" : "Active",
                },
                {
                  id: "actions",
                  cell: ({ row }) =>
                    !row.original.revoked_at ? (
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={async () => {
                          await client.revokeAPIKey(row.original.id);
                          const result = await client.apiKeys(
                            accountID,
                            pagination.request,
                          );
                          setKeys(result.api_keys);
                          pagination.setPageInfo(result.page_info);
                        }}
                      >
                        Revoke
                      </Button>
                    ) : null,
                },
              ]}
            />
          )}
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
