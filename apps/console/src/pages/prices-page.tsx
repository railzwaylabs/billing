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
  type Currency,
  type Meter,
  type Organization,
  type Price,
  type Product,
} from "@/api";
import { RelationCombobox } from "@/components/relation-combobox";
import { DataTable } from "@/components/data-table";
import { DatePicker } from "@/components/date-picker";
import { NumericInput } from "@/components/numeric-input";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { useCursorPagination } from "@/hooks/use-cursor-pagination";

type TierForm = { key: string; startQuantity: string; unitAmount: string };
type ChargeForm = {
  key: string;
  code: string;
  name: string;
  meterID: string;
  pricingModel: "per_unit" | "graduated";
  unitQuantity: string;
  tiers: TierForm[];
};
const emptyTier = (startQuantity = "0"): TierForm => ({
  key: crypto.randomUUID(),
  startQuantity,
  unitAmount: "0",
});
const emptyCharge = (): ChargeForm => ({
  key: crypto.randomUUID(),
  code: "",
  name: "",
  meterID: "",
  pricingModel: "per_unit",
  unitQuantity: "1",
  tiers: [emptyTier()],
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
const formatPeriod = (price: Price) =>
  `${new Date(price.EffectiveAt).toLocaleDateString()} – ${price.EffectiveUntil ? new Date(price.EffectiveUntil).toLocaleDateString() : "No end date"}`;

export function PricesPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { priceId } = useParams();
  const navigate = useNavigate();
  const client = organizationApi(organization.id);
  const listPath = `/organizations/${organization.id}/catalog/prices`;
  const editor = location.pathname.endsWith("/new") || Boolean(priceId);
  const [products, setProducts] = useState<Product[]>([]);
  const [meters, setMeters] = useState<Meter[]>([]);
  const [currencies, setCurrencies] = useState<Currency[]>([]);
  const [prices, setPrices] = useState<Price[]>([]);
  const [price, setPrice] = useState<Price | null>(null);
  const [productID, setProductID] = useState("");
  const [currency, setCurrency] = useState("USD");
  const [effectiveAt, setEffectiveAt] = useState(() => localDate(new Date()));
  const [effectiveUntil, setEffectiveUntil] = useState("");
  const [scheduleStart, setScheduleStart] = useState(false);
  const [scheduleEnd, setScheduleEnd] = useState(false);
  const [charges, setCharges] = useState<ChargeForm[]>([emptyCharge()]);
  const [error, setError] = useState("");
  const pagination = useCursorPagination();
  useEffect(() => {
    void Promise.all([
      client.products({ limit: 100 }),
      client.meters({ limit: 100 }),
      client.currencies(),
      client.prices(pagination.request),
    ])
      .then(([p, m, c, r]) => {
        setProducts(p.products);
        setMeters(m.meters);
        setCurrencies(c.currencies);
        setPrices(r.prices);
        pagination.setPageInfo(r.page_info);
      })
      .catch((cause) => setError(cause.message));
  }, [organization.id, pagination.cursor]);
  useEffect(() => {
    if (!priceId) return;
    void client
      .price(priceId)
      .then(({ price: value }) => {
        setPrice(value);
        setProductID(value.ProductID);
        setCurrency(value.Currency);
        setEffectiveAt(localDate(value.EffectiveAt));
        setScheduleStart(true);
        setEffectiveUntil(
          value.EffectiveUntil ? localDate(value.EffectiveUntil) : "",
        );
        setScheduleEnd(Boolean(value.EffectiveUntil));
        setCharges(
          value.Charges.map((charge) => ({
            key: charge.ID,
            code: charge.Code,
            name: charge.Name,
            meterID: charge.MeterID,
            pricingModel: charge.PricingModel,
            unitQuantity: decimal(charge.UnitQuantity.Micros, 6),
            tiers: charge.Tiers.map((tier) => ({
              key: tier.ID,
              startQuantity: decimal(tier.StartQuantity.Micros, 6),
              unitAmount: decimal(tier.UnitAmount.Nanos, 9),
            })),
          })),
        );
      })
      .catch((cause) => setError(cause.message));
  }, [priceId]);
  const updateCharge = (key: string, changes: Partial<ChargeForm>) =>
    setCharges((all) =>
      all.map((item) => (item.key === key ? { ...item, ...changes } : item)),
    );
  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      if (!charges.length) throw new Error("Add at least one price charge");
      const codes = charges.map((charge) => charge.code.trim());
      if (new Set(codes).size !== codes.length)
        throw new Error("Charge codes must be unique");
      const encodedCharges = charges.map((charge) => {
        const tiers = charge.tiers.map((tier) => ({
          start_quantity_micros: scaled(tier.startQuantity, 6),
          unit_amount_nanos: scaled(tier.unitAmount, 9),
        }));
        if (!charge.meterID || !charge.code.trim() || !charge.name.trim())
          throw new Error("Every charge needs a code, name, and meter");
        if (
          tiers[0]?.start_quantity_micros !== 0 ||
          tiers.some(
            (tier, index) =>
              index > 0 &&
              tier.start_quantity_micros <=
                tiers[index - 1].start_quantity_micros,
          )
        )
          throw new Error(
            `Tiers for ${charge.name || charge.code} must start at zero and increase`,
          );
        if (charge.pricingModel === "per_unit" && tiers.length !== 1)
          throw new Error("Per-unit charges require exactly one tier");
        return {
          code: charge.code,
          name: charge.name,
          meter_id: charge.meterID,
          pricing_model: charge.pricingModel,
          unit_quantity_micros: scaled(charge.unitQuantity, 6),
          tiers,
        };
      });
      const from = scheduleStart
        ? new Date(`${effectiveAt}T00:00:00Z`).toISOString()
        : new Date().toISOString();
      if (scheduleEnd && !effectiveUntil)
        throw new Error("Choose an effective end date");
      const until = scheduleEnd
        ? new Date(`${effectiveUntil}T00:00:00Z`).toISOString()
        : undefined;
      if (until && until <= from)
        throw new Error("Effective until must be after effective from");
      const body = {
        product_id: productID,
        currency,
        billing_interval: price?.BillingInterval ?? "month",
        interval_count: price?.IntervalCount ?? 1,
        effective_at: from,
        effective_until: until,
        status: price?.Status ?? "active",
        metadata: {},
        charges: encodedCharges,
      };
      if (priceId) await client.updatePrice(priceId, body);
      else await client.createPrice(body);
      navigate(listPath, { replace: true });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to save price");
    }
  }
  if (editor)
    return (
      <main className="content editor-page">
        <Button variant="ghost" size="sm" asChild>
          <Link to={listPath}>
            <ArrowLeft />
            Prices
          </Link>
        </Button>
        <div className="page-head">
          <div>
            <p className="eyebrow">CATALOG / PRICES</p>
            <h1>{price ? `${price.Currency} price` : "Create price"}</h1>
            <p className="muted">
              Group independently metered charges into one product price.
            </p>
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={submit}>
              <FieldGroup>
                <Field>
                  <FieldLabel hint="Catalog product that owns this price version.">
                    Product
                  </FieldLabel>
                  <RelationCombobox
                    value={productID}
                    onValueChange={setProductID}
                    options={products.map((item) => ({
                      value: item.ID,
                      label: item.Name,
                      description: item.Code,
                    }))}
                    placeholder="Select product"
                    searchPlaceholder="Search products…"
                  />
                </Field>
                <Field>
                  <FieldLabel hint="Reference currency used by every charge and invoice line from this price.">
                    Currency
                  </FieldLabel>
                  <RelationCombobox
                    value={currency}
                    onValueChange={setCurrency}
                    options={currencies.map((item) => ({
                      value: item.Code,
                      label: `${item.Code} — ${item.Name}`,
                      description: item.Symbol,
                    }))}
                    placeholder="Select currency"
                    searchPlaceholder="Search currencies…"
                  />
                </Field>
                <div className="grid gap-6 md:grid-cols-2">
                  <Field className="rounded-lg border p-4">
                    <div className="flex items-start gap-3">
                      <Checkbox
                        id="schedule-price-start"
                        checked={scheduleStart}
                        onCheckedChange={(checked) =>
                          setScheduleStart(checked === true)
                        }
                      />
                      <div className="space-y-1">
                        <FieldLabel
                          htmlFor="schedule-price-start"
                          hint="Enable this to activate the price on a future date instead of immediately."
                        >
                          Schedule start
                        </FieldLabel>
                        <FieldDescription>
                          Otherwise active immediately.
                        </FieldDescription>
                      </div>
                    </div>
                    {scheduleStart && (
                      <DatePicker
                        required
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
                          if (!checked) setEffectiveUntil("");
                        }}
                      />
                      <div className="space-y-1">
                        <FieldLabel
                          htmlFor="schedule-price-end"
                          hint="Enable this to stop selecting the price after a specific date."
                        >
                          Set an end date
                        </FieldLabel>
                        <FieldDescription>
                          Otherwise remains active.
                        </FieldDescription>
                      </div>
                    </div>
                    {scheduleEnd && (
                      <DatePicker
                        min={effectiveAt}
                        required
                        value={effectiveUntil}
                        onValueChange={setEffectiveUntil}
                      />
                    )}
                  </Field>
                </div>
                <Separator />
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="font-semibold">Price charges</h2>
                    <p className="text-sm text-muted-foreground">
                      Each charge connects one meter to its own pricing tiers.
                    </p>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setCharges((all) => [...all, emptyCharge()])}
                  >
                    <Plus />
                    Add charge
                  </Button>
                </div>
                <div className="space-y-4">
                  {charges.map((charge, chargeIndex) => (
                    <Card key={charge.key}>
                      <CardContent className="space-y-5 pt-6">
                        <div className="flex items-center justify-between">
                          <h3 className="font-medium">
                            Charge {chargeIndex + 1}
                          </h3>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            disabled={charges.length === 1}
                            onClick={() =>
                              setCharges((all) =>
                                all.filter((item) => item.key !== charge.key),
                              )
                            }
                          >
                            <Trash2 />
                          </Button>
                        </div>
                        <div className="grid gap-4 md:grid-cols-2">
                          <Field>
                            <FieldLabel hint="Stable identifier for this independently metered charge.">
                              Code
                            </FieldLabel>
                            <Input
                              type="text"
                              inputMode="text"
                              required
                              pattern="[A-Za-z0-9][A-Za-z0-9._\-]*"
                              placeholder="api_requests"
                              value={charge.code}
                              onChange={(e) =>
                                updateCharge(charge.key, {
                                  code: e.target.value,
                                })
                              }
                            />
                          </Field>
                          <Field>
                            <FieldLabel hint="Human-readable charge name used on invoice lines.">
                              Name
                            </FieldLabel>
                            <Input
                              type="text"
                              inputMode="text"
                              required
                              placeholder="API requests"
                              value={charge.name}
                              onChange={(e) =>
                                updateCharge(charge.key, {
                                  name: e.target.value,
                                })
                              }
                            />
                          </Field>
                        </div>
                        <Field>
                          <FieldLabel hint="Meter whose aggregated usage is rated by this charge.">
                            Meter
                          </FieldLabel>
                          <RelationCombobox
                            value={charge.meterID}
                            onValueChange={(meterID) =>
                              updateCharge(charge.key, { meterID })
                            }
                            options={meters.map((item) => ({
                              value: item.ID,
                              label: item.Name,
                              description: `${item.Code} · ${item.Unit}`,
                            }))}
                            placeholder="Select meter"
                            searchPlaceholder="Search meters…"
                          />
                        </Field>
                        <div className="grid gap-4 md:grid-cols-2">
                          <Field>
                            <FieldLabel hint="Per unit applies one rate; graduated applies each tier rate to its quantity band.">
                              Pricing model
                            </FieldLabel>
                            <Select
                              value={charge.pricingModel}
                              onValueChange={(value) => {
                                const pricingModel =
                                  value as ChargeForm["pricingModel"];
                                updateCharge(charge.key, {
                                  pricingModel,
                                  tiers:
                                    pricingModel === "per_unit"
                                      ? [charge.tiers[0] ?? emptyTier()]
                                      : charge.tiers,
                                });
                              }}
                            >
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="per_unit">
                                  Per unit
                                </SelectItem>
                                <SelectItem value="graduated">
                                  Graduated tiers
                                </SelectItem>
                              </SelectContent>
                            </Select>
                          </Field>
                          <Field>
                            <FieldLabel hint="Number of metered units represented by one priced unit, for example 1000 requests.">
                              Pricing unit quantity
                            </FieldLabel>
                            <NumericInput
                              required
                              placeholder="1"
                              value={charge.unitQuantity}
                              onChange={(e) =>
                                updateCharge(charge.key, {
                                  unitQuantity: e.target.value,
                                })
                              }
                            />
                          </Field>
                        </div>
                        <div className="flex items-center justify-between">
                          <h4 className="text-sm font-medium">Tiers</h4>
                          {charge.pricingModel === "graduated" && (
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              onClick={() =>
                                updateCharge(charge.key, {
                                  tiers: [...charge.tiers, emptyTier("")],
                                })
                              }
                            >
                              <Plus />
                              Add tier
                            </Button>
                          )}
                        </div>
                        <div className="space-y-3">
                          {charge.tiers.map((tier, tierIndex) => (
                            <div
                              key={tier.key}
                              className="grid gap-3 rounded-lg border p-4 sm:grid-cols-[1fr_1fr_auto] sm:items-end"
                            >
                              <Field>
                                <FieldLabel hint="Inclusive usage quantity where this tier begins; the first tier always starts at zero.">
                                  Starts at
                                </FieldLabel>
                                <NumericInput
                                  required
                                  disabled={tierIndex === 0}
                                  placeholder="0"
                                  value={tier.startQuantity}
                                  onChange={(e) =>
                                    updateCharge(charge.key, {
                                      tiers: charge.tiers.map((item) =>
                                        item.key === tier.key
                                          ? {
                                              ...item,
                                              startQuantity: e.target.value,
                                            }
                                          : item,
                                      ),
                                    })
                                  }
                                />
                              </Field>
                              <Field>
                                <FieldLabel hint="Price charged per pricing unit within this tier.">
                                  Unit amount ({currency})
                                </FieldLabel>
                                <NumericInput
                                  required
                                  placeholder="0.00"
                                  value={tier.unitAmount}
                                  onChange={(e) =>
                                    updateCharge(charge.key, {
                                      tiers: charge.tiers.map((item) =>
                                        item.key === tier.key
                                          ? {
                                              ...item,
                                              unitAmount: e.target.value,
                                            }
                                          : item,
                                      ),
                                    })
                                  }
                                />
                              </Field>
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                disabled={
                                  tierIndex === 0 ||
                                  charge.pricingModel === "per_unit"
                                }
                                onClick={() =>
                                  updateCharge(charge.key, {
                                    tiers: charge.tiers.filter(
                                      (item) => item.key !== tier.key,
                                    ),
                                  })
                                }
                              >
                                <Trash2 />
                              </Button>
                            </div>
                          ))}
                        </div>
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
                  <Button disabled={!productID || !currency}>Save price</Button>
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
            Versioned product prices with one or more metered charges.
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
            cursorPagination={{
              cursor: pagination.cursor,
              pageInfo: pagination.pageInfo,
              onCursorChange: pagination.setCursor,
            }}
            searchKey="Currency"
            searchPlaceholder="Search currency…"
            onRowClick={(item) => navigate(item.ID)}
            columns={[
              {
                id: "product",
                header: "Product",
                cell: ({ row }) => (
                  <span className="font-medium">
                    {products.find((item) => item.ID === row.original.ProductID)
                      ?.Name ?? "—"}
                  </span>
                ),
              },
              { accessorKey: "Currency", header: "Currency" },
              {
                id: "charges",
                header: "Charges",
                cell: ({ row }) => row.original.Charges.length,
              },
              {
                id: "interval",
                header: "Interval",
                cell: ({ row }) =>
                  `${row.original.IntervalCount} ${row.original.BillingInterval}`,
              },
              {
                id: "effective",
                header: "Effective period",
                cell: ({ row }) => formatPeriod(row.original),
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
