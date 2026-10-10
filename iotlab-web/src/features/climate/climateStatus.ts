/**
 * Current indoor condition from temperature + relative humidity.
 * Rules are checked in priority order; the first match wins:
 *   1. tooHot       T ≥ 32 °C or heat index ≥ 41 °C
 *   2. hot          T > T_max
 *   3. cold         T < T_min
 *   4. humid        RH ≥ 80 % or dew point ≥ 24 °C
 *   5. comfortable  everything else
 */
export const CLIMATE_THRESHOLDS = {
  tMin: 18,
  tMax: 28,
  tooHotTemp: 32,
  tooHotHeatIndex: 41,
  humidRh: 80,
  humidDewPoint: 24,
};

export type ClimateStatus = "tooHot" | "hot" | "cold" | "humid" | "comfortable";

/** Heat index (°C) — NWS Rothfusz regression with its low/high-humidity adjustments. */
export function heatIndex(tempC: number, rh: number): number {
  const t = (tempC * 9) / 5 + 32;
  const simple = 0.5 * (t + 61 + (t - 68) * 1.2 + rh * 0.094);
  let hi = simple;
  if ((simple + t) / 2 >= 80) {
    hi =
      -42.379 +
      2.04901523 * t +
      10.14333127 * rh -
      0.22475541 * t * rh -
      0.00683783 * t * t -
      0.05481717 * rh * rh +
      0.00122874 * t * t * rh +
      0.00085282 * t * rh * rh -
      0.00000199 * t * t * rh * rh;
    if (rh < 13 && t >= 80 && t <= 112) {
      hi -= ((13 - rh) / 4) * Math.sqrt((17 - Math.abs(t - 95)) / 17);
    } else if (rh > 85 && t >= 80 && t <= 87) {
      hi += ((rh - 85) / 10) * ((87 - t) / 5);
    }
  }
  return ((hi - 32) * 5) / 9;
}

/** Dew point (°C) — Magnus formula. */
export function dewPoint(tempC: number, rh: number): number {
  const a = 17.62;
  const b = 243.12;
  const gamma = Math.log(Math.max(rh, 1) / 100) + (a * tempC) / (b + tempC);
  return (b * gamma) / (a - gamma);
}

export function climateStatus(tempC: number, rh: number) {
  const hi = heatIndex(tempC, rh);
  const td = dewPoint(tempC, rh);
  const th = CLIMATE_THRESHOLDS;
  let status: ClimateStatus = "comfortable";
  if (tempC >= th.tooHotTemp || hi >= th.tooHotHeatIndex) status = "tooHot";
  else if (tempC > th.tMax) status = "hot";
  else if (tempC < th.tMin) status = "cold";
  else if (rh >= th.humidRh || td >= th.humidDewPoint) status = "humid";
  return { status, heatIndex: hi, dewPoint: td };
}
