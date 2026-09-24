import { useEffect, useState } from "react";
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  useReactTable,
  type ColumnDef,
} from "@tanstack/react-table";
import type { PageInfo } from "@/api";
import { Input } from "@/components/ui/input";
import { SearchX } from "lucide-react";
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
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";

export type CursorPagination = {
  cursor?: string;
  pageInfo: PageInfo;
  onCursorChange: (cursor?: string) => void;
  loading?: boolean;
  resetKey?: string;
};

export function DataTable<TData, TValue>({
  columns,
  data,
  searchKey,
  searchPlaceholder = "Search…",
  onRowClick,
  cursorPagination,
}: {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  searchKey?: string;
  searchPlaceholder?: string;
  onRowClick?: (row: TData) => void;
  cursorPagination?: CursorPagination;
}) {
  const [globalFilter, setGlobalFilter] = useState("");
  const [cursorHistory, setCursorHistory] = useState<(string | undefined)[]>(
    [],
  );
  useEffect(() => {
    setCursorHistory([]);
  }, [cursorPagination?.resetKey]);
  const table = useReactTable({
    data,
    columns,
    state: { globalFilter },
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: cursorPagination
      ? undefined
      : getPaginationRowModel(),
  });
  const previousCursor = () => {
    if (!cursorPagination || cursorHistory.length === 0) return;
    const history = [...cursorHistory];
    const cursor = history.pop();
    setCursorHistory(history);
    cursorPagination.onCursorChange(cursor);
  };
  const nextCursor = () => {
    const next = cursorPagination?.pageInfo.next_cursor;
    if (!cursorPagination || !next) return;
    setCursorHistory([...cursorHistory, cursorPagination.cursor]);
    cursorPagination.onCursorChange(next);
  };
  const cursorDisabled = cursorPagination?.loading === true;
  return (
    <div className="w-full space-y-4">
      {searchKey && (
        <div className="flex items-center justify-between gap-3">
          <Input
            type="search"
            className="w-full max-w-sm bg-background"
            placeholder={searchPlaceholder}
            value={
              (table.getColumn(searchKey)?.getFilterValue() as string) ?? ""
            }
            onChange={(event) =>
              table.getColumn(searchKey)?.setFilterValue(event.target.value)
            }
          />
        </div>
      )}
      <div className="overflow-hidden rounded-lg border bg-background">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((group) => (
              <TableRow key={group.id}>
                {group.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext(),
                        )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  className={onRowClick ? "cursor-pointer" : undefined}
                  onClick={() => onRowClick?.(row.original)}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext(),
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="p-0">
                  <Empty className="min-h-40 border-0 py-8">
                    <EmptyHeader>
                      <EmptyMedia variant="icon">
                        <SearchX />
                      </EmptyMedia>
                      <EmptyTitle>
                        {globalFilter ? "No matching results" : "No data yet"}
                      </EmptyTitle>
                      <EmptyDescription>
                        {globalFilter
                          ? "Try changing or clearing the current filter."
                          : "Resources will appear here after they are created."}
                      </EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      {cursorPagination ? (
        <Pagination className="justify-end">
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                href="#"
                aria-disabled={cursorHistory.length === 0 || cursorDisabled}
                tabIndex={cursorHistory.length === 0 || cursorDisabled ? -1 : 0}
                className={
                  cursorHistory.length === 0 || cursorDisabled
                    ? "pointer-events-none opacity-50"
                    : undefined
                }
                onClick={(event) => {
                  event.preventDefault();
                  previousCursor();
                }}
              />
            </PaginationItem>
            <PaginationItem>
              <span className="flex h-9 items-center px-3 text-sm text-muted-foreground">
                Page {cursorHistory.length + 1}
              </span>
            </PaginationItem>
            <PaginationItem>
              <PaginationNext
                href="#"
                aria-disabled={
                  !cursorPagination.pageInfo.has_more ||
                  !cursorPagination.pageInfo.next_cursor ||
                  cursorDisabled
                }
                tabIndex={
                  !cursorPagination.pageInfo.has_more ||
                  !cursorPagination.pageInfo.next_cursor ||
                  cursorDisabled
                    ? -1
                    : 0
                }
                className={
                  !cursorPagination.pageInfo.has_more ||
                  !cursorPagination.pageInfo.next_cursor ||
                  cursorDisabled
                    ? "pointer-events-none opacity-50"
                    : undefined
                }
                onClick={(event) => {
                  event.preventDefault();
                  nextCursor();
                }}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      ) : table.getPageCount() > 1 ? (
        <Pagination className="justify-end">
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                href="#"
                aria-disabled={!table.getCanPreviousPage()}
                tabIndex={table.getCanPreviousPage() ? 0 : -1}
                className={
                  table.getCanPreviousPage()
                    ? undefined
                    : "pointer-events-none opacity-50"
                }
                onClick={(event) => {
                  event.preventDefault();
                  table.previousPage();
                }}
              />
            </PaginationItem>
            <PaginationItem>
              <span className="flex h-9 items-center px-3 text-sm text-muted-foreground">
                Page {table.getState().pagination.pageIndex + 1}
              </span>
            </PaginationItem>
            <PaginationItem>
              <PaginationNext
                href="#"
                aria-disabled={!table.getCanNextPage()}
                tabIndex={table.getCanNextPage() ? 0 : -1}
                className={
                  table.getCanNextPage()
                    ? undefined
                    : "pointer-events-none opacity-50"
                }
                onClick={(event) => {
                  event.preventDefault();
                  table.nextPage();
                }}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      ) : null}
    </div>
  );
}
