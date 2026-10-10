/** App-level identity. Change once here, reflected in TopNav, titles. */
export const APP_CONFIG = {
  name: "IoT Lab",
  shortName: "IL",
  description: "Monitoring portal for the IoT lab devices.",
  version: "0.1.0",
  /** Poll interval for live device data. */
  refreshMs: 30_000,
} as const;

/** Route paths in one place so nav config, router, and redirects never drift. */
export const ROUTES = {
  home: "/",
  devices: "/devices",
  temperature: "/temperature",
  serverError: "/500",
} as const;

/** Temperature page, optionally pre-selecting a device (`?device=`). */
export const temperaturePath = (id?: string) =>
  id ? `${ROUTES.temperature}?device=${encodeURIComponent(id)}` : ROUTES.temperature;

export const deviceDetailPath = (id: string) =>
  `${ROUTES.devices}/${encodeURIComponent(id)}`;
