import { useEffect, useMemo, useState } from "react";
import {
  ArrowLeft,
  CalendarDays,
  Plus,
  ReceiptText,
  Trash2,
} from "lucide-react";
import {
  Link,
  useNavigate,
  useOutletContext,
  useParams,
} from "react-router-dom";
import {
  organizationApi,
  type Customer,
  type Invoice,
  type Meter,
  type Organization,
  type Price,
  type Product,
  type Subscription,
} from "@/api";
import { RelationCombobox } from "@/components/relation-combobox";
import { DataTable } from "@/components/data-table";
import { DatePicker } from "@/components/date-picker";
import { NumericInput } from "@/components/numeric-input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useCursorPagination } from "@/hooks/use-cursor-pagination";

type LineDraft = {
  subscriptionId: string;
  priceId: string;
  chargeId: string;
  description: string;
  quantity: string;
  unitAmount: string;
};
const line = (): LineDraft => ({
  subscriptionId: "",
  priceId: "",
  chargeId: "",
  description: "",
  quantity: "0",
  unitAmount: "0",
});
const scaled = (v: string, s: number) => {
  const [w = "0", f = ""] = v.split(".");
  return Number(
    BigInt(w || "0") * 10n ** BigInt(s) +
      BigInt((f + "0".repeat(s)).slice(0, s)),
  );
};

export function InvoicesPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { invoiceId } = useParams();
  const navigate = useNavigate();
  const editor = location.pathname.endsWith("/new") || Boolean(invoiceId);
  const client = organizationApi(organization.id);
  const listPath = `/organizations/${organization.id}/invoices`;
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [prices, setPrices] = useState<Price[]>([]);
  const [meters, setMeters] = useState<Meter[]>([]);
  const [editing, setEditing] = useState<Invoice | null>(null);
  const [customerID, setCustomerID] = useState("");
  const [start, setStart] = useState(
    new Date(new Date().getFullYear(), new Date().getMonth(), 1)
      .toISOString()
      .slice(0, 10),
  );
  const [end, setEnd] = useState(new Date().toISOString().slice(0, 10));
  const [currency, setCurrency] = useState("USD");
  const [tax, setTax] = useState("0");
  const [lines, setLines] = useState<LineDraft[]>([line()]);
  const [error, setError] = useState("");
  const pagination = useCursorPagination();

  const subtotal = useMemo(
    () =>
      lines.reduce(
        (total, item) =>
          total + Number(item.quantity || 0) * Number(item.unitAmount || 0),
        0,
      ),
    [lines],
  );
  const total = subtotal + Number(tax || 0);

  const load = async () => {
    const [i, c, s, p, products, m] = await Promise.all([
      client.invoices(pagination.request),
      client.customers({ limit: 100 }),
      client.subscriptions({ limit: 100 }),
      client.prices({ limit: 100 }),
      client.products({ limit: 100 }),
      client.meters({ limit: 100 }),
    ]);
    setInvoices(i.invoices);
    setCustomers(c.customers);
    setSubscriptions(s.subscriptions);
    setPrices(p.prices);
    setProducts(products.products);
    setMeters(m.meters);
    pagination.setPageInfo(i.page_info);
  };

  useEffect(() => {
    void load().catch((e) => setError(e.message));
  }, [organization.id, pagination.cursor]);

  useEffect(() => {
    if (invoiceId)
      void client
        .invoice(invoiceId)
        .then(({ invoice }) => edit(invoice))
        .catch((cause) => setError(cause.message));
  }, [invoiceId]);

  const customerSubscriptions = useMemo(
    () => subscriptions.filter((v) => v.CustomerID === customerID),
    [subscriptions, customerID],
  );

  function change(i: number, p: Partial<LineDraft>) {
    setLines((v) => v.map((x, n) => (n === i ? { ...x, ...p } : x)));
  }

  function edit(v: Invoice) {
    setEditing(v);
    setCustomerID(v.CustomerID);
    setStart(v.BillingPeriodStart.slice(0, 10));
    setEnd(v.BillingPeriodEnd.slice(0, 10));
    setCurrency(v.Total.Currency);
    setTax(String(v.Tax.Nanos / 1e9));
    setLines(
      v.Lines.map((l) => ({
        subscriptionId: l.SubscriptionID,
        priceId: l.PriceID,
        chargeId: l.PriceChargeID,
        description: l.Description,
        quantity: String(l.UsageQuantity.Micros / 1e6),
        unitAmount: String(l.UnitAmount.Nanos / 1e9),
      })),
    );
  }

  async function submit() {
    try {
      const payload = {
        customer_id: customerID,
        status: "draft",
        billing_period_start: new Date(`${start}T00:00:00Z`).toISOString(),
        billing_period_end: new Date(`${end}T00:00:00Z`).toISOString(),
        currency,
        tax_nanos: scaled(tax, 9),
        lines: lines.map((v) => {
          const subscription = subscriptions.find(
            (s) => s.ID === v.subscriptionId,
          )!;
          const item = subscription.Items.find((i) => i.PriceID === v.priceId);
          const price = prices.find((p) => p.ID === v.priceId)!;
          const product = products.find((p) => p.ID === price.ProductID)!;
          const charge =
            price.Charges.find((charge) => charge.ID === v.chargeId) ??
            price.Charges[0];
          if (!charge) throw new Error("Selected price has no charge");
          const meter = meters.find((m) => m.ID === charge.MeterID)!;
          const quantity = scaled(v.quantity, 6);
          const unitAmount = scaled(v.unitAmount, 9);
          return {
            subscription_id: v.subscriptionId,
            subscription_item_id: item?.ID,
            product_id: product.ID,
            price_id: price.ID,
            price_charge_id: charge.ID,
            meter_id: meter.ID,
            description: v.description,
            usage_quantity_micros: quantity,
            unit: meter.Unit,
            pricing_unit_quantity_micros: charge.UnitQuantity.Micros,
            unit_amount_nanos: unitAmount,
            amount_nanos: Number(
              (BigInt(quantity) * BigInt(unitAmount)) / 1_000_000n,
            ),
            pricing_details: {},
          };
        }),
      };
      if (editing) {
        await client.updateInvoice(editing.ID, payload);
      } else {
        await client.createInvoice(payload);
      }
      navigate(listPath, { replace: true });
    } catch (c) {
      setError(c instanceof Error ? c.message : "Unable to save invoice");
    }
  }

  async function generate() {
    try {
      setError("");
      const inclusiveEnd = new Date(`${end}T00:00:00Z`);
      inclusiveEnd.setUTCDate(inclusiveEnd.getUTCDate() + 1);
      await client.generateInvoices({
        period_start: new Date(`${start}T00:00:00Z`).toISOString(),
        period_end: inclusiveEnd.toISOString(),
      });
      navigate(listPath, { replace: true });
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to generate invoices",
      );
    }
  }

  if (editor && !invoiceId) {
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild>
          <Link to={listPath}>
            <ArrowLeft />
            Invoices
          </Link>
        </Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">INVOICES</p>
            <h1>Generate invoices</h1>
            <p className="muted">
              Rate usage for every active subscription in the selected billing
              period.
            </p>
          </div>
        </div>
        <Card className="resource-form">
          <CardHeader>
            <h2>Billing period</h2>
          </CardHeader>
          <CardContent className="space-y-5 pt-6">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <label className="text-sm font-medium" htmlFor="period-start">
                  Starts on
                </label>
                <DatePicker value={start} onValueChange={setStart} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium" htmlFor="period-end">
                  Usage through
                </label>
                <DatePicker min={start} value={end} onValueChange={setEnd} />
                <p className="text-xs text-muted-foreground">
                  Usage on this date is included.
                </p>
              </div>
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
                type="button"
                disabled={!start || !end || end < start}
                onClick={generate}
              >
                Generate invoices
              </Button>
            </div>
          </CardContent>
        </Card>
      </main>
    );
  }

  return (
    <main className="content">
      {editor && (
        <Button variant="ghost" size="sm" asChild>
          <Link to={listPath}>
            <ArrowLeft />
            Invoices
          </Link>
        </Button>
      )}
      <div className="page-head">
        <div>
          <p className="eyebrow">INVOICES</p>
          <h1>
            {editor
              ? editing
                ? editing.InvoiceNumber
                : "Generate invoices"
              : "Invoices"}
          </h1>
          <p className="muted">
            Create and review deterministic invoice records.
          </p>
        </div>
        {!editor && (
          <Button asChild>
            <Link to="new">
              <Plus />
              Generate invoices
            </Link>
          </Button>
        )}
      </div>
      {editor && (
        <div className="grid max-w-6xl gap-6 xl:grid-cols-[minmax(0,1fr)_18rem]">
          <Card className="overflow-hidden shadow-sm">
            <CardHeader className="border-b bg-muted/20">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="space-y-1">
                  <CardTitle className="flex items-center gap-2 text-lg">
                    <ReceiptText className="size-5 text-muted-foreground" />
                    Invoice details
                  </CardTitle>
                  <CardDescription>
                    Review the billing period and line items before saving this
                    draft.
                  </CardDescription>
                </div>
                <Badge variant="secondary" className="w-fit capitalize">
                  {editing?.Status ?? "draft"}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-8 p-6">
              <div className="grid gap-5 md:grid-cols-2">
                <Field className="md:col-span-2">
                  <FieldLabel hint="Customer receiving this draft invoice.">
                    Customer
                  </FieldLabel>
                  <RelationCombobox
                    value={customerID}
                    onValueChange={(value) => {
                      setCustomerID(value);
                      setLines([line()]);
                    }}
                    options={customers.map((customer) => ({
                      value: customer.ID,
                      label: `${customer.FirstName} ${customer.LastName}`,
                    }))}
                    placeholder="Select customer"
                  />
                </Field>
                <Field>
                  <FieldLabel hint="Inclusive start of the usage period being invoiced.">
                    Period start
                  </FieldLabel>
                  <DatePicker value={start} onValueChange={setStart} />
                </Field>
                <Field>
                  <FieldLabel hint="Exclusive end of the usage period being invoiced.">
                    Period end
                  </FieldLabel>
                  <DatePicker min={start} value={end} onValueChange={setEnd} />
                </Field>
              </div>

              <div className="space-y-3">
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <h3 className="text-sm font-semibold">Line items</h3>
                    <p className="text-sm text-muted-foreground">
                      Usage and pricing included in this invoice.
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setLines([...lines, line()])}
                  >
                    <Plus /> Add line
                  </Button>
                </div>
                <div className="overflow-hidden rounded-lg border">
                  <Table className="min-w-[920px]">
                    <TableHeader className="bg-muted/40">
                      <TableRow>
                        <TableHead className="w-[210px] pl-4">
                          Subscription
                        </TableHead>
                        <TableHead className="w-[190px]">Price</TableHead>
                        <TableHead className="w-[190px]">Charge</TableHead>
                        <TableHead>Description</TableHead>
                        <TableHead className="w-28 text-right">
                          Quantity
                        </TableHead>
                        <TableHead className="w-32 text-right">
                          Unit price
                        </TableHead>
                        <TableHead className="w-32 text-right">
                          Amount
                        </TableHead>
                        <TableHead className="w-12">
                          <span className="sr-only">Actions</span>
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {lines.map((item, index) => {
                        const subscription = subscriptions.find(
                          (value) => value.ID === item.subscriptionId,
                        );
                        const available = prices.filter((price) =>
                          subscription?.Items.some(
                            (entry) => entry.PriceID === price.ID,
                          ),
                        );
                        const amount =
                          Number(item.quantity || 0) *
                          Number(item.unitAmount || 0);
                        return (
                          <TableRow
                            key={index}
                            className="align-top hover:bg-transparent"
                          >
                            <TableCell className="pl-4">
                              <RelationCombobox
                                value={item.subscriptionId}
                                onValueChange={(value) =>
                                  change(index, {
                                    subscriptionId: value,
                                    priceId: "",
                                    chargeId: "",
                                  })
                                }
                                options={customerSubscriptions.map((value) => ({
                                  value: value.ID,
                                  label: `Subscription ${value.ID.slice(0, 8)}`,
                                  description: value.Status,
                                }))}
                                placeholder="Select subscription"
                              />
                            </TableCell>
                            <TableCell>
                              <RelationCombobox
                                value={item.priceId}
                                onValueChange={(value) => {
                                  const selected = prices.find(
                                    (price) => price.ID === value,
                                  );
                                  const charge = selected?.Charges[0];
                                  if (selected) setCurrency(selected.Currency);
                                  change(index, {
                                    priceId: value,
                                    chargeId: charge?.ID ?? "",
                                    description:
                                      charge?.Name ?? item.description,
                                    unitAmount: charge
                                      ? String(
                                          (charge.Tiers[0]?.UnitAmount.Nanos ??
                                            0) / 1e9,
                                        )
                                      : item.unitAmount,
                                  });
                                }}
                                options={available.map((price) => ({
                                  value: price.ID,
                                  label:
                                    products.find(
                                      (product) =>
                                        product.ID === price.ProductID,
                                    )?.Name ?? price.Currency,
                                  description: price.Currency,
                                }))}
                                placeholder="Select price"
                              />
                            </TableCell>
                            <TableCell>
                              <RelationCombobox
                                value={item.chargeId}
                                onValueChange={(value) => {
                                  const charge = prices
                                    .find((price) => price.ID === item.priceId)
                                    ?.Charges.find(
                                      (entry) => entry.ID === value,
                                    );
                                  change(index, {
                                    chargeId: value,
                                    description:
                                      charge?.Name ?? item.description,
                                    unitAmount: charge
                                      ? String(
                                          (charge.Tiers[0]?.UnitAmount.Nanos ??
                                            0) / 1e9,
                                        )
                                      : item.unitAmount,
                                  });
                                }}
                                options={(
                                  prices.find(
                                    (price) => price.ID === item.priceId,
                                  )?.Charges ?? []
                                ).map((charge) => ({
                                  value: charge.ID,
                                  label: charge.Name,
                                  description: charge.Code,
                                }))}
                                placeholder="Select charge"
                              />
                            </TableCell>
                            <TableCell>
                              <Input
                                type="text"
                                inputMode="text"
                                aria-label="Description"
                                placeholder="Line description"
                                value={item.description}
                                onChange={(event) =>
                                  change(index, {
                                    description: event.target.value,
                                  })
                                }
                              />
                            </TableCell>
                            <TableCell>
                              <NumericInput
                                className="text-right tabular-nums"
                                aria-label="Quantity"
                                placeholder="0"
                                value={item.quantity}
                                onChange={(event) =>
                                  change(index, {
                                    quantity: event.target.value,
                                  })
                                }
                              />
                            </TableCell>
                            <TableCell>
                              <NumericInput
                                className="text-right tabular-nums"
                                aria-label="Unit price"
                                placeholder="0.00"
                                value={item.unitAmount}
                                onChange={(event) =>
                                  change(index, {
                                    unitAmount: event.target.value,
                                  })
                                }
                              />
                            </TableCell>
                            <TableCell className="pt-4 text-right font-medium tabular-nums">
                              {currency} {amount.toFixed(2)}
                            </TableCell>
                            <TableCell>
                              <Button
                                size="icon"
                                variant="ghost"
                                aria-label="Remove line"
                                disabled={lines.length === 1}
                                onClick={() =>
                                  setLines(
                                    lines.filter(
                                      (_, position) => position !== index,
                                    ),
                                  )
                                }
                              >
                                <Trash2 />
                              </Button>
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </div>
              </div>
              {error && <p className="form-error">{error}</p>}
            </CardContent>
            <CardFooter className="justify-between border-t bg-muted/20 p-6">
              <Button variant="outline" onClick={() => navigate(listPath)}>
                Cancel
              </Button>
              <Button
                disabled={
                  !customerID ||
                  lines.some(
                    (item) =>
                      !item.subscriptionId || !item.priceId || !item.chargeId,
                  )
                }
                onClick={submit}
              >
                Save draft
              </Button>
            </CardFooter>
          </Card>

          <div className="space-y-4">
            <Card className="shadow-sm">
              <CardHeader>
                <CardTitle className="text-base">Summary</CardTitle>
                <CardDescription className="flex items-center gap-2">
                  <CalendarDays className="size-4" />
                  {start} — {end}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <Field>
                  <FieldLabel hint="Currency is derived from the selected price and cannot be entered manually.">
                    Currency
                  </FieldLabel>
                  <div className="rounded-md border bg-muted/30 px-3 py-2 text-sm font-medium">
                    {currency}
                    <span className="ml-2 font-normal text-muted-foreground">
                      Derived from the selected price
                    </span>
                  </div>
                </Field>
                <Field>
                  <FieldLabel
                    htmlFor="invoice-tax"
                    hint="Total tax amount added to this draft invoice."
                  >
                    Tax
                  </FieldLabel>
                  <NumericInput
                    id="invoice-tax"
                    className="text-right tabular-nums"
                    placeholder="0.00"
                    value={tax}
                    onChange={(event) => setTax(event.target.value)}
                  />
                </Field>
                <div className="space-y-3 border-t pt-4 text-sm">
                  <div className="flex justify-between text-muted-foreground">
                    <span>Subtotal</span>
                    <span className="tabular-nums">
                      {currency} {subtotal.toFixed(2)}
                    </span>
                  </div>
                  <div className="flex justify-between text-muted-foreground">
                    <span>Tax</span>
                    <span className="tabular-nums">
                      {currency} {Number(tax || 0).toFixed(2)}
                    </span>
                  </div>
                  <div className="flex justify-between border-t pt-3 text-base font-semibold">
                    <span>Total</span>
                    <span className="tabular-nums">
                      {currency} {total.toFixed(2)}
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>
            {editing && (
              <p className="break-all px-1 text-xs text-muted-foreground">
                Invoice ID: {editing.ID}
              </p>
            )}
          </div>
        </div>
      )}
      {!editor && (
        <Card className="resource-list">
          <CardContent>
            <DataTable
              data={invoices}
              cursorPagination={{
                cursor: pagination.cursor,
                pageInfo: pagination.pageInfo,
                onCursorChange: pagination.setCursor,
              }}
              searchKey="InvoiceNumber"
              searchPlaceholder="Search invoice number…"
              onRowClick={(invoice) => navigate(invoice.ID)}
              columns={[
                {
                  accessorKey: "InvoiceNumber",
                  header: "Invoice number",
                  cell: ({ row }) => (
                    <span className="font-medium">
                      {row.original.InvoiceNumber}
                    </span>
                  ),
                },
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
                  id: "period",
                  header: "Period",
                  cell: ({ row }) =>
                    `${new Date(row.original.BillingPeriodStart).toLocaleDateString()} – ${new Date(row.original.BillingPeriodEnd).toLocaleDateString()}`,
                },
                { accessorKey: "Status", header: "Status" },
                {
                  id: "total",
                  header: "Total",
                  cell: ({ row }) =>
                    `${row.original.Total.Currency} ${(row.original.Total.Nanos / 1e9).toFixed(2)}`,
                },
              ]}
            />
          </CardContent>
        </Card>
      )}
    </main>
  );
}
