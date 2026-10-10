import { useCallback, useEffect, useState } from "react";
import { api } from "../services/cms";
import type { Page } from "../services/contracts";

export type ContentFilters = Record<string, string>;

export default function useContentList<T>(resource: "products" | "articles") {
  const [query, setQuery] = useState({
    page: 1,
    pageSize: 20,
    filters: {} as ContentFilters,
  });
  const [revision, setRevision] = useState(0);
  const [data, setData] = useState<Page<T>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 20,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    const params = new URLSearchParams({
      ...query.filters,
      page: String(query.page),
      pageSize: String(query.pageSize),
    });
    api<Page<T>>(`${resource}?${params}`)
      .then((result) => {
        if (!active) return;
        // Deleting the last row on a page returns to the last populated page.
        const lastPage = Math.max(1, Math.ceil(result.total / query.pageSize));
        if (query.page > lastPage) {
          setQuery((current) => ({ ...current, page: lastPage }));
          return;
        }
        setData(result);
      })
      .catch((cause: Error) => {
        if (active) {
          setError(cause.message);
          setData({
            items: [],
            total: 0,
            page: query.page,
            pageSize: query.pageSize,
          });
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [resource, query, revision]);

  const refresh = useCallback(() => setRevision((value) => value + 1), []);
  const search = useCallback(
    (filters: ContentFilters) =>
      setQuery((current) => ({ ...current, page: 1, filters })),
    [],
  );
  const paginate = useCallback(
    (page: number, pageSize: number) =>
      setQuery((current) => ({
        ...current,
        page: current.pageSize === pageSize ? page : 1,
        pageSize,
      })),
    [],
  );
  return { data, loading, error, query, refresh, search, paginate };
}
