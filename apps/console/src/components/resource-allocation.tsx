import { useCallback, useEffect, useMemo, useState } from "react";
import { Database, RefreshCw } from "lucide-react";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { api, type MonitoringSample } from "@/api";

type Period = "day" | "week" | "month";
type Sample = [number, string];
type ResourceHistory = {
  cpuUsed: Sample[]; cpuAllocated: Sample[];
  memoryUsed: Sample[]; memoryAllocated: Sample[];
  diskUsed: Sample[]; diskAllocated: Sample[];
  networkReceive: Sample[]; networkTransmit: Sample[];
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
  cpuUsed: [], cpuAllocated: [], memoryUsed: [], memoryAllocated: [],
  diskUsed: [], diskAllocated: [], networkReceive: [], networkTransmit: [],
});

export function ResourceAllocation() {
  const [period, setPeriod] = useState<Period>("day");
  const [history, setHistory] = useState<ResourceHistory>(emptyHistory);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [updatedAt, setUpdatedAt] = useState<Date | null>(null);

  const load = useCallback(async (selectedPeriod: Period) => {
    setLoading(true);
    try {
      const response = await api.monitoringResources(selectedPeriod);
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
      setError(cause instanceof Error ? cause.message : "Unable to load resource metrics");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load(period);
    const timer = window.setInterval(() => void load(period), 60_000);
    return () => window.clearInterval(timer);
  }, [load, period]);

  const cpuData = useMemo(() => mergeSeries(history.cpuUsed, history.cpuAllocated, "used", "allocated"), [history]);
  const memoryData = useMemo(() => scaleSeries(mergeSeries(history.memoryUsed, history.memoryAllocated, "used", "allocated"), ["used", "allocated"], bytesToGiB), [history]);
  const diskData = useMemo(() => scaleSeries(mergeSeries(history.diskUsed, history.diskAllocated, "used", "allocated"), ["used", "allocated"], bytesToGiB), [history]);
  const networkData = useMemo(() => scaleSeries(mergeSeries(history.networkReceive, history.networkTransmit, "receive", "transmit"), ["receive", "transmit"], bytesToMiB), [history]);

  return (
    <section aria-labelledby="resource-allocation-title" className="space-y-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <Database className="size-4 text-muted-foreground" />
            <h2 id="resource-allocation-title" className="text-lg font-semibold tracking-tight">Resource allocation</h2>
            <Badge variant={error ? "destructive" : "secondary"}>{error ? "Unavailable" : loading ? "Updating" : "Live"}</Badge>
          </div>
          <p className="text-sm text-muted-foreground">Historical allocation across Billing containers.{updatedAt && !error ? ` Updated ${updatedAt.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}.` : ""}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Tabs value={period} onValueChange={(value) => setPeriod(value as Period)}>
            <TabsList aria-label="Monitor period">
              <TabsTrigger value="day">Daily</TabsTrigger>
              <TabsTrigger value="week">Weekly</TabsTrigger>
              <TabsTrigger value="month">Monthly</TabsTrigger>
            </TabsList>
          </Tabs>
          <Button variant="outline" size="sm" onClick={() => void load(period)} disabled={loading}>
            <RefreshCw className={cn("size-4", loading && "animate-spin")} /> Refresh
          </Button>
        </div>
      </div>

      {error && <p role="alert" className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">Metrics are not available. Make sure Prometheus and cAdvisor are running.</p>}

      <div className="grid gap-4 xl:grid-cols-2">
        <ResourceBarChart title="CPU allocation" description="CPU cores used compared with allocated cores." data={cpuData} config={allocationConfig} keys={["used", "allocated"]} period={period} unit=" cores" />
        <ResourceBarChart title="Memory allocation" description="Memory used compared with allocated memory." data={memoryData} config={allocationConfig} keys={["used", "allocated"]} period={period} unit=" GiB" />
        <ResourceBarChart title="Disk allocation" description="Disk used compared with available disk capacity." data={diskData} config={allocationConfig} keys={["used", "allocated"]} period={period} unit=" GiB" />
        <ResourceBarChart title="Network throughput" description="Receive and transmit throughput." data={networkData} config={networkConfig} keys={["receive", "transmit"]} period={period} unit=" MiB/s" />
      </div>
    </section>
  );
}

function ResourceBarChart({ title, description, data, config, keys, period, unit }: {
  title: string; description: string; data: ChartRow[]; config: ChartConfig;
  keys: string[]; period: Period; unit: string;
}) {
  return (
    <Card>
      <CardHeader><CardTitle>{title}</CardTitle><CardDescription>{description}</CardDescription></CardHeader>
      <CardContent>
        <ChartContainer config={config} className="h-[280px] w-full aspect-auto">
          <BarChart accessibilityLayer data={data} margin={{ left: 4, right: 4 }}>
            <CartesianGrid vertical={false} />
            <XAxis dataKey="timestamp" type="number" scale="time" domain={["dataMin", "dataMax"]} tickLine={false} axisLine={false} minTickGap={28} tickFormatter={(value) => formatTimestamp(Number(value), period)} />
            <YAxis tickLine={false} axisLine={false} width={58} tickFormatter={(value) => `${formatCompact(Number(value))}${unit}`} />
            <ChartTooltip cursor={false} content={<ChartTooltipContent labelFormatter={(value) => formatTimestamp(Number(value), period, true)} formatter={(value, name) => <div className="flex w-full min-w-36 items-center justify-between gap-4"><span className="text-muted-foreground">{config[String(name)]?.label ?? String(name)}</span><span className="font-mono font-medium tabular-nums">{Number(value).toFixed(2)}{unit}</span></div>} />} />
            {keys.map((key, index) => <Bar key={key} dataKey={key} fill={`var(--color-${key})`} radius={[4, 4, 0, 0]} opacity={index === 0 ? 1 : 0.55} />)}
          </BarChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

function mergeSeries(first: Sample[], second: Sample[], firstKey: string, secondKey: string): ChartRow[] {
  const rows = new Map<number, ChartRow>();
  for (const [timestamp, value] of first) rows.set(timestamp, { timestamp, [firstKey]: finiteNumber(value) });
  for (const [timestamp, value] of second) {
    const row = rows.get(timestamp) ?? { timestamp };
    row[secondKey] = finiteNumber(value);
    rows.set(timestamp, row);
  }
  return [...rows.values()].sort((left, right) => left.timestamp - right.timestamp);
}
function scaleSeries(rows: ChartRow[], keys: string[], scale: (value: number) => number) {
  return rows.map((row) => {
    const scaled = { ...row };
    for (const key of keys) scaled[key] = scale(row[key] ?? 0);
    return scaled;
  });
}
function finiteNumber(value: string) { const parsed = Number(value); return Number.isFinite(parsed) ? parsed : 0; }
function samples(values: MonitoringSample[]): Sample[] { return values.map((sample) => [sample.timestamp, String(sample.value)]); }
function bytesToGiB(value: number) { return Number((value / 1024 ** 3).toFixed(3)); }
function bytesToMiB(value: number) { return Number((value / 1024 ** 2).toFixed(3)); }
function formatCompact(value: number) { return new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(value); }
function formatTimestamp(timestamp: number, period: Period, long = false) {
  const options: Intl.DateTimeFormatOptions = period === "day"
    ? { hour: "2-digit", minute: "2-digit" }
    : period === "week"
      ? { weekday: long ? "long" : "short", hour: long ? "2-digit" : undefined }
      : { day: "numeric", month: long ? "long" : "short" };
  return new Intl.DateTimeFormat(undefined, options).format(new Date(timestamp * 1000));
}
