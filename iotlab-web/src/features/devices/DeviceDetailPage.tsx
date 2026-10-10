import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Banner } from "@astryxdesign/core/Banner";
import { BreadcrumbItem, Breadcrumbs } from "@astryxdesign/core/Breadcrumbs";
import { Button } from "@astryxdesign/core/Button";
import { IconButton } from "@astryxdesign/core/IconButton";
import { Card } from "@astryxdesign/core/Card";
import { Center } from "@astryxdesign/core/Center";
import { EmptyState } from "@astryxdesign/core/EmptyState";
import { Grid } from "@astryxdesign/core/Grid";
import { Icon } from "@astryxdesign/core/Icon";
import { MetadataList, MetadataListItem } from "@astryxdesign/core/MetadataList";
import { Skeleton } from "@astryxdesign/core/Skeleton";
import { VStack } from "@astryxdesign/core/Stack";
import { Heading, Text } from "@astryxdesign/core/Text";
import { Timestamp } from "@astryxdesign/core/Timestamp";
import {
  ArrowPathIcon,
  ClockIcon,
  CpuChipIcon,
  ExclamationTriangleIcon,
  SignalIcon,
} from "@heroicons/react/24/outline";

import { ThermometerIcon } from "@/components/icons/ThermometerIcon";
import { Page } from "@/components/layout/Page";
import { PageHeader } from "@/components/layout/PageHeader";
import { APP_CONFIG, ROUTES, temperaturePath } from "@/config/app";
import { ApiError } from "@/lib/api";
import {
  formatBytes,
  formatDecimal,
  formatDuration,
  formatNumber,
  formatTime,
} from "@/lib/format";
import { useIsMobile } from "@/lib/useMediaQuery";
import { useQuery } from "@/lib/useQuery";
import { devicesApi } from "./api";
import { DeviceAnalysis } from "./DeviceAnalysis";
import { DevicePresence } from "./DeviceStatusLabel";
import { HealthGauge } from "./HealthGauge";
import { healthGauges, signalQuality } from "./signal";

const REFRESH = { refreshMs: APP_CONFIG.refreshMs };

function isNotFound(error: Error | undefined) {
  return error instanceof ApiError && error.status === 404;
}

/** One device: info + telemetry (health gauges, 1h telemetry summary). */
export function DeviceDetailPage() {
  const { t } = useTranslation();
  const { deviceId = "" } = useParams();
  const isMobile = useIsMobile();

  // There is no GET /devices/{id}; the list is small, so pick from it.
  const devices = useQuery("devices", (s) => devicesApi.list({ signal: s }), REFRESH);
  const device = devices.data?.find((d) => d.device_id === deviceId);
  const isUnknownDevice = devices.data !== undefined && !device;
  // Wait for the device to resolve: avoids 404 bursts for unknown IDs and
  // spreads requests under the API rate limit.
  const isEnabled = device !== undefined;

  const telemetry = useQuery(
    `telemetry:${deviceId}`,
    (s) => devicesApi.telemetryCurrent(deviceId, { signal: s }),
    { ...REFRESH, isEnabled },
  );
  const telemetryStats = useQuery(
    `telemetryStats:${deviceId}`,
    (s) => devicesApi.telemetryStats(deviceId, "60m", { signal: s }),
    { ...REFRESH, isEnabled },
  );
  // Recent raw readings, only to estimate the sensor cadence for the error ring.
  const recentReadings = useQuery(
    `stats:${deviceId}:30m`,
    (s) => devicesApi.sensorStats(deviceId, "30m", { signal: s }),
    { ...REFRESH, isEnabled },
  );

  const refetchAll = () => {
    devices.refetch();
    telemetry.refetch();
    telemetryStats.refetch();
    recentReadings.refetch();
  };
  const isFetching =
    devices.isFetching ||
    telemetry.isFetching ||
    telemetryStats.isFetching ||
    recentReadings.isFetching;

  if (isUnknownDevice) {
    return (
      <Center height="100%" minHeight={400}>
        <EmptyState
          icon={<Icon icon={CpuChipIcon} size="lg" color="secondary" />}
          title={t("devices.detail.notFoundTitle")}
          description={t("devices.detail.notFoundDescription", { id: deviceId })}
          actions={
            <Button
              label={t("devices.detail.backToList")}
              variant="primary"
              href={ROUTES.devices}
            />
          }
        />
      </Center>
    );
  }

  const tel = telemetry.data;
  const hasNoTelemetry = isNotFound(telemetry.error);
  const isLoading = devices.isLoading || telemetry.isLoading;
  const quality = tel ? signalQuality(tel.rssi_dbm) : undefined;
  const gauges = healthGauges(tel, recentReadings.data?.points ?? undefined);
  const loadError = devices.error ?? (hasNoTelemetry ? undefined : telemetry.error);

  return (
    <Page>
      <VStack gap={3}>
        <Breadcrumbs variant="supporting">
          <BreadcrumbItem href={ROUTES.devices}>{t("nav.devices")}</BreadcrumbItem>
          <BreadcrumbItem isCurrent>{device?.name ?? deviceId}</BreadcrumbItem>
        </Breadcrumbs>
        <PageHeader
          title={device?.name ?? deviceId}
          description={deviceId}
          meta={device ? <DevicePresence device={device} /> : null}
          actions={
            <>
              <Button
                label={t("devices.detail.viewTemperature")}
                icon={<ThermometerIcon />}
                href={temperaturePath(deviceId)}
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
      </VStack>

      {loadError ? (
        <Banner
          status="error"
          title={t("devices.loadErrorTitle")}
          description={t("devices.loadErrorDescription", { message: loadError.message })}
          endContent={<Button label={t("devices.retry")} size="sm" onClick={refetchAll} />}
        />
      ) : null}

      <Card padding={4}>
        {device ? (
          <MetadataList
            columns="multi"
            title={<Heading level={4}>{t("devices.detail.info")}</Heading>}
          >
            <MetadataListItem label={t("devices.detail.ip")}>
              <Text hasTabularNumbers>{tel?.ip ?? device.ip}</Text>
            </MetadataListItem>
            <MetadataListItem label={t("devices.detail.firmware")}>
              {tel?.fw ?? device.fw}
            </MetadataListItem>
            <MetadataListItem label={t("devices.detail.location")}>
              {device.location ?? t("devices.noLocation")}
            </MetadataListItem>
            <MetadataListItem label={t("devices.detail.firstSeen")}>
              <Timestamp value={device.first_seen} format="date_time" color="primary" type="body" />
            </MetadataListItem>
            <MetadataListItem label={t("devices.detail.statusSince")}>
              <Timestamp value={device.status_changed_at} format="date_time" color="primary" type="body" />
            </MetadataListItem>
            <MetadataListItem label={t("devices.detail.resetReason")}>
              {tel?.reset_reason ?? "—"}
            </MetadataListItem>
          </MetadataList>
        ) : (
          <Skeleton height={120} />
        )}
      </Card>

      <VStack gap={3}>
        <VStack gap={1}>
          <Heading level={4}>{t("devices.detail.health")}</Heading>
          <Text type="supporting" color="secondary">
            {hasNoTelemetry
              ? t("devices.detail.noTelemetry")
              : tel
                ? `${t("devices.detail.healthDescription")} ${t("devices.updatedAt", { time: formatTime(tel.time) })}`
                : t("devices.detail.healthDescription")}
          </Text>
        </VStack>
        <Grid gap={isMobile ? 3 : 4} columns={{ minWidth: isMobile ? 140 : 190, repeat: "fit" }}>
          <HealthGauge
            icon={<Icon icon={ClockIcon} size="sm" color="secondary" />}
            label={t("devices.stats.uptime")}
            percent={gauges.uptimePct}
            tone="accent"
            value={tel ? formatDuration(tel.uptime_s) : "—"}
            caption={tel ? t("devices.stats.uptimeCaption", { pct: formatDecimal(gauges.uptimePct, 0) }) : undefined}
            hint={tel ? t("devices.stats.uptimeHint", { reason: tel.reset_reason }) : undefined}
            isLoading={isLoading}
          />
          <HealthGauge
            icon={<Icon icon={CpuChipIcon} size="sm" color="secondary" />}
            label={t("devices.stats.memory")}
            percent={gauges.memoryPct}
            tone={gauges.memoryTone}
            value={tel ? formatBytes(tel.heap_free) : "—"}
            caption={t("devices.stats.memoryCaption")}
            hint={tel ? t("devices.stats.memoryHint", { min: formatBytes(tel.heap_min) }) : undefined}
            isLoading={isLoading}
          />
          <HealthGauge
            icon={<Icon icon={SignalIcon} size="sm" color="secondary" />}
            label={t("devices.stats.connection")}
            percent={gauges.signalPct}
            tone={gauges.signalTone}
            value={tel ? `${tel.rssi_dbm} dBm` : "—"}
            caption={quality ? t(`devices.signal.${quality}`) : undefined}
            hint={
              tel
                ? t("devices.stats.connectionHint", {
                    wifi: tel.wifi_reconnects,
                    mqtt: tel.mqtt_reconnects,
                  })
                : undefined
            }
            isLoading={isLoading}
          />
          <HealthGauge
            icon={<Icon icon={ExclamationTriangleIcon} size="sm" color="secondary" />}
            label={t("devices.stats.errors")}
            percent={gauges.readOkPct}
            tone={gauges.errorTone}
            value={tel ? formatNumber(tel.sensor_errors) : "—"}
            caption={
              tel
                ? t("devices.stats.errorsCaption", {
                    pct: formatDecimal(gauges.readOkPct, gauges.readOkPct >= 99.95 ? 0 : 1),
                  })
                : undefined
            }
            hint={t("devices.stats.errorsHint")}
            isLoading={isLoading}
          />
        </Grid>
      </VStack>

      <DeviceAnalysis
        telemetryStats={telemetryStats.data}
        isLoading={devices.isLoading || telemetryStats.isLoading}
      />
    </Page>
  );
}
