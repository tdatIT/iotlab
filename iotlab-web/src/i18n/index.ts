import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import { en } from "./locales/en";
import { vi } from "./locales/vi";

export const SUPPORTED_LANGUAGES = ["en", "vi"] as const;
export type AppLanguage = (typeof SUPPORTED_LANGUAGES)[number];

const STORAGE_KEY = "admin-astryx.lang";

function readStoredLanguage(): AppLanguage {
  const raw = localStorage.getItem(STORAGE_KEY);
  return raw === "vi" || raw === "en" ? raw : "en";
}

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    vi: { translation: vi },
  },
  lng: readStoredLanguage(),
  fallbackLng: "en",
  interpolation: { escapeValue: false }, // React already escapes
});

i18n.on("languageChanged", (lng) => {
  localStorage.setItem(STORAGE_KEY, lng);
  document.documentElement.lang = lng;
});
document.documentElement.lang = i18n.language;

/**
 * Translate a data enum value (status, role, category, ...) for display.
 * Keys live under `enums.<group>` in the catalogs; unknown values fall back
 * to the raw value so new API values never render blank.
 */
export function enumLabel(
  group: keyof typeof en.enums,
  value: string,
): string {
  return i18n.t(`enums.${group}.${value}` as never, {
    defaultValue: value,
  }) as string;
}

export default i18n;
