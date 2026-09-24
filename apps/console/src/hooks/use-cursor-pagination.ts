import { useCallback, useMemo, useState } from "react";
import type { PageInfo, PageParams } from "@/api";

const DEFAULT_PAGE_SIZE = 20;

export function useCursorPagination(pageSize = DEFAULT_PAGE_SIZE) {
  const [cursor, setCursor] = useState<string>();
  const [pageInfo, setPageInfo] = useState<PageInfo>({ has_more: false });

  const reset = useCallback(() => {
    setCursor(undefined);
    setPageInfo({ has_more: false });
  }, []);

  const request = useMemo<PageParams>(
    () => ({ limit: pageSize, cursor }),
    [cursor, pageSize],
  );

  return {
    cursor,
    pageInfo,
    request,
    setCursor,
    setPageInfo,
    reset,
  };
}
