import { useEffect, useState } from "react";
import { Trash2 } from "lucide-react";
import {
  iamApi,
  type IAMPolicy,
  type IAMRole,
  type Organization,
  type ServiceAccount,
} from "@/api";
import { RelationCombobox } from "@/components/relation-combobox";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { DataTable } from "@/components/data-table";
export function PolicyEditor({
  organization,
  roles,
}: {
  organization: Organization;
  roles: IAMRole[];
}) {
  const client = iamApi(organization);
  const [policy, setPolicy] = useState<IAMPolicy | null>(null);
  const [accounts, setAccounts] = useState<ServiceAccount[]>([]);
  const [accountID, setAccountID] = useState("");
  const [roleName, setRoleName] = useState("");
  const [error, setError] = useState("");
  const load = async () => {
    const [p, a] = await Promise.all([
      client.policy(),
      client.serviceAccounts(),
    ]);
    setPolicy(p);
    setAccounts(a.service_accounts);
  };
  useEffect(() => {
    void load().catch((e) => setError(e.message));
  }, [organization.id]);
  async function save(bindings: IAMPolicy["bindings"]) {
    if (!policy) return;
    try {
      setPolicy(await client.setPolicy({ ...policy, bindings }));
    } catch (c) {
      setError(c instanceof Error ? c.message : "Unable to update policy");
      await load();
    }
  }
  function add() {
    const account = accounts.find((a) => a.id === accountID);
    if (!account || !roleName || !policy) return;
    const duplicate = policy.bindings.some(
      (b) =>
        b.role === roleName &&
        b.principal.issuer === account.issuer &&
        b.principal.subject === account.subject,
    );
    if (!duplicate)
      void save([
        ...policy.bindings,
        {
          role: roleName,
          principal: {
            type: "service_account",
            issuer: account.issuer,
            subject: account.subject,
          },
        },
      ]);
    setAccountID("");
    setRoleName("");
  }
  return (
    <Card className="resource-form">
      <CardHeader>
        <h2>Organization policy</h2>
        <p className="muted">
          Bind service accounts to roles at the organization resource.
        </p>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="inline-resource-form">
          <RelationCombobox
            value={accountID}
            onValueChange={setAccountID}
            options={accounts
              .filter((a) => !a.disabled)
              .map((a) => ({
                value: a.id,
                label: a.display_name,
                description: a.subject,
              }))}
            placeholder="Select service account"
          />
          <RelationCombobox
            value={roleName}
            onValueChange={setRoleName}
            options={roles.map((r) => ({
              value: r.name,
              label: r.display_name,
              description: r.name,
            }))}
            placeholder="Select role"
          />
          <Button
            type="button"
            disabled={!accountID || !roleName}
            onClick={add}
          >
            Add binding
          </Button>
        </div>
        {error && <p className="form-error">{error}</p>}
        <DataTable
          data={policy?.bindings ?? []}
          searchKey="role"
          searchPlaceholder="Search bindings…"
          columns={[
            {
              id: "principal",
              header: "Principal",
              cell: ({ row }) =>
                accounts.find(
                  (account) =>
                    account.issuer === row.original.principal.issuer &&
                    account.subject === row.original.principal.subject,
                )?.display_name ?? row.original.principal.subject,
            },
            { accessorKey: "principal.type", header: "Type" },
            {
              accessorKey: "role",
              header: "Role",
              cell: ({ row }) =>
                roles.find((role) => role.name === row.original.role)
                  ?.display_name ?? row.original.role,
            },
            {
              id: "actions",
              cell: ({ row }) => (
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  aria-label="Remove binding"
                  onClick={() =>
                    void save(
                      (policy?.bindings ?? []).filter(
                        (binding) => binding !== row.original,
                      ),
                    )
                  }
                >
                  <Trash2 />
                </Button>
              ),
            },
          ]}
        />
      </CardContent>
    </Card>
  );
}
