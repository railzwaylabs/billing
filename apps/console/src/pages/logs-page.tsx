import {
  type FormEvent,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { RefreshCw, Search } from "lucide-react";
import { useOutletContext } from "react-router-dom";
import { api, type LogEntry, type LogService, type Organization } from "@/api";
import {
  formatLogTimestamp,
  LogLevelBadge,
  LogViewerTable,
  logServiceName,
} from "@/components/log-viewer-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

type TimeRange = "15m" | "1h" | "6h" | "24h";

const rangeMilliseconds: Record<TimeRange, number> = {
  "15m": 15 * 60 * 1000,
  "1h": 60 * 60 * 1000,
  "6h": 6 * 60 * 60 * 1000,
  "24h": 24 * 60 * 60 * 1000,
};

export function LogsPage() {
  const { organization } = useOutletContext<{ organization: Organization }>();
  const [services, setServices] = useState<LogService[]>([]);
  const [service, setService] = useState("");
  const [level, setLevel] = useState("all");
  const [range, setRange] = useState<TimeRange>("1h");
  const [search, setSearch] = useState("");
  const [appliedSearch, setAppliedSearch] = useState("");
  const [live, setLive] = useState(false);
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [selected, setSelected] = useState<LogEntry | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<string>();
  const queryWindow = useRef<
    | {
        from: string;
        to: string;
      }
    | undefined
  >(undefined);
  const [error, setError] = useState("");
  const requestID = useRef(0);

  useEffect(() => {
    let active = true;
    api
      .logServices(organization.slug)
      .then((result) => {
        if (!active) return;
        setServices(result.services);
        setService((current) => current || result.services[0]?.id || "");
      })
      .catch((cause) => {
        if (active)
          setError(
            cause instanceof Error
              ? cause.message
              : "Unable to load log services",
          );
      });
    return () => {
      active = false;
    };
  }, [organization.slug]);

  const load = useCallback(async () => {
    if (!service) return;
    const currentRequest = ++requestID.current;
    setLoading(true);
    setLoadingMore(false);
    setEntries([]);
    setNextCursor(undefined);
    const to = new Date();
    const from = new Date(to.getTime() - rangeMilliseconds[range]);
    const window = { from: from.toISOString(), to: to.toISOString() };
    queryWindow.current = window;
    try {
      const result = await api.logs({
        organization: organization.slug,
        service,
        level: level === "all" ? undefined : level,
        search: appliedSearch || undefined,
        ...window,
        limit: 100,
      });
      if (requestID.current !== currentRequest) return;
      setEntries(result.entries);
      setNextCursor(result.next_cursor);
      setError("");
    } catch (cause) {
      if (requestID.current !== currentRequest) return;
      setError(
        cause instanceof Error ? cause.message : "Unable to query service logs",
      );
    } finally {
      if (requestID.current === currentRequest) setLoading(false);
    }
  }, [appliedSearch, level, organization.slug, range, service]);

  const loadMore = useCallback(async () => {
    if (!service || !nextCursor || loadingMore) {
      return;
    }

    const window = queryWindow.current;
    if (!window) return;

    setLive(false);
    setLoadingMore(true);
    const currentRequest = requestID.current;
    try {
      const result = await api.logs({
        organization: organization.slug,
        service,
        level: level === "all" ? undefined : level,
        search: appliedSearch || undefined,
        ...window,
        limit: 100,
        cursor: nextCursor,
      });
      if (requestID.current !== currentRequest) return;
      setEntries((current) => appendUniqueLogs(current, result.entries));
      setNextCursor(result.next_cursor);
      setError("");
    } catch (cause) {
      if (requestID.current !== currentRequest) return;
      setError(
        cause instanceof Error
          ? cause.message
          : "Unable to load older service logs",
      );
    } finally {
      if (requestID.current === currentRequest) setLoadingMore(false);
    }
  }, [
    appliedSearch,
    level,
    loadingMore,
    nextCursor,
    organization.slug,
    service,
  ]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!live) return;
    const timer = window.setInterval(() => void load(), 5_000);
    return () => window.clearInterval(timer);
  }, [live, load]);

  function applyFilters(event: FormEvent) {
    event.preventDefault();
    setAppliedSearch(search.trim());
  }

  function changeFilter(action: () => void) {
    action();
  }

  return (
    <main className="w-full flex-1 space-y-6 p-4 md:p-6 lg:p-8">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1">
          <p className="text-sm font-medium text-muted-foreground">Developer</p>
          <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
            Logs
          </h1>
          <p className="max-w-2xl text-sm text-muted-foreground">
            Search structured runtime logs without exposing the provider or
            LogQL to the browser.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <Switch id="live-logs" checked={live} onCheckedChange={setLive} />
            <Label htmlFor="live-logs">Live tail</Label>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => void load()}
            disabled={loading || !service}
          >
            <RefreshCw className={cn("size-4", loading && "animate-spin")} />
            Refresh
          </Button>
        </div>
      </div>

      <Card>
        <CardContent className="pt-6">
          <form
            className="grid gap-3 lg:grid-cols-[minmax(12rem,0.8fr)_minmax(10rem,0.6fr)_minmax(9rem,0.5fr)_minmax(18rem,2fr)_auto]"
            onSubmit={applyFilters}
          >
            <Select
              value={service}
              onValueChange={(value) => changeFilter(() => setService(value))}
            >
              <SelectTrigger aria-label="Service">
                <SelectValue placeholder="Select service" />
              </SelectTrigger>
              <SelectContent>
                {services.map((item) => (
                  <SelectItem key={item.id} value={item.id}>
                    {item.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={level}
              onValueChange={(value) => changeFilter(() => setLevel(value))}
            >
              <SelectTrigger aria-label="Log level">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All levels</SelectItem>
                <SelectItem value="debug">Debug</SelectItem>
                <SelectItem value="info">Info</SelectItem>
                <SelectItem value="warn">Warning</SelectItem>
                <SelectItem value="error">Error</SelectItem>
              </SelectContent>
            </Select>
            <Select
              value={range}
              onValueChange={(value) =>
                changeFilter(() => setRange(value as TimeRange))
              }
            >
              <SelectTrigger aria-label="Time range">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="15m">Last 15 minutes</SelectItem>
                <SelectItem value="1h">Last hour</SelectItem>
                <SelectItem value="6h">Last 6 hours</SelectItem>
                <SelectItem value="24h">Last 24 hours</SelectItem>
              </SelectContent>
            </Select>
            <Input
              type="search"
              inputMode="search"
              placeholder="Search log messages…"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
            <Button type="submit" disabled={!service}>
              <Search className="size-4" />
              Search
            </Button>
          </form>
        </CardContent>
      </Card>

      {error && (
        <p
          role="alert"
          className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {error}
        </p>
      )}

      <LogViewerTable
        entries={entries}
        services={services}
        loading={loading}
        loadingMore={loadingMore}
        hasMore={Boolean(nextCursor)}
        onLoadMore={() => void loadMore()}
        onSelect={setSelected}
      />

      <Dialog
        open={Boolean(selected)}
        onOpenChange={(open) => !open && setSelected(null)}
      >
        <DialogContent className="max-h-[80vh] max-w-3xl overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Log entry</DialogTitle>
            <DialogDescription>
              {selected ? formatLogTimestamp(selected.timestamp) : ""}
            </DialogDescription>
          </DialogHeader>
          {selected && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <LogLevelBadge level={selected.level} />
                <Badge variant="outline">
                  {logServiceName(services, selected.service)}
                </Badge>
              </div>
              <pre className="whitespace-pre-wrap break-words rounded-lg border bg-muted/40 p-4 font-mono text-xs">
                {selected.message}
              </pre>
              {selected.fields && (
                <pre className="overflow-x-auto rounded-lg border bg-muted/40 p-4 font-mono text-xs">
                  {JSON.stringify(selected.fields, null, 2)}
                </pre>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </main>
  );
}

function appendUniqueLogs(current: LogEntry[], incoming: LogEntry[]) {
  const existing = new Set(
    current.map(
      (entry) =>
        `${entry.timestamp}\u0000${entry.service}\u0000${entry.message}`,
    ),
  );
  return current.concat(
    incoming.filter(
      (entry) =>
        !existing.has(
          `${entry.timestamp}\u0000${entry.service}\u0000${entry.message}`,
        ),
    ),
  );
}
