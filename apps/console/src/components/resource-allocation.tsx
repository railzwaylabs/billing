import { useCallback, useEffect, useMemo, useState } from "react";
import { Activity, Database, RefreshCw } from "lucide-react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useOutletContext } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { api, type MonitoringSample, type Organization } from "@/api";

type Period = "day" | "week" | "month";
type Sample = [number, string];
type ResourceHistory = {
  cpuUsed: Sample[];
  cpuAllocated: Sample[];
  memoryUsed: Sample[];
  memoryAllocated: Sample[];
  diskUsed: Sample[];
  diskAllocated: Sample[];
  networkReceive: Sample[];
  networkTransmit: Sample[];
};
type ChartRow = { timestamp: number } & Record<string, number>;

const allocationConfig = {
  used: { label: "Used", color: "var(--chart-2)" },
  allocated: { label: "Allocated", color: "var(--chart-1)" },
} satisfies ChartConfig;
const networkConfig = {
  receive: { label: "Receive", color: "var(--chart-2)" },
  transmit: { label: "Transmit", color: "var(--chart-1)" },
} satisfies ChartConfig;
const emptyHistory = (): ResourceHistory => ({
  cpuUsed: [],
  cpuAllocated: [],
  memoryUsed: [],
  memoryAllocated: [],
  diskUsed: [],
  diskAllocated: [],
  networkReceive: [],
  networkTransmit: [],
});

export function ResourceAllocation({
  serviceId,
  onRefresh,
}: {
  serviceId: string;
  onRefresh?: () => void;
}) {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const [period, setPeriod] = useState<Period>("day");
  const [history, setHistory] = useState<ResourceHistory>(emptyHistory);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [updatedAt, setUpdatedAt] = useState<Date | null>(null);

  const load = useCallback(
    async (selectedPeriod: Period) => {
      setLoading(true);
      try {
        const response = await api.monitoringResources(
          organization.slug,
          serviceId,
          selectedPeriod,
        );
        setHistory({
          cpuUsed: samples(response.cpu.used),
          cpuAllocated: samples(response.cpu.allocated),
          memoryUsed: samples(response.memory.used),
          memoryAllocated: samples(response.memory.allocated),
          diskUsed: samples(response.disk.used),
          diskAllocated: samples(response.disk.allocated),
          networkReceive: samples(response.network.receive),
          networkTransmit: samples(response.network.transmit),
        });
        setUpdatedAt(new Date());
        setError("");
      } catch (cause) {
        setError(
          cause instanceof Error
            ? cause.message
            : "Unable to load resource metrics",
        );
      } finally {
        setLoading(false);
      }
    },
    [organization.slug, serviceId],
  );

  useEffect(() => {
    void load(period);
    const timer = window.setInterval(() => void load(period), 60_000);
    return () => window.clearInterval(timer);
  }, [load, period]);

  const cpuData = useMemo(
    () =>
      mergeSeries(history.cpuUsed, history.cpuAllocated, "used", "allocated"),
    [history],
  );
  const memoryData = useMemo(
    () =>
      scaleSeries(
        mergeSeries(
          history.memoryUsed,
          history.memoryAllocated,
          "used",
          "allocated",
        ),
        ["used", "allocated"],
        bytesToGiB,
      ),
    [history],
  );
  const diskData = useMemo(
    () =>
      scaleSeries(
        mergeSeries(
          history.diskUsed,
          history.diskAllocated,
          "used",
          "allocated",
        ),
        ["used", "allocated"],
        bytesToGiB,
      ),
    [history],
  );
  const networkData = useMemo(
    () =>
      scaleSeries(
        mergeSeries(
          history.networkReceive,
          history.networkTransmit,
          "receive",
          "transmit",
        ),
        ["receive", "transmit"],
        bytesToMiB,
      ),
    [history],
  );

  return (
    <section aria-labelledby="resource-allocation-title" className="space-y-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Database className="size-4 text-muted-foreground" />
            <h2
              id="resource-allocation-title"
              className="text-lg font-semibold tracking-tight"
            >
              Resource allocation
            </h2>
            <Badge variant={error ? "destructive" : "secondary"}>
              {error ? "Unavailable" : loading ? "Updating" : "Live"}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Historical allocation for this service.
            {updatedAt && !error
              ? ` Updated ${updatedAt.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}.`
              : ""}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Tabs
            value={period}
            onValueChange={(value) => setPeriod(value as Period)}
          >
            <TabsList aria-label="Monitor period">
              <TabsTrigger value="day">Daily</TabsTrigger>
              <TabsTrigger value="week">Weekly</TabsTrigger>
              <TabsTrigger value="month">Monthly</TabsTrigger>
            </TabsList>
          </Tabs>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              void load(period);
              onRefresh?.();
            }}
            disabled={loading}
          >
            <RefreshCw className={cn("size-4", loading && "animate-spin")} />{" "}
            Refresh
          </Button>
        </div>
      </div>

      {error && (
        <p
          role="alert"
          className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          Metrics are not available. Make sure Prometheus and cAdvisor are
          running.
        </p>
      )}

      <div className="grid gap-4 xl:grid-cols-2">
        <ResourceTimeSeriesChart
          title="CPU allocation"
          description="CPU cores used compared with allocated cores."
          data={cpuData}
          config={allocationConfig}
          keys={["used", "allocated"]}
          period={period}
          unit=" cores"
        />
        <ResourceTimeSeriesChart
          title="Memory allocation"
          description="Memory used compared with allocated memory."
          data={memoryData}
          config={allocationConfig}
          keys={["used", "allocated"]}
          period={period}
          unit=" GiB"
        />
        <ResourceTimeSeriesChart
          title="Disk allocation"
          description="Disk used compared with available disk capacity."
          data={diskData}
          config={allocationConfig}
          keys={["used", "allocated"]}
          period={period}
          unit=" GiB"
        />
        <ResourceTimeSeriesChart
          title="Network throughput"
          description="Receive and transmit throughput."
          data={networkData}
          config={networkConfig}
          keys={["receive", "transmit"]}
          period={period}
          unit=" MiB/s"
        />
      </div>
    </section>
  );
}

function ResourceTimeSeriesChart({
  title,
  description,
  data,
  config,
  keys,
  period,
  unit,
}: {
  title: string;
  description: string;
  data: ChartRow[];
  config: ChartConfig;
  keys: string[];
  period: Period;
  unit: string;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        {data.length === 0 ? (
          <Empty className="min-h-[280px] border">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Activity />
              </EmptyMedia>
              <EmptyTitle>No monitoring data</EmptyTitle>
              <EmptyDescription>
                No samples were recorded for the selected period. Try another
                range or refresh the metrics.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <ChartContainer
            config={config}
            className="h-[280px] w-full aspect-auto"
          >
            <LineChart
              accessibilityLayer
              data={data}
              margin={{ left: 4, right: 4 }}
            >
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="timestamp"
                type="number"
                scale="time"
                domain={["dataMin", "dataMax"]}
                tickLine={false}
                axisLine={false}
                minTickGap={28}
                tickFormatter={(value) =>
                  formatTimestamp(Number(value), period)
                }
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                width={58}
                tickFormatter={(value) =>
                  `${formatCompact(Number(value))}${unit}`
                }
              />
              <ChartTooltip
                cursor={false}
                content={
                  <ChartTooltipContent
                    labelFormatter={(_, payload) =>
                      formatTimestamp(
                        payload?.[0]?.payload?.timestamp,
                        period,
                        true,
                      )
                    }
                    formatter={(value, name) => (
                      <div className="flex w-full min-w-36 items-center justify-between gap-4">
                        <span className="text-muted-foreground">
                          {config[String(name)]?.label ?? String(name)}
                        </span>
                        <span className="font-mono font-medium tabular-nums">
                          {Number(value).toFixed(2)}
                          {unit}
                        </span>
                      </div>
                    )}
                  />
                }
              />
              {keys.map((key, index) => (
                <Line
                  key={key}
                  type="monotone"
                  dataKey={key}
                  stroke={`var(--color-${key})`}
                  strokeWidth={2}
                  strokeDasharray={index === 0 ? undefined : "5 4"}
                  dot={data.length === 1 ? { r: 3 } : false}
                  activeDot={{ r: 4 }}
                  connectNulls
                />
              ))}
            </LineChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}

function mergeSeries(
  first: Sample[],
  second: Sample[],
  firstKey: string,
  secondKey: string,
): ChartRow[] {
  const rows = new Map<number, ChartRow>();
  for (const [timestamp, value] of first)
    rows.set(timestamp, { timestamp, [firstKey]: finiteNumber(value) });
  for (const [timestamp, value] of second) {
    const row = rows.get(timestamp) ?? { timestamp };
    row[secondKey] = finiteNumber(value);
    rows.set(timestamp, row);
  }
  return [...rows.values()].sort(
    (left, right) => left.timestamp - right.timestamp,
  );
}
function scaleSeries(
  rows: ChartRow[],
  keys: string[],
  scale: (value: number) => number,
) {
  return rows.map((row) => {
    const scaled = { ...row };
    for (const key of keys) scaled[key] = scale(row[key] ?? 0);
    return scaled;
  });
}
function finiteNumber(value: string) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}
function samples(values: MonitoringSample[]): Sample[] {
  return values.map((sample) => [sample.timestamp, String(sample.value)]);
}
function bytesToGiB(value: number) {
  return Number((value / 1024 ** 3).toFixed(3));
}
function bytesToMiB(value: number) {
  return Number((value / 1024 ** 2).toFixed(3));
}
function formatCompact(value: number) {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(
    value,
  );
}
function formatTimestamp(timestamp: unknown, period: Period, long = false) {
  const seconds = Number(timestamp);
  if (!Number.isFinite(seconds)) return "Timestamp unavailable";

  const date = new Date(seconds * 1000);
  if (Number.isNaN(date.getTime())) return "Timestamp unavailable";

  const options: Intl.DateTimeFormatOptions = long
    ? {
        day: "numeric",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }
    : period === "day"
      ? { hour: "2-digit", minute: "2-digit" }
      : period === "week"
        ? { weekday: "short", day: "numeric" }
        : { day: "numeric", month: "short" };
  return new Intl.DateTimeFormat(undefined, options).format(date);
}
