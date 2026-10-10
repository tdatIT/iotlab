import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Banner } from "@astryxdesign/core/Banner";
import { Button } from "@astryxdesign/core/Button";
import { IconButton } from "@astryxdesign/core/IconButton";
import { Card } from "@astryxdesign/core/Card";
import { Center } from "@astryxdesign/core/Center";
import { EmptyState } from "@astryxdesign/core/EmptyState";
import { Grid, GridSpan } from "@astryxdesign/core/Grid";
import { Icon } from "@astryxdesign/core/Icon";
import { List, ListItem } from "@astryxdesign/core/List";
import {
  SegmentedControl,
  SegmentedControlItem,
} from "@astryxdesign/core/SegmentedControl";
import { Selector } from "@astryxdesign/core/Selector";
import { Skeleton } from "@astryxdesign/core/Skeleton";
import { VStack } from "@astryxdesign/core/Stack";
import { Heading, Text } from "@astryxdesign/core/Text";
import { ArrowPathIcon, CpuChipIcon } from "@heroicons/react/24/outline";

import { Page } from "@/components/layout/Page";
import { PageHeader } from "@/components/layout/PageHeader";
import { APP_CONFIG } from "@/config/app";
import { SERIES_COLORS } from "@/config/colors";
import { enumLabel } from "@/i18n";
import { ApiError } from "@/lib/api";
import { formatDecimal, formatTime } from "@/lib/format";
import { useIsCompact } from "@/lib/useMediaQuery";
import { useQuery } from "@/lib/useQuery";
import { devicesApi } from "@/features/devices/api";
import { DeviceAnalysis } from "@/features/devices/DeviceAnalysis";
import { DevicePresence } from "@/features/devices/DeviceStatusLabel";
import { SENSOR_RANGES, type Device, type SensorRange } from "@/features/devices/types";
import { ClimateChart, type ClimatePoint } from "./ClimateChart";
import { ClimateOverview } from "./ClimateOverview";
import { climateStatus } from "./climateStatus";

/** Rows in the "latest readings" list next to the chart. */
const LATEST_COUNT = 6;

const REFRESH = { refreshMs: APP_CONFIG.refreshMs };

function isNotFound(error: Error | undefined) {
  return error instanceof ApiError && error.status === 404;
}

/** `?device=` if it exists, else the first online device, else the first one. */
function pickDevice(devices: Device[] | undefined, requested: string | null) {
  if (!devices?.length) return undefined;
  return (
    devices.find((d) => d.device_id === requested) ??
    devices.find((d) => d.status === "online") ??
    devices[0]
  );
}

/** Temperature & humidity for one device, picked from a small selector. */
export function ClimatePage() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const [range, setRange] = useState<SensorRange>("60m");
  // Chart and latest-readings list sit side by side only on wide screens.
  const isStacked = useIsCompact();
  // 24h charts use buckets; still keep a raw window for the latest-readings list.
  const rawRange = range === "24h" ? "90m" : range;

  const devices = useQuery("devices", (s) => devicesApi.list({ signal: s }), REFRESH);
  const device = pickDevice(devices.data, searchParams.get("device"));
  const deviceId = device?.device_id ?? "";
  const isEnabled = device !== undefined;

  const sensorNow = useQuery(
    `sensor:${deviceId}`,
    (s) => devicesApi.sensorCurrent(deviceId, { signal: s }),
    { ...REFRESH, isEnabled },
  );
  const stats24h = useQuery(
    `stats24h:${deviceId}`,
    (s) => devicesApi.sensorStats24h(deviceId, { signal: s }),
    { ...REFRESH, isEnabled },
  );
  const rawStats = useQuery(
    `stats:${deviceId}:${rawRange}`,
    (s) => devicesApi.sensorStats(deviceId, rawRange, { signal: s }),
    { ...REFRESH, isEnabled },
  );

  // Combined chart series: raw readings for 30/60/90m, 30-minute buckets for 24h.
  const chartData = useMemo<ClimatePoint[]>(() => {
    if (range === "24h") {
      return (stats24h.data?.points ?? []).map((p) => ({
        t: new Date(p.bucket).getTime(),
        temp: p.avg_c,
        hum: p.avg_h,
        tempBand: [p.min_c, p.max_c],
        humBand: [p.min_h, p.max_h],
      }));
    }
    return (rawStats.data?.points ?? []).map((p) => ({
      t: new Date(p.time).getTime(),
      temp: p.temp_c,
      hum: p.humidity_pct,
    }));
  }, [range, stats24h.data, rawStats.data]);
  const chartQuery = range === "24h" ? stats24h : rawStats;

  const refetchAll = () => {
    devices.refetch();
    sensorNow.refetch();
    stats24h.refetch();
    rawStats.refetch();
  };
  const isFetching =
    devices.isFetching || sensorNow.isFetching || stats24h.isFetching || rawStats.isFetching;

  if (devices.data && devices.data.length === 0) {
    return (
      <Center height="100%" minHeight={400}>
        <EmptyState
          icon={<Icon icon={CpuChipIcon} size="lg" color="secondary" />}
          title={t("devices.emptyTitle")}
          description={t("devices.emptyDescription")}
        />
      </Center>
    );
  }

  const now = sensorNow.data;
  const summary24h =
    stats24h.data && stats24h.data.summary.samples > 0 ? stats24h.data.summary : undefined;
  const sensorLoading = devices.isLoading || sensorNow.isLoading || stats24h.isLoading;
  const climate = now ? climateStatus(now.temp_c, now.humidity_pct) : undefined;
  const readingSubtitle = now
    ? t("devices.overview.subtitle", { time: formatTime(now.time) })
    : t("devices.overview.noReading");
  const latest = (rawStats.data?.points ?? []).slice(-LATEST_COUNT).reverse();
  const rangeLabel = t(`devices.charts.ranges.${range}`);
  const loadError =
    devices.error ??
    (isNotFound(chartQuery.error) ? undefined : chartQuery.error) ??
    (isNotFound(sensorNow.error) ? undefined : sensorNow.error);

  return (
    <Page>
      <PageHeader
        title={t("climate.title")}
        description={t("climate.description")}
        meta={device ? <DevicePresence device={device} /> : null}
        actions={
          <>
            <Selector
              label={t("climate.deviceLabel")}
              isLabelHidden
              hasSearch={(devices.data?.length ?? 0) > 6}
              options={(devices.data ?? []).map((d) => ({
                value: d.device_id,
                label: `${d.name} · ${enumLabel("deviceStatus", d.status)}`,
              }))}
              value={device?.device_id}
              onChange={(id) => setSearchParams({ device: id }, { replace: true })}
              isDisabled={!devices.data}
            />
            <IconButton
              label={t("devices.refresh")}
              tooltip={t("devices.refresh")}
              icon={<ArrowPathIcon />}
              onClick={refetchAll}
              isLoading={isFetching}
            />
          </>
        }
      />

      {loadError ? (
        <Banner
          status="error"
          title={t("devices.loadErrorTitle")}
          description={t("devices.loadErrorDescription", { message: loadError.message })}
          endContent={<Button label={t("devices.retry")} size="sm" onClick={refetchAll} />}
        />
      ) : null}

      {/* Overview — current temperature + humidity */}
      <ClimateOverview
        title={t("devices.overview.title")}
        subtitle={readingSubtitle}
        deltaLabel={t("devices.overview.vsAverage")}
        isLoading={sensorLoading}
        statusLabel={t("devices.overview.statusLabel")}
        status={
          climate
            ? {
                status: climate.status,
                value: t(`devices.overview.status.${climate.status}`),
              }
            : undefined
        }
        temperature={{
          label: t("devices.charts.temperature"),
          value: now ? `${formatDecimal(now.temp_c)} °C` : "—",
          delta: now && summary24h ? now.temp_c - summary24h.temp_c.avg : undefined,
          unit: "°C",
          color: SERIES_COLORS.temperature,
        }}
        humidity={{
          label: t("devices.charts.humidity"),
          value: now ? `${formatDecimal(now.humidity_pct)} %` : "—",
          delta: now && summary24h ? now.humidity_pct - summary24h.humidity_pct.avg : undefined,
          unit: "%",
          color: SERIES_COLORS.humidity,
        }}
      />

      {/* Combined temperature + humidity chart, latest readings beside it */}
      <Grid gap={4} columns={isStacked ? 1 : 4}>
        <GridSpan columns={isStacked ? "full" : 3}>
          <ClimateChart
            title={t("devices.charts.title")}
            description={t("devices.charts.description", { range: rangeLabel })}
            data={chartData}
            isLoading={devices.isLoading || chartQuery.isLoading}
            actions={
              <SegmentedControl
                label={t("devices.charts.rangeLabel")}
                value={range}
                onChange={(v) => setRange(v as SensorRange)}
                size="sm"
              >
                {SENSOR_RANGES.map((r) => (
                  <SegmentedControlItem key={r} value={r} label={t(`devices.charts.ranges.${r}`)} />
                ))}
              </SegmentedControl>
            }
          />
        </GridSpan>
        <GridSpan columns={isStacked ? "full" : 1}>
          <Card padding={4}>
            <VStack gap={3}>
              <Heading level={4}>{t("devices.overview.latest")}</Heading>
              {rawStats.isLoading || devices.isLoading ? (
                <VStack gap={2}>
                  {Array.from({ length: LATEST_COUNT }, (_, i) => (
                    <Skeleton key={i} height={36} index={i} />
                  ))}
                </VStack>
              ) : latest.length === 0 ? (
                <Text color="secondary">{t("devices.charts.noData")}</Text>
              ) : (
                <List density="compact" hasDividers>
                  {latest.map((p) => (
                    <ListItem
                      key={p.time}
                      label={<Text hasTabularNumbers>{formatTime(p.time)}</Text>}
                      endContent={
                        <VStack gap={0} hAlign="end">
                          <Text hasTabularNumbers weight="semibold">
                            {`${formatDecimal(p.temp_c)} °C`}
                          </Text>
                          <Text type="supporting" color="secondary" hasTabularNumbers>
                            {`${formatDecimal(p.humidity_pct)} %`}
                          </Text>
                        </VStack>
                      }
                    />
                  ))}
                </List>
              )}
            </VStack>
          </Card>
        </GridSpan>
      </Grid>

      <DeviceAnalysis
        stats24h={stats24h.data}
        isLoading={devices.isLoading || stats24h.isLoading}
      />
    </Page>
  );
}
