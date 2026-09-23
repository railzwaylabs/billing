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
  type Organization,
  type Price,
  type Product,
} from "@/api";
import { RelationCombobox } from "@/components/relation-combobox";
import { DataTable } from "@/components/data-table";
import { DatePicker } from "@/components/date-picker";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Separator } from "@/components/ui/separator";

type TierForm = {
  key: string;
  startQuantity: string;
  unitAmount: string;
};

const emptyTier = (startQuantity = ""): TierForm => ({
  key: crypto.randomUUID(),
  startQuantity,
  unitAmount: "0",
});
const scaled = (value: string, scale: number) => {
  const [whole = "0", fraction = ""] = value.split(".");
  return Number(
    BigInt(whole || "0") * 10n ** BigInt(scale) +
    BigInt((fraction + "0".repeat(scale)).slice(0, scale)),
  );
};
const decimal = (value: number, scale: number) =>
  (value / 10 ** scale).toString();
const localDate = (value: string | Date) => {
  const date = value instanceof Date ? value : new Date(value);
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 10);
};
const formatEffectivePeriod = (price: Price) => {
  const from = new Date(price.EffectiveAt).toLocaleString();
  const until = price.EffectiveUntil
    ? new Date(price.EffectiveUntil).toLocaleString()
    : "No end date";
  return `${from} – ${until}`;
};
export function PricesPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { priceId } = useParams();
  const navigate = useNavigate();
  const client = organizationApi(organization.id);
  const editor = location.pathname.endsWith("/new") || Boolean(priceId);
  const [products, setProducts] = useState<Product[]>([]);
  const [prices, setPrices] = useState<Price[]>([]);
  const [price, setPrice] = useState<Price | null>(null);
  const [productID, setProductID] = useState("");
  const [currency, setCurrency] = useState("USD");
  const [unitQuantity, setUnitQuantity] = useState("1");
  const [effectiveAt, setEffectiveAt] = useState(() => localDate(new Date()));
  const [effectiveUntil, setEffectiveUntil] = useState("");
  const [scheduleStart, setScheduleStart] = useState(false);
  const [scheduleEnd, setScheduleEnd] = useState(false);
  const [tiers, setTiers] = useState<TierForm[]>([emptyTier("0")]);
  const [error, setError] = useState("");
  useEffect(() => {
    void Promise.all([client.products(), client.prices()])
      .then(([p, r]) => {
        setProducts(p.products);
        setPrices(r.prices);
      })
      .catch((cause) => setError(cause.message));
  }, [organization.id]);
  useEffect(() => {
    if (priceId)
      void client
        .price(priceId)
        .then(({ price: value }) => {
          setPrice(value);
          setProductID(value.ProductID);
          setCurrency(value.Currency);
          setUnitQuantity(decimal(value.UnitQuantity.Micros, 6));
          setEffectiveAt(localDate(value.EffectiveAt));
          setScheduleStart(true);
          setEffectiveUntil(
            value.EffectiveUntil ? localDate(value.EffectiveUntil) : "",
          );
          setScheduleEnd(Boolean(value.EffectiveUntil));
          setTiers(value.Tiers.length > 0
            ? value.Tiers.map((tier) => ({
              key: tier.ID,
              startQuantity: decimal(tier.StartQuantity.Micros, 6),
              unitAmount: decimal(tier.UnitAmount.Nanos, 9),
            }))
            : [emptyTier("0")]);
        })
        .catch((cause) => setError(cause.message));
  }, [priceId]);
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      const encodedTiers = tiers.map((tier) => ({
        start_quantity_micros: scaled(tier.startQuantity, 6),
        unit_amount_nanos: scaled(tier.unitAmount, 9),
      }));
      const starts = encodedTiers.map((tier) => tier.start_quantity_micros);
      if (!starts.includes(0)) throw new Error("The first tier must start at 0");
      if (new Set(starts).size !== starts.length) throw new Error("Tier start quantities must be unique");
      if (encodedTiers.some((tier) => tier.start_quantity_micros < 0 || tier.unit_amount_nanos < 0))
        throw new Error("Tier quantities and amounts cannot be negative");
      if (starts.some((start, index) => index > 0 && start <= starts[index - 1]))
        throw new Error("Each tier must start above the previous tier");
      const effectiveAtISO = scheduleStart
        ? new Date(`${effectiveAt}T00:00:00Z`).toISOString()
        : new Date().toISOString();
      if (scheduleEnd && !effectiveUntil)
        throw new Error("Choose when this price should stop being effective");
      const effectiveUntilISO = scheduleEnd && effectiveUntil
        ? new Date(`${effectiveUntil}T00:00:00Z`).toISOString()
        : undefined;
      if (
        effectiveUntilISO &&
        new Date(effectiveUntilISO) <= new Date(effectiveAtISO)
      )
        throw new Error("Effective until must be after effective from");
      const body = {
        product_id: productID,
        currency,
        unit_quantity_micros: scaled(unitQuantity, 6),
        aggregation_interval: price?.AggregationInterval ?? "month",
        billing_interval: price?.BillingInterval ?? "month",
        interval_count: price?.IntervalCount ?? 1,
        effective_at: effectiveAtISO,
        effective_until: effectiveUntilISO,
        status: price?.Status ?? "active",
        metadata: {},
        tiers: encodedTiers,
      };
      if (priceId) await client.updatePrice(priceId, body);
      else await client.createPrice(body);
      navigate("..");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to save price");
    }
  }
  if (editor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild><Link to=".."><ArrowLeft />Prices</Link></Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">CATALOG / PRICES</p>
            <h1>{price ? `${price.Currency} price` : "Create price"}</h1>
            <p className="muted">
              Configure fixed-point rates without floating-point loss.
            </p>
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
                <Field>
                  <FieldLabel>Product</FieldLabel>
                  <RelationCombobox
                    value={productID}
                    onValueChange={setProductID}
                    options={products.map((product) => ({
                      value: product.ID,
                      label: product.Name,
                      description: product.Code,
                    }))}
                    placeholder="Select product"
                    searchPlaceholder="Search products…"
                  />
                </Field>
                <Field>
                  <FieldLabel>Currency</FieldLabel>
                  <Input
                    type="text"
                    inputMode="text"
                    pattern="[A-Za-z]{3}"
                    autoComplete="off"
                    spellCheck={false}
                    required
                    maxLength={3}
                    value={currency}
                    onChange={(event) =>
                      setCurrency(event.target.value.toUpperCase())
                    }
                  />
                </Field>
                <Field>
                  <FieldLabel>Pricing unit</FieldLabel>
                  <Input
                    type="text"
                    required
                    inputMode="decimal"
                    pattern="[0-9]+(?:\.[0-9]+)?"
                    value={unitQuantity}
                    onChange={(event) => setUnitQuantity(event.target.value)}
                  />
                </Field>
                <div className="grid gap-6 md:grid-cols-2">
                  <Field className="rounded-lg border p-4">
                    <div className="flex items-start gap-3">
                      <Checkbox
                        id="schedule-price-start"
                        checked={scheduleStart}
                        onCheckedChange={(checked) => setScheduleStart(checked === true)}
                      />
                      <div className="space-y-1">
                        <FieldLabel htmlFor="schedule-price-start">Schedule start</FieldLabel>
                        <FieldDescription>
                          Otherwise the price is effective immediately after saving.
                        </FieldDescription>
                      </div>
                    </div>
                    {scheduleStart && (
                      <DatePicker
                        required
                        min={new Date().toISOString().slice(0, 10)}
                        value={effectiveAt}
                        onValueChange={setEffectiveAt}
                      />
                    )}
                  </Field>
                  <Field className="rounded-lg border p-4">
                    <div className="flex items-start gap-3">
                      <Checkbox
                        id="schedule-price-end"
                        checked={scheduleEnd}
                        onCheckedChange={(checked) => {
                          setScheduleEnd(checked === true);
                          if (checked !== true) setEffectiveUntil("");
                        }}
                      />
                      <div className="space-y-1">
                        <FieldLabel htmlFor="schedule-price-end">Set an end date</FieldLabel>
                        <FieldDescription>
                          Otherwise the price remains effective without an end date.
                        </FieldDescription>
                      </div>
                    </div>
                    {scheduleEnd && (
                      <DatePicker
                        min={effectiveAt}
                        required
                        value={effectiveUntil}
                        onValueChange={setEffectiveUntil}
                        placeholder="Pick an end date"
                      />
                    )}
                  </Field>
                </div>
                <Separator />
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <h2 className="text-sm font-semibold">Pricing tiers</h2>
                    <p className="text-sm text-muted-foreground">Define the unit price starting at each usage threshold.</p>
                  </div>
                  <Button type="button" variant="outline" size="sm" onClick={() => setTiers((current) => [...current, emptyTier()])}>
                    <Plus /> Add tier
                  </Button>
                </div>
                <div className="space-y-3">
                  {tiers.map((tier, index) => (
                    <div key={tier.key} className="grid gap-3 rounded-lg border p-4 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
                      <Field>
                        <FieldLabel>{index === 0 ? "Starts at" : `Tier ${index + 1} starts at`}</FieldLabel>
                        <Input
                          type="text"
                          required
                          inputMode="decimal"
                          pattern="[0-9]+(?:\.[0-9]+)?"
                          value={tier.startQuantity}
                          disabled={index === 0}
                          onChange={(event) => setTiers((current) => current.map((item) => item.key === tier.key ? { ...item, startQuantity: event.target.value } : item))}
                        />
                      </Field>
                      <Field>
                        <FieldLabel>Unit amount ({currency})</FieldLabel>
                        <Input
                          type="text"
                          required
                          inputMode="decimal"
                          pattern="[0-9]+(?:\.[0-9]+)?"
                          value={tier.unitAmount}
                          onChange={(event) => setTiers((current) => current.map((item) => item.key === tier.key ? { ...item, unitAmount: event.target.value } : item))}
                        />
                      </Field>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={`Remove tier ${index + 1}`}
                        disabled={index === 0}
                        onClick={() => setTiers((current) => current.filter((item) => item.key !== tier.key))}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  ))}
                </div>
                {error && <p className="form-error">{error}</p>}
                <div className="form-actions">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => navigate("..")}
                  >
                    Cancel
                  </Button>
                  <Button disabled={!productID}>Save price</Button>
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
          <p className="eyebrow">CATALOG</p>
          <h1>Prices</h1>
          <p className="muted">
            Versioned product rates and billing intervals.
          </p>
        </div>
        <Button asChild>
          <Link to="new">
            <Plus />
            Create price
          </Link>
        </Button>
      </div>
      <Card className="resource-list">
        <CardContent>
          <DataTable
            data={prices}
            searchKey="Currency"
            searchPlaceholder="Search currency…"
            onRowClick={(price) => navigate(price.ID)}
            columns={[
              {
                id: "product",
                header: "Product",
                cell: ({ row }) => (
                  <span className="font-medium">
                    {products.find(
                      (product) => product.ID === row.original.ProductID,
                    )?.Name ?? "—"}
                  </span>
                ),
              },
              { accessorKey: "Currency", header: "Currency" },
              {
                id: "amount",
                header: "Pricing",
                cell: ({ row }) =>
                  row.original.Tiers?.length
                    ? row.original.Tiers.length === 1
                      ? `${decimal(row.original.Tiers[0].UnitAmount.Nanos, 9)} ${row.original.Currency}`
                      : `${row.original.Tiers.length} tiers`
                    : "—",
              },
              {
                id: "interval",
                header: "Interval",
                cell: ({ row }) =>
                  `${row.original.IntervalCount} ${row.original.BillingInterval}`,
              },
              {
                id: "effectivePeriod",
                header: "Effective period",
                cell: ({ row }) => formatEffectivePeriod(row.original),
              },
              { accessorKey: "Status", header: "Status" },
            ]}
          />
          {error && <p className="form-error">{error}</p>}
        </CardContent>
      </Card>
    </main>
  );
}
