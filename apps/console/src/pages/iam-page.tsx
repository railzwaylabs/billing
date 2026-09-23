import { type FormEvent, useEffect, useMemo, useState } from "react";
import { ArrowLeft, Plus } from "lucide-react";
import {
  Link,
  useNavigate,
  useOutletContext,
  useParams,
} from "react-router-dom";
import { iamApi, type IAMRole, type Organization } from "@/api";
import { DataTable } from "@/components/data-table";
import { PolicyEditor } from "@/components/policy-editor";
import { RelationMultiCombobox } from "@/components/relation-multi-combobox";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export function IAMPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { roleId } = useParams();
  const navigate = useNavigate();
  const client = iamApi(organization);
  const policyPage = location.pathname.endsWith("/policy");
  const editor = location.pathname.endsWith("/new") || Boolean(roleId);
  const [roles, setRoles] = useState<IAMRole[]>([]);
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [permissions, setPermissions] = useState<string[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    void client
      .roles()
      .then((result) => setRoles(result.roles))
      .catch((cause) =>
        setError(
          cause instanceof Error ? cause.message : "Unable to load roles",
        ),
      );
  }, [organization.id]);

  const current = roles.find((role) => role.id === roleId);
  useEffect(() => {
    if (!current) return;
    setName(current.name);
    setDisplayName(current.display_name);
    setDescription(current.description);
    setPermissions(current.permissions);
  }, [current?.id]);

  const permissionOptions = useMemo(
    () =>
      [...new Set(roles.flatMap((role) => role.permissions))]
        .sort()
        .map((permission) => ({ value: permission, label: permission })),
    [roles],
  );

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    try {
      if (current) {
        await client.updateRole({
          id: current.id,
          name,
          display_name: displayName,
          description,
          permissions,
          etag: current.etag,
        });
      } else {
        await client.createRole({
          name,
          display_name: displayName,
          description,
          permissions,
        });
      }
      navigate("../roles");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to save role");
    }
  }

  if (policyPage)
    return (
      <main className="content">
        <div className="page-head">
          <div>
            <p className="eyebrow">IAM</p>
            <h1>Policy</h1>
            <p className="muted">
              Assign roles to service accounts for this organization.
            </p>
          </div>
        </div>
        <PolicyEditor organization={organization} roles={roles} />
      </main>
    );

  if (editor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild>
          <Link to="../roles">
            <ArrowLeft />
            Roles
          </Link>
        </Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">IAM / ROLES</p>
            <h1>{current ? current.display_name : "Create role"}</h1>
            <p className="muted">
              Group permissions into a reusable organization role.
            </p>
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="role-name">Role name</FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    spellCheck={false}
                    id="role-name"
                    required
                    disabled={current?.predefined}
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    placeholder="billingOperator"
                  />
                  <FieldDescription>
                    A stable identifier for this role.
                  </FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor="role-display-name">
                    Display name
                  </FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    id="role-display-name"
                    required
                    disabled={current?.predefined}
                    value={displayName}
                    onChange={(event) => setDisplayName(event.target.value)}
                    placeholder="Billing operator"
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="role-description">
                    Description
                  </FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    id="role-description"
                    disabled={current?.predefined}
                    value={description}
                    onChange={(event) => setDescription(event.target.value)}
                  />
                </Field>
                <Field>
                  <FieldLabel>Permissions</FieldLabel>
                  <RelationMultiCombobox
                    values={permissions}
                    options={permissionOptions}
                    onValuesChange={setPermissions}
                    placeholder="Select permissions"
                    searchPlaceholder="Search permissions…"
                  />
                </Field>
                {error && <p className="form-error">{error}</p>}
                <div className="form-actions">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => navigate("../roles")}
                  >
                    Cancel
                  </Button>
                  {!current?.predefined && (
                    <Button type="submit">Save role</Button>
                  )}
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
          <p className="eyebrow">IAM</p>
          <h1>Roles</h1>
          <p className="muted">
            Manage predefined and organization-specific permission sets.
          </p>
        </div>
        <Button asChild>
          <Link to="new">
            <Plus />
            Create role
          </Link>
        </Button>
      </div>
      <Card>
        <CardContent>
          <DataTable
            data={roles}
            searchKey="display_name"
            searchPlaceholder="Search roles…"
            onRowClick={(role) => navigate(role.id)}
            columns={[
              {
                accessorKey: "display_name",
                header: "Role",
                cell: ({ row }) => (
                  <div>
                    <div className="font-medium">
                      {row.original.display_name}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {row.original.name}
                    </div>
                  </div>
                ),
              },
              {
                id: "type",
                header: "Type",
                cell: ({ row }) =>
                  row.original.predefined ? "Predefined" : "Custom",
              },
              {
                id: "permissions",
                header: "Permissions",
                cell: ({ row }) => row.original.permissions.length,
              },
              { accessorKey: "description", header: "Description" },
            ]}
          />
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
