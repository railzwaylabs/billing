import { LoaderCircle, SearchX } from "lucide-react";
import type { LogEntry, LogService } from "@/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

export function LogViewerTable({
  entries,
  services,
  loading,
  loadingMore,
  hasMore,
  onLoadMore,
  onSelect,
}: {
  entries: LogEntry[];
  services: LogService[];
  loading: boolean;
  loadingMore: boolean;
  hasMore: boolean;
  onLoadMore: () => void;
  onSelect: (entry: LogEntry) => void;
}) {
  return (
    <div className="overflow-hidden rounded-lg border bg-background">
      <Table
        className="min-w-[760px] table-fixed"
        containerClassName="h-[calc(100vh-25rem)] min-h-[24rem] max-h-[44rem]"
      >
        <TableHeader className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
          <TableRow>
            <TableHead className="w-52">Timestamp</TableHead>
            <TableHead className="w-24">Level</TableHead>
            <TableHead className="w-32">Service</TableHead>
            <TableHead>Message</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {!loading && entries.length === 0 ? (
            <TableRow>
              <TableCell colSpan={4} className="h-[20rem] p-0">
                <Empty className="h-full border-0">
                  <EmptyHeader>
                    <EmptyMedia variant="icon">
                      <SearchX />
                    </EmptyMedia>
                    <EmptyTitle>No logs found</EmptyTitle>
                    <EmptyDescription>
                      Try another service, time range, level, or search term.
                    </EmptyDescription>
                  </EmptyHeader>
                </Empty>
              </TableCell>
            </TableRow>
          ) : (
            entries.map((entry, index) => (
              <TableRow
                key={`${entry.timestamp}:${entry.service}:${index}`}
                className="cursor-pointer"
                onClick={() => onSelect(entry)}
              >
                <TableCell className="whitespace-nowrap font-mono text-xs text-muted-foreground">
                  {formatLogTimestamp(entry.timestamp)}
                </TableCell>
                <TableCell>
                  <LogLevelBadge level={entry.level} />
                </TableCell>
                <TableCell className="truncate">
                  {logServiceName(services, entry.service)}
                </TableCell>
                <TableCell>
                  <span className="block truncate font-mono text-xs">
                    {entry.message}
                  </span>
                </TableCell>
              </TableRow>
            ))
          )}
          {loading && (
            <TableRow>
              <TableCell colSpan={4} className="h-16 text-center">
                <span className="inline-flex items-center gap-2 text-sm text-muted-foreground">
                  <LoaderCircle className="size-4 animate-spin" />
                  Loading logs…
                </span>
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      <div className="flex min-h-14 items-center justify-center border-t bg-muted/20 px-4 py-2">
        {hasMore ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={loadingMore}
            onClick={() => onLoadMore()}
          >
            {loadingMore && <LoaderCircle className="size-4 animate-spin" />}
            {loadingMore ? "Loading…" : "Load more"}
          </Button>
        ) : (
          entries.length > 0 && (
            <span className="text-xs text-muted-foreground">
              All logs in this time range are loaded.
            </span>
          )
        )}
      </div>
    </div>
  );
}

export function LogLevelBadge({ level }: { level?: string }) {
  const value = level?.toLowerCase() || "unknown";
  return (
    <Badge
      variant="outline"
      className={cn(
        "uppercase",
        value === "error" && "border-destructive/30 text-destructive",
        value === "warn" && "border-amber-300 text-amber-700",
        value === "info" && "border-sky-300 text-sky-700",
      )}
    >
      {value}
    </Badge>
  );
}

export function formatLogTimestamp(value: string) {
  const timestamp = new Date(value);
  return Number.isNaN(timestamp.getTime())
    ? value
    : timestamp.toLocaleString(undefined, {
        dateStyle: "medium",
        timeStyle: "medium",
      });
}

export function logServiceName(services: LogService[], id: string) {
  return services.find((service) => service.id === id)?.name ?? id;
}
