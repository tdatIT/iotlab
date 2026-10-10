/**
 * Shared formatters — keep display logic out of components.
 * Locale follows the active i18n language (en → en-US, vi → vi-VN).
 * Formatter instances are cached per locale (Intl construction is costly).
 */
import i18n from "@/i18n";

const INTL_LOCALES: Record<string, string> = {
  en: "en-US",
  vi: "vi-VN",
};

function currentLocale(): string {
  return INTL_LOCALES[i18n.language] ?? "en-US";
}

const cache = new Map<string, Intl.NumberFormat | Intl.DateTimeFormat>();

function cached<T extends Intl.NumberFormat | Intl.DateTimeFormat>(
  key: string,
  create: () => T,
): T {
  let formatter = cache.get(key);
  if (!formatter) {
    formatter = create();
    cache.set(key, formatter);
  }
  return formatter as T;
}

export function formatNumber(value: number): string {
  const locale = currentLocale();
  return cached(`num:${locale}`, () => new Intl.NumberFormat(locale)).format(
    value,
  );
}

/** Fixed-precision number in the active locale, e.g. 31.0 / 31,0. */
export function formatDecimal(value: number, digits = 1): string {
  const locale = currentLocale();
  return cached(
    `dec${digits}:${locale}`,
    () =>
      new Intl.NumberFormat(locale, {
        minimumFractionDigits: digits,
        maximumFractionDigits: digits,
      }),
  ).format(value);
}

export function formatTime(value: string | number | Date): string {
  const locale = currentLocale();
  return cached(
    `time:${locale}`,
    () => new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit" }),
  ).format(new Date(value));
}

export function formatDateTime(value: string | number | Date): string {
  const locale = currentLocale();
  return cached(
    `datetime:${locale}`,
    () =>
      new Intl.DateTimeFormat(locale, {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      }),
  ).format(new Date(value));
}

/** Localized "5 minutes ago" / "5 phút trước", relative to now. */
export function formatRelative(value: string | number | Date): string {
  const locale = currentLocale();
  const seconds = Math.round((new Date(value).getTime() - Date.now()) / 1000);
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
  const abs = Math.abs(seconds);
  if (abs < 60) return rtf.format(seconds, "second");
  if (abs < 3600) return rtf.format(Math.round(seconds / 60), "minute");
  if (abs < 86_400) return rtf.format(Math.round(seconds / 3600), "hour");
  return rtf.format(Math.round(seconds / 86_400), "day");
}

/** Compact duration: 3905 → "1h 5m", 93784 → "1d 2h 3m". */
export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const d = Math.floor(s / 86_400);
  const h = Math.floor((s % 86_400) / 3_600);
  const m = Math.floor((s % 3_600) / 60);
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return s % 60 ? `${m}m ${s % 60}s` : `${m}m`;
  return `${s}s`;
}

/** Bytes in KiB/MiB with one decimal, e.g. 226192 → "220.9 KB". */
export function formatBytes(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${formatDecimal(bytes / 1024 / 1024)} MB`;
  return `${formatDecimal(bytes / 1024)} KB`;
}

