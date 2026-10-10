import { apiGet } from "@/lib/api";
import type {
  Device,
  SensorReading,
  SensorStats,
  SensorStats24h,
  Telemetry,
  TelemetryRange,
  TelemetryStats,
} from "./types";

type Opts = { signal?: AbortSignal };

const enc = encodeURIComponent;

export const devicesApi = {
  list: (opts?: Opts) => apiGet<Device[]>("/devices", opts),

  sensorCurrent: (id: string, opts?: Opts) =>
    apiGet<SensorReading>(`/devices/${enc(id)}/sensor/current`, opts),

  sensorStats: (id: string, range: "30m" | "60m" | "90m", opts?: Opts) =>
    apiGet<SensorStats>(`/devices/${enc(id)}/sensor/stats?range=${range}`, opts),

  sensorStats24h: (id: string, opts?: Opts) =>
    apiGet<SensorStats24h>(`/devices/${enc(id)}/sensor/stats-24h`, opts),

  telemetryCurrent: (id: string, opts?: Opts) =>
    apiGet<Telemetry>(`/devices/${enc(id)}/telemetry/current`, opts),

  telemetryStats: (id: string, range: TelemetryRange, opts?: Opts) =>
    apiGet<TelemetryStats>(
      `/devices/${enc(id)}/telemetry/stats?range=${range}`,
      opts,
    ),
};
