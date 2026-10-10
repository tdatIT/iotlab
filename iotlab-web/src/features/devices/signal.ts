/**
 * Health gauge math — maps raw telemetry onto 0–100 rings.
 * Thresholds are lab defaults for an indoor ESP32 + DHT22 node; tune here.
 */
import type { GaugeTone } from "./HealthGauge";
import type { SensorReading, Telemetry } from "./types";

const RSSI = { excellent: -55, good: -67, fair: -75, floor: -100, ceiling: -30 };
const HEAP = { minWarn: 40 * 1024, minError: 20 * 1024 };
/** Fallback when the sampling interval can't be inferred from readings. */
const DEFAULT_SENSOR_INTERVAL_S = 15;

export type SignalQuality = "excellent" | "good" | "fair" | "weak";

export function signalQuality(rssi: number): SignalQuality {
  if (rssi >= RSSI.excellent) return "excellent";
  if (rssi >= RSSI.good) return "good";
  if (rssi >= RSSI.fair) return "fair";
  return "weak";
}

const QUALITY_TONE: Record<SignalQuality, GaugeTone> = {
  excellent: "success",
  good: "success",
  fair: "warning",
  weak: "error",
};

function clampPct(v: number): number {
  return Math.max(0, Math.min(100, v));
}

function medianIntervalS(points: SensorReading[] | undefined): number {
  if (!points || points.length < 3) return DEFAULT_SENSOR_INTERVAL_S;
  const t = points.map((p) => new Date(p.time).getTime());
  const gaps = t.slice(1).map((v, i) => (v - t[i]) / 1000).sort((a, b) => a - b);
  return gaps[Math.floor(gaps.length / 2)] || DEFAULT_SENSOR_INTERVAL_S;
}

export interface HealthGauges {
  /** Uptime as a share of 24h (capped). */
  uptimePct: number;
  /** Lowest heap watermark relative to current free heap — how deep it has dipped. */
  memoryPct: number;
  memoryTone: GaugeTone;
  /** RSSI mapped from -100 dBm (0%) to -30 dBm (100%). */
  signalPct: number;
  signalTone: GaugeTone;
  /** Share of sensor reads since boot that succeeded. */
  readOkPct: number;
  errorTone: GaugeTone;
}

export function healthGauges(
  tel: Telemetry | undefined,
  rawPoints: SensorReading[] | undefined,
): HealthGauges {
  if (!tel) {
    return {
      uptimePct: 0,
      memoryPct: 0,
      memoryTone: "neutral",
      signalPct: 0,
      signalTone: "neutral",
      readOkPct: 0,
      errorTone: "neutral",
    };
  }

  const reads = Math.max(1, Math.round(tel.uptime_s / medianIntervalS(rawPoints)));
  const readOkPct = clampPct(((reads - tel.sensor_errors) / reads) * 100);

  return {
    uptimePct: clampPct((tel.uptime_s / 86_400) * 100),
    memoryPct: clampPct(tel.heap_free > 0 ? (tel.heap_min / tel.heap_free) * 100 : 0),
    memoryTone:
      tel.heap_min < HEAP.minError
        ? "error"
        : tel.heap_min < HEAP.minWarn
          ? "warning"
          : "success",
    signalPct: clampPct(
      ((tel.rssi_dbm - RSSI.floor) / (RSSI.ceiling - RSSI.floor)) * 100,
    ),
    signalTone: QUALITY_TONE[signalQuality(tel.rssi_dbm)],
    readOkPct,
    errorTone:
      tel.sensor_errors === 0 ? "success" : readOkPct >= 99 ? "warning" : "error",
  };
}
