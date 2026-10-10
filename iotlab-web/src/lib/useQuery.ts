import { useCallback, useEffect, useRef, useState } from "react";

export interface QueryState<T> {
  data: T | undefined;
  error: Error | undefined;
  /** True only until the first response (success or error) arrives. */
  isLoading: boolean;
  /** True while any request — first load, poll or manual refetch — is in flight. */
  isFetching: boolean;
  refetch: () => void;
}

interface QueryOptions {
  /** Re-run the fetcher every N ms while mounted. */
  refreshMs?: number;
  /** Skip fetching (e.g. while a dependency is missing). */
  isEnabled?: boolean;
}

/**
 * Minimal data-fetching hook: aborts stale requests, keeps the last data
 * while refetching (no flicker on polls), and re-runs whenever `key` changes.
 * Swap for TanStack Query if caching across pages becomes necessary.
 */
export function useQuery<T>(
  key: string,
  fetcher: (signal: AbortSignal) => Promise<T>,
  { refreshMs, isEnabled = true }: QueryOptions = {},
): QueryState<T> {
  const [data, setData] = useState<T>();
  const [error, setError] = useState<Error>();
  const [isFetching, setIsFetching] = useState(false);
  const [hasSettled, setHasSettled] = useState(false);
  const [tick, setTick] = useState(0);

  // Latest fetcher without making it an effect dependency (callers pass inline lambdas).
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  // Reset when the query identity changes so old data never shows under a new key.
  const [prevKey, setPrevKey] = useState(key);
  if (prevKey !== key) {
    setPrevKey(key);
    setData(undefined);
    setError(undefined);
    setHasSettled(false);
  }

  useEffect(() => {
    if (!isEnabled) return;
    const controller = new AbortController();
    setIsFetching(true);
    fetcherRef
      .current(controller.signal)
      .then((result) => {
        setData(result);
        setError(undefined);
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return;
        setError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        if (controller.signal.aborted) return;
        setIsFetching(false);
        setHasSettled(true);
      });
    return () => controller.abort();
  }, [key, tick, isEnabled]);

  useEffect(() => {
    if (!refreshMs || !isEnabled) return;
    const id = setInterval(() => setTick((n) => n + 1), refreshMs);
    return () => clearInterval(id);
  }, [refreshMs, isEnabled, key]);

  const refetch = useCallback(() => setTick((n) => n + 1), []);

  return {
    data,
    error,
    isLoading: isEnabled && !hasSettled,
    isFetching,
    refetch,
  };
}
