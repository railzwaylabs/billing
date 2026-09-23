import { useCallback, useEffect, useState } from "react";
import {
  Activity,
  ArrowLeft,
  ArrowRight,
  Cpu,
  MemoryStick,
  RefreshCw,
  Server,
} from "lucide-react";
import { Link, useOutletContext, useParams } from "react-router-dom";
import {
  api,
  type MonitoredService,
  type Organization,
  type ServiceHealthStatus,
} from "@/api";
import { ResourceAllocation } from "@/components/resource-allocation";
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
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

const statusStyles: Record<ServiceHealthStatus, string> = {
  healthy:
    "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300",
  degraded:
    "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-300",
  unhealthy: "border-destructive/30 bg-destructive/10 text-destructive",
  unknown: "border-border bg-muted text-muted-foreground",
};

export function MonitorPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const [services, setServices] = useState<MonitoredService[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.monitoredServices(organization.slug);
      setServices(response.services);
      setError("");
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Unable to load service health",
      );
    } finally {
      setLoading(false);
    }
  }, [organization.slug]);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => void load(), 60_000);
    return () => window.clearInterval(timer);
  }, [load]);

  return (
    <main className="w-full flex-1 space-y-6 p-4 md:p-6 lg:p-8">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1">
          <p className="text-sm font-medium text-muted-foreground">Developer</p>
          <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
            Monitor
          </h1>
          <p className="max-w-2xl text-sm text-muted-foreground">
            Service health and current utilization across the Billing runtime.
            Health is evaluated independently from CPU and memory usage.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void load()}
          disabled={loading}
        >
          <RefreshCw className={cn("size-4", loading && "animate-spin")} />
          Refresh
        </Button>
      </div>

      {error && (
        <p
          role="alert"
          className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {error}
        </p>
      )}

      {loading && services.length === 0 ? (
        <div className="grid gap-4 lg:grid-cols-3">
          {Array.from({ length: 3 }, (_, index) => (
            <Card key={index}>
              <CardHeader>
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-4 w-full" />
              </CardHeader>
              <CardContent className="grid grid-cols-2 gap-4">
                <Skeleton className="h-14" />
                <Skeleton className="h-14" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : services.length === 0 && !error ? (
        <Empty className="min-h-64 border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Server />
            </EmptyMedia>
            <EmptyTitle>No monitored services</EmptyTitle>
            <EmptyDescription>
              Services will appear after they are registered with monitoring.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="grid gap-4 lg:grid-cols-3">
          {services.map((service) => (
            <Link
              key={service.id}
              to={service.id}
              className="group rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              <ServiceCard service={service} />
            </Link>
          ))}
        </div>
      )}
    </main>
  );
}

export function ServiceMonitorPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const { serviceId = "" } = useParams();
  const [service, setService] = useState<MonitoredService | null>(null);
  const [error, setError] = useState("");
  const monitorPath = `/organizations/${organization.id}/developer/monitor`;

  const load = useCallback(async () => {
    try {
      const response = await api.monitoredService(organization.slug, serviceId);
      setService(response.service);
      setError("");
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Unable to load service",
      );
    }
  }, [organization.slug, serviceId]);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => void load(), 60_000);
    return () => window.clearInterval(timer);
  }, [load]);

  return (
    <main className="w-full flex-1 space-y-6 p-4 md:p-6 lg:p-8">
      <Button variant="ghost" size="sm" asChild className="-ml-3">
        <Link to={monitorPath}>
          <ArrowLeft />
          Monitor
        </Link>
      </Button>

      {error && (
        <p
          role="alert"
          className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {error}
        </p>
      )}

      {!service && !error ? (
        <div className="space-y-3">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-full max-w-xl" />
          <Skeleton className="h-28 w-full" />
        </div>
      ) : service ? (
        <>
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">
              Service monitoring
            </p>
            <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
              {service.name}
            </h1>
            <p className="max-w-2xl text-sm text-muted-foreground">
              {service.description}
            </p>
          </div>

          <Card>
            <CardHeader className="flex-row items-center justify-between gap-4">
              <div className="space-y-1">
                <CardTitle>Service health</CardTitle>
                <CardDescription>
                  Availability signal from the latest health checks.
                </CardDescription>
              </div>
              <StatusBadge status={service.status} />
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-3">
              <Metric label="CPU usage" value={formatCPU(service.cpu_usage)} />
              <Metric
                label="Memory usage"
                value={formatBytes(service.memory_usage_bytes)}
              />
              <Metric
                label="Last health check"
                value={formatUpdatedAt(service.updated_at)}
              />
            </CardContent>
          </Card>

          <ResourceAllocation serviceId={service.id} onRefresh={load} />
        </>
      ) : null}
    </main>
  );
}

function ServiceCard({ service }: { service: MonitoredService }) {
  return (
    <Card className="h-full transition-colors group-hover:border-foreground/20 group-hover:bg-muted/20">
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3">
            <span className="flex size-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">
              <Server className="size-5" />
            </span>
            <CardTitle>{service.name}</CardTitle>
          </div>
          <StatusBadge status={service.status} />
        </div>
        <CardDescription className="min-h-10">
          {service.description}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-3">
          <Metric
            icon={Cpu}
            label="CPU usage"
            value={formatCPU(service.cpu_usage)}
          />
          <Metric
            icon={MemoryStick}
            label="Memory usage"
            value={formatBytes(service.memory_usage_bytes)}
          />
        </div>
        <div className="flex items-center justify-between border-t pt-4 text-xs text-muted-foreground">
          <span>Updated {formatUpdatedAt(service.updated_at)}</span>
          <span className="inline-flex items-center gap-1 font-medium text-foreground">
            View details
            <ArrowRight className="size-3.5 transition-transform group-hover:translate-x-0.5" />
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function StatusBadge({ status }: { status: ServiceHealthStatus }) {
  return (
    <Badge variant="outline" className={statusStyles[status]}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </Badge>
  );
}

function Metric({
  icon: Icon = Activity,
  label,
  value,
}: {
  icon?: typeof Activity;
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-lg border bg-background p-3">
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <Icon className="size-3.5" />
        {label}
      </div>
      <p className="mt-1 truncate text-sm font-semibold tabular-nums">
        {value}
      </p>
    </div>
  );
}

function formatCPU(value?: number) {
  return value === undefined ? "Unavailable" : `${value.toFixed(3)} cores`;
}

function formatBytes(value?: number) {
  if (value === undefined) return "Unavailable";
  if (value >= 1024 ** 3) return `${(value / 1024 ** 3).toFixed(2)} GiB`;
  return `${(value / 1024 ** 2).toFixed(1)} MiB`;
}

function formatUpdatedAt(value?: string) {
  if (!value) return "Unavailable";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Unavailable";
  return date.toLocaleString([], {
    dateStyle: "medium",
    timeStyle: "short",
  });
}
