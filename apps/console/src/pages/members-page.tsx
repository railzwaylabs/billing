import { useEffect, useMemo, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { useOutletContext } from "react-router-dom";

import { iamApi, type DirectoryUser, type IAMPolicy, type IAMRole, type Organization } from "@/api";
import { DataTable } from "@/components/data-table";
import { RelationCombobox } from "@/components/relation-combobox";
import { RelationMultiCombobox } from "@/components/relation-multi-combobox";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";

type Member = DirectoryUser & { roles: string[] };

export function MembersPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const client = iamApi(organization);
  const [users, setUsers] = useState<DirectoryUser[]>([]);
  const [roles, setRoles] = useState<IAMRole[]>([]);
  const [policy, setPolicy] = useState<IAMPolicy | null>(null);
  const [userID, setUserID] = useState("");
  const [roleName, setRoleName] = useState("");
  const [error, setError] = useState("");

  async function load() {
    try {
      const [userResult, roleResult, policyResult] = await Promise.all([
        client.users({ limit: 100 }),
        client.roles({ limit: 100 }),
        client.policy(),
      ]);
      setUsers(userResult.users);
      setRoles(roleResult.roles);
      setPolicy(policyResult);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to load members");
    }
  }

  useEffect(() => { void load(); }, [organization.id]);

  const members = useMemo(() => {
    if (!policy) return [];
    const grouped = new Map<string, string[]>();
    for (const binding of policy.bindings) {
      if (binding.principal.type !== "user" || binding.principal.issuer !== "billing-console") continue;
      grouped.set(binding.principal.subject, [...(grouped.get(binding.principal.subject) ?? []), binding.role]);
    }
    return users.filter((user) => grouped.has(user.id)).map((user) => ({ ...user, roles: grouped.get(user.id) ?? [] }));
  }, [policy, users]);

  async function addMember() {
    if (!policy || !userID || !roleName) return;
    setError("");
    try {
      const duplicate = policy.bindings.some((binding) => binding.principal.type === "user" && binding.principal.issuer === "billing-console" && binding.principal.subject === userID && binding.role === roleName);
      const updated = duplicate ? policy : await client.setPolicy({ ...policy, bindings: [...policy.bindings, { role: roleName, principal: { type: "user", issuer: "billing-console", subject: userID } }] });
      setPolicy(updated); setUserID(""); setRoleName("");
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Unable to add member"); }
  }

  async function removeMember(member: Member) {
    if (!policy) return;
    setError("");
    try {
      const updated = await client.setPolicy({ ...policy, bindings: policy.bindings.filter((binding) => !(binding.principal.type === "user" && binding.principal.issuer === "billing-console" && binding.principal.subject === member.id)) });
      setPolicy(updated);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Unable to remove member"); }
  }

  async function updateMemberRoles(member: Member, nextRoles: string[]) {
    if (!policy) return;
    setError("");
    try {
      const otherBindings = policy.bindings.filter((binding) => !(binding.principal.type === "user" && binding.principal.issuer === "billing-console" && binding.principal.subject === member.id));
      const memberBindings = nextRoles.map((role) => ({ role, principal: { type: "user" as const, issuer: "billing-console", subject: member.id } }));
      const updated = await client.setPolicy({ ...policy, bindings: [...otherBindings, ...memberBindings] });
      setPolicy(updated);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Unable to update member roles"); }
  }

  return (
    <main className="content">
      <div className="page-head"><div><p className="eyebrow">IAM</p><h1>Members</h1><p className="muted">Manage user access to this organization.</p></div></div>
      <Card className="mb-6">
        <CardHeader><h2>Add member</h2></CardHeader>
        <CardContent>
          <FieldGroup className="grid gap-4 md:grid-cols-[1fr_1fr_auto] md:items-end">
            <Field><FieldLabel>User</FieldLabel><RelationCombobox value={userID} onValueChange={setUserID} options={users.filter((user) => !user.disabled).map((user) => ({ value: user.id, label: user.display_name || user.username, description: user.email || user.username }))} placeholder="Select user" searchPlaceholder="Search users…" /></Field>
            <Field><FieldLabel>Role</FieldLabel><RelationCombobox value={roleName} onValueChange={setRoleName} options={roles.map((role) => ({ value: role.name, label: role.display_name, description: role.name }))} placeholder="Select role" searchPlaceholder="Search roles…" /></Field>
            <Button type="button" disabled={!userID || !roleName} onClick={() => void addMember()}><Plus />Add member</Button>
          </FieldGroup>
          {error && <p className="form-error mt-4">{error}</p>}
        </CardContent>
      </Card>
      <Card className="resource-list"><CardContent><DataTable data={members} searchKey="username" searchPlaceholder="Search members…" columns={[
        { id: "member", header: "Member", cell: ({ row }) => <div><p className="font-medium">{row.original.display_name || row.original.username}</p><p className="text-sm text-muted-foreground">{row.original.email || row.original.username}</p></div> },
        { id: "roles", header: "Roles", cell: ({ row }) => <RelationMultiCombobox values={row.original.roles} onValuesChange={(values) => void updateMemberRoles(row.original, values)} options={roles.map((role) => ({ value: role.name, label: role.display_name, description: role.name }))} placeholder="Select roles" searchPlaceholder="Search roles…" /> },
        { id: "actions", header: "", cell: ({ row }) => <Button type="button" variant="ghost" size="icon" aria-label="Remove member" onClick={(event) => { event.stopPropagation(); void removeMember(row.original); }}><Trash2 /></Button> },
      ]} /></CardContent></Card>
    </main>
  );
}
