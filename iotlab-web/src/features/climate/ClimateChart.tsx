import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Center } from "@astryxdesign/core/Center";
import { HStack, StackItem } from "@astryxdesign/core/Stack";
import { Icon } from "@astryxdesign/core/Icon";
import { Skeleton } from "@astryxdesign/core/Skeleton";
import { Text } from "@astryxdesign/core/Text";
import { StopIcon } from "@heroicons/react/24/solid";
import {
  Area,
  CartesianGrid,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { ChartCard } from "@/components/common/ChartCard";
import { SERIES_COLORS } from "@/config/colors";
import { formatDateTime, formatDecimal, formatTime } from "@/lib/format";
import { useIsMobile } from "@/lib/useMediaQuery";

export interface ClimatePoint {
  t: number;
  temp: number;
  hum: number;
  /** [min, max] bands — present for aggregated (bucketed) series. */
  tempBand?: [number, number];
  humBand?: [number, number];
}


interface ClimateChartProps {
  title: string;
  description: string;
  data: ClimatePoint[];
  /** Header end content (range picker, ...). */
  actions?: ReactNode;
  isLoading?: boolean;
}

const AXIS_TICK = { fontSize: 12, fill: "var(--color-text-secondary, #56606B)" };
const GRID_STROKE = "var(--color-border, #E5E8EB)";

function LegendItem({ color, label, opacity = 1 }: { color: string; label: string; opacity?: number }) {
  return (
    <HStack gap={1} vAlign="center">
      <Icon icon={StopIcon} size="xsm" style={{ color, opacity }} />
      <Text type="supporting" color="secondary">
        {label}
      </Text>
    </HStack>
  );
}

/**
 * Temperature (line, left °C axis) and humidity (area, right % axis) on one
 * chart — layout follows the Astryx dashboard-composition area chart.
 */
export function ClimateChart({ title, description, data, actions, isLoading }: ClimateChartProps) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const hasBand = data.some((p) => p.tempBand);
  const temperature = t("devices.charts.temperature");
  const humidity = t("devices.charts.humidity");

  return (
    <ChartCard
      title={title}
      description={description}
      height={isMobile ? 260 : 340}
      actions={actions}
    >
      {isLoading ? (
        <Skeleton height="100%" />
      ) : data.length === 0 ? (
        <Center height="100%">
          <Text color="secondary">{t("devices.charts.noData")}</Text>
        </Center>
      ) : (
        <>
          <StackItem size="fill">
            <ResponsiveContainer width="100%" height="100%">
              <ComposedChart data={data} margin={{ top: 10, right: 0, bottom: 0, left: 0 }}>
                <defs>
                  <linearGradient id="humidityGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor={SERIES_COLORS.humidity} stopOpacity={0.25} />
                    <stop offset="95%" stopColor={SERIES_COLORS.humidity} stopOpacity={0.03} />
                  </linearGradient>
                </defs>
                <CartesianGrid horizontal vertical={false} stroke={GRID_STROKE} />
                <XAxis
                  dataKey="t"
                  type="number"
                  scale="time"
                  domain={["dataMin", "dataMax"]}
                  tickFormatter={(v: number) => formatTime(v)}
                  tickLine={false}
                  axisLine={false}
                  tick={AXIS_TICK}
                  minTickGap={40}
                />
                <YAxis
                  yAxisId="temp"
                  domain={["auto", "auto"]}
                  tickFormatter={(v: number) => `${formatDecimal(v, 1)}°`}
                  tickLine={false}
                  axisLine={false}
                  tick={{ ...AXIS_TICK, fill: SERIES_COLORS.temperature }}
                  width={48}
                />
                <YAxis
                  yAxisId="hum"
                  orientation="right"
                  domain={["auto", "auto"]}
                  tickFormatter={(v: number) => `${formatDecimal(v, Number.isInteger(v) ? 0 : 1)}%`}
                  tickLine={false}
                  axisLine={false}
                  tick={{ ...AXIS_TICK, fill: SERIES_COLORS.humidity }}
                  width={44}
                />
                <Tooltip
                  labelFormatter={(v) => formatDateTime(Number(v))}
                  formatter={(v, _name, item) => {
                    const unit = String(item.dataKey).startsWith("temp") ? "°C" : "%";
                    return Array.isArray(v)
                      ? `${formatDecimal(Number(v[0]))} – ${formatDecimal(Number(v[1]))} ${unit}`
                      : `${formatDecimal(Number(v))} ${unit}`;
                  }}
                  contentStyle={{
                    background: "var(--color-background-popover, #fff)",
                    border: `1px solid ${GRID_STROKE}`,
                    borderRadius: "var(--radius-md, 8px)",
                    fontSize: 12,
                  }}
                  cursor={{ stroke: GRID_STROKE }}
                />
                {hasBand ? (
                  <Area
                    yAxisId="temp"
                    dataKey="tempBand"
                    name={`${temperature} ${t("devices.charts.minMax")}`}
                    stroke="none"
                    fill={SERIES_COLORS.temperature}
                    fillOpacity={0.12}
                    isAnimationActive={false}
                  />
                ) : null}
                <Area
                  yAxisId="hum"
                  type="monotone"
                  dataKey="hum"
                  name={humidity}
                  stroke={SERIES_COLORS.humidity}
                  strokeWidth={1.5}
                  fill="url(#humidityGradient)"
                  dot={false}
                  isAnimationActive={false}
                />
                <Line
                  yAxisId="temp"
                  type="monotone"
                  dataKey="temp"
                  name={temperature}
                  stroke={SERIES_COLORS.temperature}
                  strokeWidth={2}
                  dot={false}
                  isAnimationActive={false}
                />
              </ComposedChart>
            </ResponsiveContainer>
          </StackItem>
          <HStack gap={4} wrap="wrap">
            <LegendItem color={SERIES_COLORS.temperature} label={`${temperature} (°C)`} />
            <LegendItem color={SERIES_COLORS.humidity} label={`${humidity} (%)`} />
            {hasBand ? (
              <LegendItem
                color={SERIES_COLORS.temperature}
                opacity={0.3}
                label={`${temperature} ${t("devices.charts.minMax")}`}
              />
            ) : null}
          </HStack>
        </>
      )}
    </ChartCard>
  );
}
