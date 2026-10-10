import { useSyncExternalStore } from "react";

/** Phones: two gauges per row, smaller charts. */
export const MOBILE_QUERY = "(max-width: 640px)";
/** Phones + tablets (side nav eats ~260px): stack wide layouts, tables become lists. */
export const COMPACT_QUERY = "(max-width: 1024px)";

/** Live `window.matchMedia(query).matches`. */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const list = window.matchMedia(query);
      list.addEventListener("change", onChange);
      return () => list.removeEventListener("change", onChange);
    },
    () => window.matchMedia(query).matches,
  );
}

export const useIsMobile = () => useMediaQuery(MOBILE_QUERY);
export const useIsCompact = () => useMediaQuery(COMPACT_QUERY);
