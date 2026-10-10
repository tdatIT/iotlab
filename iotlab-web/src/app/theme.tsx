import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { Theme } from "@astryxdesign/core";
import { neutralTheme } from "@astryxdesign/theme-neutral/built";

export type ThemeMode = "system" | "light" | "dark";

interface ThemeModeContextValue {
  mode: ThemeMode;
  /** The mode actually in effect (resolves "system" to light/dark). */
  resolvedMode: "light" | "dark";
  setMode: (mode: ThemeMode) => void;
  toggle: () => void;
}

const STORAGE_KEY = "admin-astryx.theme-mode";

const ThemeModeContext = createContext<ThemeModeContextValue | null>(null);

function readStoredMode(): ThemeMode {
  const raw = localStorage.getItem(STORAGE_KEY);
  return raw === "light" || raw === "dark" || raw === "system"
    ? raw
    : "system";
}

function useSystemPrefersDark(): boolean {
  const [prefersDark, setPrefersDark] = useState(
    () => window.matchMedia("(prefers-color-scheme: dark)").matches,
  );
  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = (e: MediaQueryListEvent) => setPrefersDark(e.matches);
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);
  return prefersDark;
}

/**
 * Wraps the Astryx <Theme> provider with a persisted light/dark/system mode.
 * Swap `neutralTheme` for a custom theme from `npx astryx theme` in one place.
 */
export function AppThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>(readStoredMode);
  const systemPrefersDark = useSystemPrefersDark();

  const resolvedMode =
    mode === "system" ? (systemPrefersDark ? "dark" : "light") : mode;

  const setMode = useCallback((next: ThemeMode) => {
    localStorage.setItem(STORAGE_KEY, next);
    setModeState(next);
  }, []);

  const toggle = useCallback(() => {
    setMode(resolvedMode === "dark" ? "light" : "dark");
  }, [resolvedMode, setMode]);

  const value = useMemo(
    () => ({ mode, resolvedMode, setMode, toggle }),
    [mode, resolvedMode, setMode, toggle],
  );

  return (
    <ThemeModeContext.Provider value={value}>
      <Theme theme={neutralTheme} mode={mode}>
        {children}
      </Theme>
    </ThemeModeContext.Provider>
  );
}

export function useThemeMode(): ThemeModeContextValue {
  const ctx = useContext(ThemeModeContext);
  if (!ctx) {
    throw new Error("useThemeMode must be used inside <AppThemeProvider>");
  }
  return ctx;
}
