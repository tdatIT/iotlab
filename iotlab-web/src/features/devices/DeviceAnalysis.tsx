import { useTranslation } from "react-i18next";
import { Card } from "@astryxdesign/core/Card";
import { List, ListItem } from "@astryxdesign/core/List";
import { Skeleton } from "@astryxdesign/core/Skeleton";
import { VStack } from "@astryxdesign/core/Stack";
import { Heading, Text } from "@astryxdesign/core/Text";
import { Table, pixel, proportional, type TableColumn } from "@astryxdesign/core/Table";

import { formatBytes, formatDecimal, formatNumber } from "@/lib/format";
import { useIsCompact } from "@/lib/useMediaQuery";
import type { SensorStats24h, Stat, TelemetryStats } from "./types";

interface DeviceAnalysisProps {
  stats24h?: SensorStats24h;
  telemetryStats?: TelemetryStats;
  isLoading: boolean;
}

interface SummaryRow extends Record<string, unknown> {
  id: string;
  metric: string;
  window: string;
  stat: Stat;
  samples: number;
  format: (v: number) => string;
}

export function DeviceAnalysis({
  stats24h,
  telemetryStats,
  isLoading,
}: DeviceAnalysisProps) {
  const { t } = useTranslation();
  const isCompact = useIsCompact();

  const rows: SummaryRow[] = [];
  if (stats24h && stats24h.summary.samples > 0) {
    const { summary } = stats24h;
    rows.push(
      {
        id: "temp",
        metric: t("devices.analysis.metrics.temperature"),
        window: "24h",
        stat: summary.temp_c,
        samples: summary.samples,
        format: (v) => formatDecimal(v),
      },
      {
        id: "hum",
        metric: t("devices.analysis.metrics.humidity"),
        window: "24h",
        stat: summary.humidity_pct,
        samples: summary.samples,
        format: (v) => formatDecimal(v),
      },
    );
  }
  if (telemetryStats && telemetryStats.summary.samples > 0) {
    const { summary } = telemetryStats;
    rows.push(
      {
        id: "rssi",
        metric: t("devices.analysis.metrics.rssi"),
        window: "1h",
        stat: summary.rssi_dbm,
        samples: summary.samples,
        format: (v) => formatDecimal(v),
      },
      {
        id: "heapFree",
        metric: t("devices.analysis.metrics.heapFree"),
        window: "1h",
        stat: summary.heap_free,
        samples: summary.samples,
        format: formatBytes,
      },
      {
        id: "heapMin",
        metric: t("devices.analysis.metrics.heapMin"),
        window: "1h",
        stat: summary.heap_min,
        samples: summary.samples,
        format: formatBytes,
      },
    );
  }

  const columns: TableColumn<SummaryRow>[] = [
    { key: "metric", header: t("devices.analysis.columns.metric"), width: proportional(1) },
    { key: "window", header: t("devices.analysis.columns.window"), width: pixel(80) },
    {
      key: "min",
      header: t("devices.analysis.columns.min"),
      width: pixel(110),
      align: "end",
      renderCell: (r) => <Text hasTabularNumbers>{r.format(r.stat.min)}</Text>,
    },
    {
      key: "avg",
      header: t("devices.analysis.columns.avg"),
      width: pixel(110),
      align: "end",
      renderCell: (r) => (
        <Text hasTabularNumbers weight="semibold">
          {r.format(r.stat.avg)}
        </Text>
      ),
    },
    {
      key: "max",
      header: t("devices.analysis.columns.max"),
      width: pixel(110),
      align: "end",
      renderCell: (r) => <Text hasTabularNumbers>{r.format(r.stat.max)}</Text>,
    },
    {
      key: "samples",
      header: t("devices.analysis.columns.samples"),
      width: pixel(90),
      align: "end",
      renderCell: (r) => <Text hasTabularNumbers>{formatNumber(r.samples)}</Text>,
    },
  ];

  return (
    <Card padding={4}>
      <VStack gap={3}>
        <Heading level={4}>{t("devices.analysis.title")}</Heading>
        {isLoading ? (
          <VStack gap={2}>
            {[0, 1, 2, 3, 4].map((i) => (
              <Skeleton key={i} height={28} index={i} />
            ))}
          </VStack>
        ) : isCompact ? (
          // Six columns don't fit narrow screens: one row per metric, stats underneath.
          <List density="compact" hasDividers>
            {rows.map((r) => (
              <ListItem
                key={r.id}
                label={`${r.metric} · ${r.window}`}
                description={`${t("devices.analysis.columns.min")}–${t("devices.analysis.columns.max")}: ${r.format(r.stat.min)} – ${r.format(r.stat.max)}`}
                endContent={
                  <VStack gap={0} hAlign="end">
                    <Text hasTabularNumbers weight="semibold">
                      {r.format(r.stat.avg)}
                    </Text>
                    <Text type="supporting" color="secondary">
                      {t("devices.analysis.columns.avg")}
                    </Text>
                  </VStack>
                }
              />
            ))}
          </List>
        ) : (
          <Table data={rows} columns={columns} idKey="id" density="compact" />
        )}
      </VStack>
    </Card>
  );
}
