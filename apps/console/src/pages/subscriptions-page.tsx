import { type FormEvent, useEffect, useState } from "react";
import { ArrowLeft, Plus } from "lucide-react";
import {
  Link,
  useNavigate,
  useOutletContext,
  useParams,
} from "react-router-dom";
import {
  organizationApi,
  type Customer,
  type Organization,
  type Price,
  type Product,
  type Subscription,
} from "@/api";
import { RelationCombobox } from "@/components/relation-combobox";
import { RelationMultiCombobox } from "@/components/relation-multi-combobox";
import { DataTable } from "@/components/data-table";
import { DatePicker } from "@/components/date-picker";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";

export function SubscriptionsPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { subscriptionId } = useParams();
  const navigate = useNavigate();
  const client = organizationApi(organization.id);
  const editor = location.pathname.endsWith("/new") || Boolean(subscriptionId);
  const [values, setValues] = useState<Subscription[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [prices, setPrices] = useState<Price[]>([]);
  const [current, setCurrent] = useState<Subscription | null>(null);
  const [customerID, setCustomerID] = useState("");
  const [priceIDs, setPriceIDs] = useState<string[]>([]);
  const [startDate, setStartDate] = useState(
    new Date().toISOString().slice(0, 10),
  );
  const [endDate, setEndDate] = useState("");
  const [error, setError] = useState("");
  useEffect(() => {
    void Promise.all([
      client.customers(),
      client.prices(),
      client.products(),
      ...(editor ? [] : [client.subscriptions()]),
    ])
      .then(([c, p, products, subscriptions]) => {
        setCustomers(c.customers);
        setPrices(p.prices);
        setProducts(products.products);
        if (subscriptions) setValues(subscriptions.subscriptions);
      })
      .catch((cause) => setError(cause.message));
  }, [organization.id, editor]);
  useEffect(() => {
    if (subscriptionId)
      void client
        .subscription(subscriptionId)
        .then(({ subscription }) => {
          setCurrent(subscription);
          setCustomerID(subscription.CustomerID);
          setPriceIDs(subscription.Items.map((item) => item.PriceID));
          setStartDate(subscription.StartDate.slice(0, 10));
          setEndDate(subscription.EndDate?.slice(0, 10) ?? "");
        })
        .catch((cause) => setError(cause.message));
  }, [subscriptionId]);
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      const body = {
        customer_id: customerID,
        start_date: new Date(`${startDate}T00:00:00Z`).toISOString(),
        end_date: endDate
          ? new Date(`${endDate}T00:00:00Z`).toISOString()
          : null,
        status: "active",
        metadata: {},
        price_ids: priceIDs,
      };
      if (current) await client.updateSubscription(current.ID, body);
      else await client.createSubscription(body);
      navigate("..");
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to save subscription",
      );
    }
  }
  if (editor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild><Link to=".."><ArrowLeft />Subscriptions</Link></Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">SUBSCRIPTIONS</p>
            <h1>
              {current
                ? `Subscription ${current.ID.slice(0, 8)}`
                : "Create subscription"}
            </h1>
            <p className="muted">
              Connect a customer to one or more catalog prices.
            </p>
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
              <Field>
                <FieldLabel>Customer</FieldLabel>
                <RelationCombobox
                  value={customerID}
                  onValueChange={setCustomerID}
                  options={customers.map((customer) => ({
                    value: customer.ID,
                    label: `${customer.FirstName} ${customer.LastName}`,
                  }))}
                  placeholder="Select customer"
                  searchPlaceholder="Search customers…"
                />
              </Field>
              <Field>
                <FieldLabel>Prices</FieldLabel>
                <RelationMultiCombobox
                  values={priceIDs}
                  onValuesChange={setPriceIDs}
                  options={prices.map((price) => ({
                    value: price.ID,
                    label:
                      products.find((product) => product.ID === price.ProductID)
                        ?.Name ?? price.Currency,
                    description: `${price.Currency} · ${price.Status}`,
                  }))}
                  placeholder="Select prices"
                  searchPlaceholder="Search prices…"
                />
              </Field>
              <Field>
                <FieldLabel>Start date</FieldLabel>
                <DatePicker
                  required
                  value={startDate}
                  onValueChange={setStartDate}
                />
              </Field>
              <Field>
                <FieldLabel>End date (optional)</FieldLabel>
                <DatePicker
                  min={startDate}
                  value={endDate}
                  onValueChange={setEndDate}
                  placeholder="No end date"
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
                <Button disabled={!customerID || priceIDs.length === 0}>
                  Save subscription
                </Button>
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
          <p className="eyebrow">SUBSCRIPTIONS</p>
          <h1>Subscriptions</h1>
          <p className="muted">
            Active customer access to your product prices.
          </p>
        </div>
        <Button asChild>
          <Link to="new">
            <Plus />
            Create subscription
          </Link>
        </Button>
      </div>
      <Card className="resource-list">
        <CardContent>
          <DataTable
            data={values}
            searchKey="Status"
            searchPlaceholder="Filter status…"
            onRowClick={(subscription) => navigate(subscription.ID)}
            columns={[
              {
                id: "customer",
                header: "Customer",
                cell: ({ row }) => {
                  const customer = customers.find(
                    (item) => item.ID === row.original.CustomerID,
                  );
                  return (
                    <span className="font-medium">
                      {customer
                        ? `${customer.FirstName} ${customer.LastName}`
                        : "—"}
                    </span>
                  );
                },
              },
              {
                id: "prices",
                header: "Prices",
                cell: ({ row }) => row.original.Items.length,
              },
              { accessorKey: "Status", header: "Status" },
              {
                accessorKey: "StartDate",
                header: "Start date",
                cell: ({ row }) =>
                  new Date(row.original.StartDate).toLocaleDateString(),
              },
            ]}
          />
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
