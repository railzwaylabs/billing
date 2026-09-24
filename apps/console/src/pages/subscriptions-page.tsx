import { type FormEvent, useEffect, useState } from "react";
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
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
import { DataTable } from "@/components/data-table";
import { DatePicker } from "@/components/date-picker";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { useCursorPagination } from "@/hooks/use-cursor-pagination";

type ItemForm = {
  key: string;
  id?: string;
  priceID: string;
  startAt: string;
  endAt: string;
  hasEnd: boolean;
};
const today = () => new Date().toISOString().slice(0, 10);
const emptyItem = (startAt = today()): ItemForm => ({
  key: crypto.randomUUID(),
  priceID: "",
  startAt,
  endAt: "",
  hasEnd: false,
});
const iso = (date: string) => new Date(`${date}T00:00:00Z`).toISOString();

export function SubscriptionsPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { subscriptionId } = useParams();
  const navigate = useNavigate();
  const client = organizationApi(organization.id);
  const listPath = `/organizations/${organization.id}/subscriptions`;
  const editor = location.pathname.endsWith("/new") || Boolean(subscriptionId);
  const [values, setValues] = useState<Subscription[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [prices, setPrices] = useState<Price[]>([]);
  const [current, setCurrent] = useState<Subscription | null>(null);
  const [customerID, setCustomerID] = useState("");
  const [items, setItems] = useState<ItemForm[]>([emptyItem()]);
  const [startDate, setStartDate] = useState(today());
  const [endDate, setEndDate] = useState("");
  const [error, setError] = useState("");
  const pagination = useCursorPagination();
  useEffect(() => {
    void Promise.all([
      client.customers({ limit: 100 }),
      client.prices({ limit: 100 }),
      client.products({ limit: 100 }),
      ...(editor ? [] : [client.subscriptions(pagination.request)]),
    ])
      .then(([c, p, productsResult, subscriptions]) => {
        setCustomers(c.customers);
        setPrices(p.prices);
        setProducts(productsResult.products);
        if (subscriptions) {
          setValues(subscriptions.subscriptions);
          pagination.setPageInfo(subscriptions.page_info);
        }
      })
      .catch((cause) => setError(cause.message));
  }, [organization.id, editor, pagination.cursor]);
  useEffect(() => {
    if (!subscriptionId) return;
    void client
      .subscription(subscriptionId)
      .then(({ subscription }) => {
        setCurrent(subscription);
        setCustomerID(subscription.CustomerID);
        setStartDate(subscription.StartDate.slice(0, 10));
        setEndDate(subscription.EndDate?.slice(0, 10) ?? "");
        setItems(
          subscription.Items.map((item) => ({
            key: item.ID,
            id: item.ID,
            priceID: item.PriceID,
            startAt: item.StartAt.slice(0, 10),
            endAt: item.EndAt?.slice(0, 10) ?? "",
            hasEnd: Boolean(item.EndAt),
          })),
        );
      })
      .catch((cause) => setError(cause.message));
  }, [subscriptionId]);
  const updateItem = (key: string, changes: Partial<ItemForm>) =>
    setItems((all) =>
      all.map((item) => (item.key === key ? { ...item, ...changes } : item)),
    );
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      if (items.some((item) => !item.priceID || !item.startAt))
        throw new Error("Every item needs a price and start date");
      const currencies = new Set(
        items.map(
          (item) => prices.find((price) => price.ID === item.priceID)?.Currency,
        ),
      );
      if (currencies.size > 1)
        throw new Error(
          "All items in a subscription must use the same currency",
        );
      const body = {
        customer_id: customerID,
        start_date: iso(startDate),
        end_date: endDate ? iso(endDate) : null,
        status: "active",
        metadata: {},
        items: items.map((item) => ({
          id: item.id,
          price_id: item.priceID,
          start_at: iso(item.startAt),
          end_at: item.hasEnd && item.endAt ? iso(item.endAt) : null,
        })),
      };
      if (current) await client.updateSubscription(current.ID, body);
      else await client.createSubscription(body);
      navigate(listPath, { replace: true });
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to save subscription",
      );
    }
  }
  if (editor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild>
          <Link to={listPath}>
            <ArrowLeft />
            Subscriptions
          </Link>
        </Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">SUBSCRIPTIONS</p>
            <h1>
              {current
                ? `Subscription ${current.ID.slice(0, 8)}`
                : "Create subscription"}
            </h1>
            <p className="muted">
              Maintain one customer subscription and activate products when they
              are enabled.
            </p>
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
                <Field>
                  <FieldLabel hint="Customer billed by this subscription.">
                    Customer
                  </FieldLabel>
                  <RelationCombobox
                    value={customerID}
                    onValueChange={setCustomerID}
                    options={customers.map((item) => ({
                      value: item.ID,
                      label: `${item.FirstName} ${item.LastName}`,
                    }))}
                    placeholder="Select customer"
                    searchPlaceholder="Search customers…"
                  />
                </Field>
                <div className="grid gap-4 md:grid-cols-2">
                  <Field>
                    <FieldLabel hint="First calendar date on which the subscription can be billed.">
                      Subscription start
                    </FieldLabel>
                    <DatePicker
                      required
                      value={startDate}
                      onValueChange={(value) => {
                        setStartDate(value);
                        setItems((all) =>
                          all.map((item) =>
                            item.id ? item : { ...item, startAt: value },
                          ),
                        );
                      }}
                    />
                  </Field>
                  <Field>
                    <FieldLabel hint="Inclusive final service date. Leave empty for an ongoing subscription.">
                      Subscription end (optional)
                    </FieldLabel>
                    <DatePicker
                      min={startDate}
                      value={endDate}
                      onValueChange={setEndDate}
                      placeholder="No end date"
                    />
                  </Field>
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="font-semibold">Subscription items</h2>
                    <p className="text-sm text-muted-foreground">
                      Add a product price now or schedule it for later.
                    </p>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() =>
                      setItems((all) => [...all, emptyItem(startDate)])
                    }
                  >
                    <Plus />
                    Add item
                  </Button>
                </div>
                <div className="space-y-4">
                  {items.map((item, index) => (
                    <Card key={item.key}>
                      <CardContent className="space-y-4 pt-6">
                        <div className="flex items-center justify-between">
                          <h3 className="font-medium">Item {index + 1}</h3>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            disabled={items.length === 1 || Boolean(item.id)}
                            onClick={() =>
                              setItems((all) =>
                                all.filter((value) => value.key !== item.key),
                              )
                            }
                          >
                            <Trash2 />
                          </Button>
                        </div>
                        <Field>
                          <FieldLabel hint="Effective catalog price attached to this subscription item.">
                            Price
                          </FieldLabel>
                          <RelationCombobox
                            value={item.priceID}
                            onValueChange={(priceID) =>
                              updateItem(item.key, { priceID })
                            }
                            options={prices.map((price) => ({
                              value: price.ID,
                              label:
                                products.find(
                                  (product) => product.ID === price.ProductID,
                                )?.Name ?? price.ID,
                              description: `${price.Currency} · ${price.Charges.length} charge${price.Charges.length === 1 ? "" : "s"}`,
                            }))}
                            placeholder="Select price"
                            searchPlaceholder="Search prices…"
                          />
                        </Field>
                        <Field>
                          <FieldLabel hint="Date this product price starts contributing to billing.">
                            Effective from
                          </FieldLabel>
                          <DatePicker
                            min={startDate}
                            required
                            value={item.startAt}
                            onValueChange={(startAt) =>
                              updateItem(item.key, { startAt })
                            }
                          />
                        </Field>
                        <Field className="rounded-lg border p-4">
                          <div className="flex items-start gap-3">
                            <Checkbox
                              checked={item.hasEnd}
                              onCheckedChange={(checked) =>
                                updateItem(item.key, {
                                  hasEnd: checked === true,
                                  endAt: checked ? item.endAt : "",
                                })
                              }
                            />
                            <div>
                              <FieldLabel hint="End only this price item without ending the customer subscription.">
                                Stop this item
                              </FieldLabel>
                              <FieldDescription>
                                The subscription remains active; only this
                                product price stops.
                              </FieldDescription>
                            </div>
                          </div>
                          {item.hasEnd && (
                            <DatePicker
                              min={item.startAt}
                              required
                              value={item.endAt}
                              onValueChange={(endAt) =>
                                updateItem(item.key, { endAt })
                              }
                            />
                          )}
                        </Field>
                      </CardContent>
                    </Card>
                  ))}
                </div>
                {error && <p className="form-error">{error}</p>}
                <div className="form-actions">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => navigate(listPath)}
                  >
                    Cancel
                  </Button>
                  <Button
                    disabled={
                      !customerID || items.some((item) => !item.priceID)
                    }
                  >
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
            Customer billing agreements and their effective product items.
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
            cursorPagination={{
              cursor: pagination.cursor,
              pageInfo: pagination.pageInfo,
              onCursorChange: pagination.setCursor,
            }}
            searchKey="Status"
            searchPlaceholder="Filter status…"
            onRowClick={(item) => navigate(item.ID)}
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
                id: "items",
                header: "Items",
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
