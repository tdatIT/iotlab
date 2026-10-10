# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

IoT Lab monitoring frontend for the `iotlab-service` API (Swagger: `http://localhost:3000/swagger/index.html`, spec at `/swagger/doc.json`, base path `/api/v1`). React 19 + Vite + TypeScript (strict) + react-router v7, UI from the Astryx design system, charts via recharts, i18n en/vi via react-i18next. Astryx component rules and CLI workflow live in `.claude/CLAUDE.md` — read it before writing UI.

## Commands

```bash
npm run dev       # http://localhost:5173; API at VITE_API_BASE_URL (default http://localhost:3000/api/v1)
npm run build     # tsc typecheck + vite build — the only automated check
npm run preview   # serves dist/
npx astryx <cmd>  # design-system CLI: build "<idea>", component <Name>, docs <topic>
docker compose up --build   # nginx image on :8080; VITE_API_BASE_URL is a build arg
```

No test runner or linter. E2E checks are ad-hoc Playwright scripts (`*.tmp.mjs`, gitignored) using `playwright-core` with local Chrome at `C:/Program Files/Google/Chrome/Application/chrome.exe` against a running dev server.

## Architecture

- **API access** — `src/lib/api.ts` `apiGet()` unwraps the `{code, message, data}` envelope (`code === 0` is success), throws `ApiError` (with HTTP `status`), and retries 429s (the service rate-limits ~10 req/s). The browser calls the backend directly at `VITE_API_BASE_URL` (build-time; default `http://localhost:3000/api/v1`). There is no proxy anywhere, so the backend must send CORS headers. Docker passes it as a build arg; Vercel reads it as a project env var.
- **Data fetching** — `src/lib/useQuery.ts`: tiny hook (abort on key change, keeps data during polls, `refreshMs`, `isEnabled`). All device data polls every `APP_CONFIG.refreshMs`.
- **Devices feature** (`src/features/devices/`): `types.ts` mirrors the Swagger DTOs; `api.ts` one function per endpoint. There is no `GET /devices/{id}` — the detail page finds the device in the list, and only enables the other queries once it resolves. Chart ranges `30m/60m/90m` use raw `/sensor/stats`, `24h` uses bucketed `/sensor/stats-24h`.
- **Pages** — `/devices/:id` (`DeviceDetailPage`) is telemetry only: info, `HealthGauge` donuts, telemetry min/avg/max. `/temperature` (`features/climate/ClimatePage`) is sensor data only and needs no list page: a `Selector` writes `?device=` (fallback: first online device), then one `ClimateOverview` card (3 columns: temp, humidity, condition icon + label from `climateStatus.ts` priority rules — heat index/dew point are computed there but not displayed; styled after the Astryx 0.6 `dashboard-composition` template, which the local 0.1.4 CLI lacks) → `ClimateChart` (temp + humidity on dual Y axes) beside a latest-readings list → 24h table. `DeviceAnalysis` renders whichever of `stats24h` / `telemetryStats` it is given.
- **Thresholds** — `devices/signal.ts`: `healthGauges()` maps telemetry to 0–100 rings + tones (`HealthGauge` is a fixed-size recharts donut; centre label uses fixed coordinates because recharts 3 gives `Label` no polar viewBox). Climate condition thresholds: `climate/climateStatus.ts` `CLIMATE_THRESHOLDS`.
- **Side nav** — desktop rail is always collapsed (48px) and expands on hover/focus as an overlay: `AdminShell` toggles `collapsible.isCollapsed` from mouse/focus listeners on the SideNav ref, and `styles/global.css` pins `.astryx-app-shell-sidenav` to 48px with the nav absolutely positioned so content never reflows. ≤768px uses the AppShell drawer, always expanded.
- **Shell/routing** — `app/router.tsx` lazy-loads pages; `/` redirects to `/devices`; `/500` renders bare. Providers: theme → BrowserRouter → Astryx `LinkProvider` (`lib/router-link.tsx`, so Astryx `href`s navigate client-side) → toasts. Paths in `config/app.ts` `ROUTES`, side nav in `config/navigation.ts`.

## Adding things

- **Page**: route in `app/router.tsx` + path in `ROUTES` + entry in `config/navigation.ts` + folder under `features/`.
- **UI copy**: keys in `i18n/locales/en.ts` (source of truth). `vi.ts` is typed `Messages<typeof en>`, so a missing key fails `npm run build`. `t("...")` keys are compile-checked. Data enums render via `enumLabel("<group>", value)`.
- **Formatting**: `lib/format.ts` (locale-aware numbers/dates, `formatDuration`, `formatBytes`).

## Styling

Astryx components and props first; tokens (`var(--color-*|--spacing-*|--radius-*)`) for custom values. No Tailwind (removed); `src/styles/global.css` only sets the body font. Series colors live in `config/colors.ts` `SERIES_COLORS` (temperature red-orange, humidity/devices blue): `--color-data-categorical-*` names with hex fallbacks (those tokens are not defined in Astryx 0.1.4, so the fallbacks render; theme-neutral mutes `--color-icon-*`, so they are not used). Charts use them, and so do the selected side-nav icons via `tinted()` in `config/navigation.ts`. Responsive switches use `lib/useMediaQuery.ts`: `useIsMobile()` ≤640px, `useIsCompact()` ≤1024px (the side nav takes ~260px).

Import alias `@/` → `src/`. Astryx imports are per-module: `@astryxdesign/core/<Component>`.
