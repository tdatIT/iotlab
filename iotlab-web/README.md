# IoT Lab

Monitoring frontend for the **iotlab-service** API, built on the [Astryx design system](https://www.npmjs.com/package/@astryxdesign/core), React 19, Vite, TypeScript, react-router, recharts and react-i18next (en/vi). Light/dark mode included.

## Pages

- **`/devices`** — every registered device: status, IP, firmware, last seen, with search and online/offline counts. Click a row for its telemetry.
- **`/devices/:id`** — one device's telemetry: header with status and last seen, device info, four donut gauges (uptime as share of 24h, free memory with ring = lowest watermark / current free, connection with RSSI mapped −100…−30 dBm, sensor errors with ring = share of successful reads) and a 1h min/avg/max table for RSSI and heap.
- **`/temperature`** — opens directly, no device list. A selector in the header picks the device (`?device=<id>`; defaults to the first online device). Shows one overview card in three columns: temperature, humidity (each with change vs the 24h average) and a condition icon + label (Too hot / Hot / Cold / Humid / Comfortable, from the current reading; styled after the Astryx `dashboard-composition` template), one combined chart over 30m / 1h / 90m / 24h with the latest readings beside it, and a 24h min/avg/max table.

Data refreshes every 30 s (`APP_CONFIG.refreshMs`).

## API

The backend is `iotlab-service`; its Swagger docs are at `http://localhost:3000/swagger/index.html`. The browser calls it **directly**, with no proxy, at `VITE_API_BASE_URL` (default `http://localhost:3000/api/v1`). Vite bakes this value in at build time, so set it before `npm run build` / `docker build`. The backend must allow the app's origin via CORS (it currently sends `Access-Control-Allow-Origin: *`).

```bash
cp .env.example .env   # edit VITE_API_BASE_URL if the API lives elsewhere
```

## Run locally

```bash
npm install
npm run dev      # http://localhost:5173
npm run build    # typecheck + production build
```

## Docker

```bash
docker compose up --build                                            # http://localhost:8080
VITE_API_BASE_URL=https://iotlab.example.com/api/v1 docker compose up --build
# or
docker build --build-arg VITE_API_BASE_URL=https://iotlab.example.com/api/v1 -t iotlab-web .
docker run -p 8080:80 iotlab-web
```

The image is built in two stages (Node build → `nginx:alpine`). nginx only serves the static files: an SPA fallback, long caching for `/assets/` and a `/healthz` endpoint. The URL must be reachable from the user's browser, not from inside the container.

## Vercel

1. Import the repo. `vercel.json` sets up the Vite build and the SPA fallback.
2. Under Project → Settings → Environment Variables, set `VITE_API_BASE_URL` to the public API URL, for example `https://iotlab.example.com/api/v1`. `localhost` will not work.
3. Deploy with `vercel --prod` or a git push.

## Structure

```
src/
  app/            providers, theme, router (lazy routes), ErrorBoundary
  config/         app.ts (name, routes, refresh interval), navigation.ts (side nav)
  lib/            api.ts (envelope + 429 retry), useQuery.ts (fetch/poll hook),
                  useMediaQuery.ts (mobile/compact breakpoints), format.ts
  features/
    devices/      DevicesPage, DeviceDetailPage (telemetry), DeviceAnalysis, HealthGauge,
                  signal.ts (gauge math + thresholds), api.ts, types.ts (DTOs)
    climate/      ClimatePage (/temperature), ClimateOverview, ClimateChart
    misc/         404 + 500 pages
  components/     AdminShell (AppShell + TopNav + SideNav), Page, PageHeader,
                  StatCard, ChartCard, DataTable, TableToolbar, StatusLabel
  i18n/           en.ts (source of truth), vi.ts (type-enforced mirror)
docker/           nginx.conf (static files + SPA fallback)
```

## Conventions

- **Astryx first.** Use Astryx components for layout (`VStack`, `HStack`, `Grid`, `Card`), not `<div>`s. To look up a component, run `npx astryx component <Name>`.
- **Tokens for values.** Write colors, spacing and radii as `var(--color-*|--spacing-*|--radius-*)`.
- **Responsive.** `useIsMobile()` (≤640px: two gauges per row, smaller charts) and `useIsCompact()` (≤1024px: stacked chart/list, tables become lists) from `lib/useMediaQuery.ts`.
- **i18n.** Add every UI string to `locales/en.ts` and `locales/vi.ts`. `npm run build` fails when a Vietnamese key is missing.
- **Thresholds**: RSSI quality and heap watermark live in `features/devices/signal.ts`. The climate condition rules are in `features/climate/climateStatus.ts`; checked in priority order:
  1. **Too hot:** T ≥ 32 °C or heat index ≥ 41 °C
  2. **Hot:** T > 28 °C
  3. **Cold:** T < 18 °C
  4. **Humid:** RH ≥ 80 % or dew point ≥ 24 °C
  5. **Comfortable:** everything else
