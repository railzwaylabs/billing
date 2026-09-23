import { useEffect, useMemo, useState } from "react";
import {
  Activity,
  ArrowRight,
  CircleDollarSign,
  FileText,
  Plus,
  Users,
  Workflow,
} from "lucide-react";
import { Link, useOutletContext } from "react-router-dom";
import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";
import {
  organizationApi,
  type Invoice,
  type Organization,
  type UsageSummary,
} from "@/api";
import { DataTable } from "@/components/data-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";

type DashboardData = {
  usageEvents: number;
  subscriptions: number;
  activeSubscriptions: number;
  customers: number;
  meters: number;
  products: number;
  prices: number;
  invoices: Invoice[];
  usageSummary: UsageSummary;
};

const emptyData: DashboardData = {
  usageEvents: 0,
  subscriptions: 0,
  activeSubscriptions: 0,
  customers: 0,
  meters: 0,
  products: 0,
  prices: 0,
  invoices: [],
  usageSummary: { from: "", to: "", interval: "month", points: [] },
};

const usageChartConfig = {
  events: { label: "Usage events", color: "var(--chart-2)" },
} satisfies ChartConfig;

export function DashboardPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const [data, setData] = useState<DashboardData>(emptyData);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [usageRange, setUsageRange] = useState<"7d" | "30d" | "3m" | "12m">(
    "12m",
  );

  useEffect(() => {
    const client = organizationApi(organization.id);
    setLoading(true);
    Promise.all([
      client.usageEvents(),
      client.subscriptions(),
      client.customers(),
      client.meters(),
      client.products(),
      client.prices(),
      client.invoices(),
      client.usageSummary(usageRange),
    ])
      .then(
        ([
          usage,
          subscriptions,
          customers,
          meters,
          products,
          prices,
          invoices,
          usageSummary,
        ]) => {
          setData({
            usageEvents: usage.events.length,
            subscriptions: subscriptions.subscriptions.length,
            activeSubscriptions: subscriptions.subscriptions.filter(
              (item) => item.Status.toLowerCase() === "active",
            ).length,
            customers: customers.customers.length,
            meters: meters.meters.length,
            products: products.products.length,
            prices: prices.prices.length,
            invoices: invoices.invoices,
            usageSummary,
          });
          setError("");
        },
      )
      .catch((cause) =>
        setError(
          cause instanceof Error ? cause.message : "Unable to load dashboard",
        ),
      )
      .finally(() => setLoading(false));
  }, [organization.id, usageRange]);

  const totalInvoiced = useMemo(
    () => data.invoices.reduce((sum, invoice) => sum + invoice.Total.Nanos, 0),
    [data.invoices],
  );
  const currency =
    data.invoices.find((invoice) => invoice.Total.Currency)?.Total.Currency ??
    "USD";
  const recentInvoices = useMemo(
    () =>
      [...data.invoices]
        .sort(
          (left, right) =>
            Date.parse(right.BillingPeriodEnd) -
            Date.parse(left.BillingPeriodEnd),
        )
        .slice(0, 5),
    [data.invoices],
  );

  return (
    <main className="w-full flex-1 space-y-6 p-4 md:p-6 lg:p-8">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1">
          <p className="text-sm font-medium text-muted-foreground">Overview</p>
          <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
            {organization.name}
          </h1>
          <p className="text-sm text-muted-foreground">
            Monitor usage, billing activity, and invoice readiness.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" asChild>
            <Link to="usage/ingest">
              <Activity />
              Ingest usage
            </Link>
          </Button>
          <Button asChild>
            <Link to="invoices/new">
              <Plus />
              Generate invoice
            </Link>
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          title="Usage events"
          value={loading ? "—" : formatNumber(data.usageEvents)}
          detail="All ingested events"
          icon={Activity}
        />
        <MetricCard
          title="Active subscriptions"
          value={loading ? "—" : formatNumber(data.activeSubscriptions)}
          detail={`${data.subscriptions} total subscriptions`}
          icon={Workflow}
        />
        <MetricCard
          title="Total invoiced"
          value={loading ? "—" : formatMoney(totalInvoiced, currency)}
          detail={`${data.invoices.length} generated invoices`}
          icon={CircleDollarSign}
        />
        <MetricCard
          title="Customers"
          value={loading ? "—" : formatNumber(data.customers)}
          detail="Billable customer records"
          icon={Users}
        />
      </section>

      <Card>
        <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle>Monthly usage</CardTitle>
            <CardDescription>
              Ingested event volume over the last 12 months.
            </CardDescription>
          </div>
          <div className="flex items-center gap-4">
            <div className="hidden text-right sm:block">
              <p className="text-2xl font-semibold tabular-nums">
                {loading
                  ? "—"
                  : formatNumber(
                      data.usageSummary.points.reduce(
                        (sum, point) => sum + point.event_count,
                        0,
                      ),
                    )}
              </p>
              <p className="text-xs text-muted-foreground">
                events in this period
              </p>
            </div>
            <Select
              value={usageRange}
              onValueChange={(value) =>
                setUsageRange(value as typeof usageRange)
              }
            >
              <SelectTrigger className="w-[150px]" aria-label="Usage range">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="7d">Last 7 days</SelectItem>
                <SelectItem value="30d">Last 30 days</SelectItem>
                <SelectItem value="3m">Last 3 months</SelectItem>
                <SelectItem value="12m">Last 12 months</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          <ChartContainer
            config={usageChartConfig}
            className="h-[280px] w-full aspect-auto"
          >
            <BarChart
              accessibilityLayer
              data={data.usageSummary.points.map((point) => ({
                ...point,
                events: point.event_count,
              }))}
              margin={{ left: 4, right: 4 }}
            >
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="bucket"
                tickLine={false}
                axisLine={false}
                tickMargin={10}
                minTickGap={24}
                tickFormatter={(value) =>
                  formatBucket(value, data.usageSummary.interval)
                }
              />
              <ChartTooltip
                cursor={false}
                content={
                  <ChartTooltipContent
                    labelFormatter={(label) =>
                      formatBucket(
                        String(label),
                        data.usageSummary.interval,
                        true,
                      )
                    }
                  />
                }
              />
              <Bar
                dataKey="events"
                fill="var(--color-events)"
                radius={[5, 5, 0, 0]}
              />
            </BarChart>
          </ChartContainer>
        </CardContent>
      </Card>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.6fr)_minmax(320px,0.8fr)]">
        <Card>
          <CardHeader className="flex-row items-start justify-between gap-4">
            <div>
              <CardTitle>Recent invoices</CardTitle>
              <CardDescription>
                Latest billing periods generated for this organization.
              </CardDescription>
            </div>
            <Button variant="ghost" size="sm" asChild>
              <Link to="invoices">
                View all <ArrowRight />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            <DataTable
              data={recentInvoices}
              columns={[
                {
                  accessorKey: "ID",
                  header: "Invoice",
                  cell: ({ row }) => (
                    <Link
                      className="font-medium hover:underline"
                      to={`invoices/${row.original.ID}`}
                    >
                      {shortID(row.original.ID)}
                    </Link>
                  ),
                },
                {
                  accessorKey: "BillingPeriodEnd",
                  header: "Period end",
                  cell: ({ row }) => formatDate(row.original.BillingPeriodEnd),
                },
                {
                  accessorKey: "Status",
                  header: "Status",
                  cell: ({ row }) => (
                    <Badge variant="secondary" className="capitalize">
                      {row.original.Status}
                    </Badge>
                  ),
                },
                {
                  id: "total",
                  header: "Total",
                  cell: ({ row }) => (
                    <span className="font-medium tabular-nums">
                      {formatMoney(
                        row.original.Total.Nanos,
                        row.original.Total.Currency,
                      )}
                    </span>
                  ),
                },
              ]}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Billing setup</CardTitle>
            <CardDescription>
              Your usage-to-invoice configuration.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-1">
            <SetupRow label="Meters" value={data.meters} href="meters" />
            <Separator />
            <SetupRow
              label="Products"
              value={data.products}
              href="catalog/products"
            />
            <Separator />
            <SetupRow
              label="Prices"
              value={data.prices}
              href="catalog/prices"
            />
            <Separator />
            <SetupRow
              label="Subscriptions"
              value={data.subscriptions}
              href="subscriptions"
            />
          </CardContent>
        </Card>
      </section>

      <Card>
        <CardHeader>
          <CardTitle>Quick actions</CardTitle>
          <CardDescription>
            Common tasks for operating usage-based billing.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <QuickAction
            title="Create customer"
            description="Add a billable customer."
            href="customers/new"
            icon={Users}
          />
          <QuickAction
            title="Create subscription"
            description="Attach prices to a customer."
            href="subscriptions/new"
            icon={Workflow}
          />
          <QuickAction
            title="Ingest usage"
            description="Submit one or more events."
            href="usage/ingest"
            icon={Activity}
          />
          <QuickAction
            title="Generate invoice"
            description="Create an invoice for a period."
            href="invoices/new"
            icon={FileText}
          />
        </CardContent>
      </Card>
    </main>
  );
}

function MetricCard({
  title,
  value,
  detail,
  icon: Icon,
}: {
  title: string;
  value: string;
  detail: string;
  icon: typeof Activity;
}) {
  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-semibold tracking-tight tabular-nums">
          {value}
        </div>
        <p className="mt-1 text-xs text-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  );
}

function SetupRow({
  label,
  value,
  href,
}: {
  label: string;
  value: number;
  href: string;
}) {
  return (
    <Button
      variant="ghost"
      className="h-auto w-full justify-between px-2 py-3"
      asChild
    >
      <Link to={href}>
        <span>{label}</span>
        <span className="flex items-center gap-2 text-muted-foreground">
          <span className="tabular-nums">{value}</span>
          <ArrowRight className="size-4" />
        </span>
      </Link>
    </Button>
  );
}

function QuickAction({
  title,
  description,
  href,
  icon: Icon,
}: {
  title: string;
  description: string;
  href: string;
  icon: typeof Activity;
}) {
  return (
    <Button
      variant="outline"
      className="h-auto justify-start gap-3 p-4 text-left"
      asChild
    >
      <Link to={href}>
        <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
          <Icon className="size-4" />
        </span>
        <span className="min-w-0">
          <span className="block font-medium">{title}</span>
          <span className="block truncate text-xs font-normal text-muted-foreground">
            {description}
          </span>
        </span>
      </Link>
    </Button>
  );
}

function formatNumber(value: number) {
  return new Intl.NumberFormat().format(value);
}
function formatBucket(
  value: string,
  interval: UsageSummary["interval"],
  long = false,
) {
  return new Intl.DateTimeFormat(
    undefined,
    interval === "month"
      ? {
          month: long ? "long" : "short",
          year: long ? "numeric" : undefined,
          timeZone: "UTC",
        }
      : { day: "numeric", month: long ? "long" : "short", timeZone: "UTC" },
  ).format(new Date(`${value}T00:00:00Z`));
}
function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(
    new Date(value),
  );
}
function formatMoney(nanos: number, currency: string) {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "USD",
  }).format(nanos / 1_000_000_000);
}
function shortID(value: string) {
  return value.length > 12 ? `${value.slice(0, 8)}…` : value;
}
