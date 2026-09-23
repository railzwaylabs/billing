import { type FormEvent, useEffect, useState } from "react";
import { ArrowLeft, Plus } from "lucide-react";
import {
  Link,
  useNavigate,
  useOutletContext,
  useParams,
} from "react-router-dom";
import { organizationApi, type Customer, type Organization } from "@/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { DataTable } from "@/components/data-table";

export function CustomersPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { customerId } = useParams();
  const navigate = useNavigate();
  const client = organizationApi(organization.id);
  const isEditor = location.pathname.endsWith("/new") || Boolean(customerId);
  const [values, setValues] = useState<Customer[]>([]);
  const [current, setCurrent] = useState<Customer | null>(null);
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [error, setError] = useState("");
  useEffect(() => {
    if (!isEditor)
      void client
        .customers()
        .then((result) => setValues(result.customers))
        .catch((cause) => setError(cause.message));
  }, [organization.id, isEditor]);
  useEffect(() => {
    if (customerId)
      void client
        .customer(customerId)
        .then(({ customer }) => {
          setCurrent(customer);
          setFirstName(customer.FirstName);
          setLastName(customer.LastName);
        })
        .catch((cause) => setError(cause.message));
  }, [customerId]);
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      const body = { first_name: firstName, last_name: lastName };
      if (current) await client.updateCustomer(current.ID, body);
      else await client.createCustomer(body);
      navigate("..");
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to save customer",
      );
    }
  }
  if (isEditor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild><Link to=".."><ArrowLeft />Customers</Link></Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">CUSTOMERS</p>
            <h1>
              {current
                ? `${current.FirstName} ${current.LastName}`
                : "Create customer"}
            </h1>
            <p className="muted">
              Customer details used by subscriptions, usage, and invoices.
            </p>
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
              <Field>
                <FieldLabel>First name</FieldLabel>
                <Input
                  type="text"
                  inputMode="text"
                  autoComplete="given-name"
                  required
                  value={firstName}
                  onChange={(event) => setFirstName(event.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel>Last name</FieldLabel>
                <Input
                  type="text"
                  inputMode="text"
                  autoComplete="family-name"
                  required
                  value={lastName}
                  onChange={(event) => setLastName(event.target.value)}
                />
              </Field>
              {error && <p className="form-error">{error}</p>}
              <div className="form-actions">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => navigate("..")}
                >
                  Cancel
                </Button>
                <Button>Save customer</Button>
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
          <p className="eyebrow">CUSTOMERS</p>
          <h1>Customers</h1>
          <p className="muted">
            Accounts with subscriptions, usage, and invoice history.
          </p>
        </div>
        <Button asChild>
          <Link to="new">
            <Plus />
            Create customer
          </Link>
        </Button>
      </div>
      <Card className="resource-list">
        <CardContent>
          <DataTable
            data={values}
            searchKey="FirstName"
            searchPlaceholder="Search customers…"
            onRowClick={(customer) => navigate(customer.ID)}
            columns={[
              {
                accessorKey: "FirstName",
                header: "Name",
                cell: ({ row }) => (
                  <span className="font-medium">
                    {row.original.FirstName} {row.original.LastName}
                  </span>
                ),
              },
              {
                accessorKey: "CreatedAt",
                header: "Created",
                cell: ({ row }) =>
                  new Date(row.original.CreatedAt).toLocaleDateString(),
              },
            ]}
          />
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
