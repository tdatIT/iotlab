import type { ReactNode } from "react";
import { BrowserRouter } from "react-router-dom";
import { LinkProvider } from "@astryxdesign/core/Link";
import { ToastViewport } from "@astryxdesign/core/Toast";

import { RouterLink } from "@/lib/router-link";
import { AppThemeProvider } from "@/app/theme";

/**
 * Global providers, ordered outside-in:
 * Theme (light/dark) → Router → LinkProvider (Astryx href → client-side nav)
 * → Toasts. Add new app-wide providers here (query client, i18n, ...).
 */
export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <AppThemeProvider>
      <BrowserRouter>
        <LinkProvider component={RouterLink}>
          <ToastViewport position="bottomEnd" maxVisible={3}>
            {children}
          </ToastViewport>
        </LinkProvider>
      </BrowserRouter>
    </AppThemeProvider>
  );
}
