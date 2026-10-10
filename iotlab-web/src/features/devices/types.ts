/** DTOs of the iotlab-service API (see http://localhost:3000/swagger/index.html). */

export type DeviceStatus = "online" | "offline" | (string & {});

export interface Device {
  device_id: string;
  name: string;
  location: string | null;
  status: DeviceStatus;
  status_changed_at: string;
  last_seen: string;
  first_seen: string;
  ip: string;
  fw: string;
  metadata: Record<string, unknown>;
}

export interface SensorReading {
  time: string;
  temp_c: number;
  humidity_pct: number;
}

export interface SensorBucket {
  bucket: string;
  avg_c: number;
  min_c: number;
  max_c: number;
  avg_h: number;
  min_h: number;
  max_h: number;
  samples: number;
}

export interface Stat {
  min: number;
  max: number;
  avg: number;
}

export interface SensorSummary {
  temp_c: Stat;
  humidity_pct: Stat;
  samples: number;
}

export interface SensorStats {
  range: string;
  summary: SensorSummary;
  points: SensorReading[] | null;
}

export interface SensorStats24h {
  range: string;
  summary: SensorSummary;
  points: SensorBucket[] | null;
}

export interface Telemetry {
  time: string;
  fw: string;
  uptime_s: number;
  reset_reason: string;
  rssi_dbm: number;
  ip: string;
  heap_free: number;
  heap_min: number;
  wifi_reconnects: number;
  mqtt_reconnects: number;
  sensor_errors: number;
}

export interface TelemetryStats {
  range: string;
  summary: {
    rssi_dbm: Stat;
    heap_free: Stat;
    heap_min: Stat;
    samples: number;
  };
  points: Telemetry[] | null;
}

export const SENSOR_RANGES = ["30m", "60m", "90m", "24h"] as const;
export type SensorRange = (typeof SENSOR_RANGES)[number];

export type TelemetryRange = "15m" | "30m" | "60m";
