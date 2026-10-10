/**
 * Brand colors shared by charts and nav icons. The theme-neutral palette
 * mutes --color-icon-orange/blue (brown/navy), and Astryx 0.1.4 does not
 * define the data tokens, so the hex fallbacks are what renders.
 */
export const SERIES_COLORS = {
  /** Temperature — red-orange. */
  temperature: "var(--color-data-categorical-orange, #E8590C)",
  /** Humidity and devices — blue. */
  humidity: "var(--color-data-categorical-blue, #0171E3)",
} as const;

/** Climate status icon colors (overview card). */
export const STATUS_COLORS = {
  tooHot: "var(--color-data-categorical-red, #DC2626)",
  hot: SERIES_COLORS.temperature,
  cold: "var(--color-data-categorical-cyan, #0891B2)",
  humid: SERIES_COLORS.humidity,
  comfortable: "var(--color-data-categorical-green, #16A34A)",
} as const;
