import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Banner } from "@astryxdesign/core/Banner";
import { Button } from "@astryxdesign/core/Button";
import { IconButton } from "@astryxdesign/core/IconButton";
import { Card } from "@astryxdesign/core/Card";
import { Grid } from "@astryxdesign/core/Grid";
import { Link } from "@astryxdesign/core/Link";
import { Skeleton } from "@astryxdesign/core/Skeleton";
import { VStack } from "@astryxdesign/core/Stack";
import { Text } from "@astryxdesign/core/Text";
import { pixel, proportional, type TableColumn } from "@astryxdesign/core/Table";
import { ArrowPathIcon } from "@heroicons/react/24/outline";

import { Page } from "@/components/layout/Page";
import { PageHeader } from "@/components/layout/PageHeader";
import { DataTable } from "@/components/common/DataTable";
import { StatCard } from "@/components/common/StatCard";
import { TableToolbar } from "@/components/common/TableToolbar";
import { APP_CONFIG, deviceDetailPath } from "@/config/app";
import { formatRelative } from "@/lib/format";
import { useIsMobile } from "@/lib/useMediaQuery";
import { useQuery } from "@/lib/useQuery";
import { devicesApi } from "./api";
import { DeviceStatusLabel } from "./DeviceStatusLabel";
import type { Device } from "./types";

type DeviceRow = Device & Record<string, unknown>;

export function DevicesPage() {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [search, setSearch] = useState("");
  const devices = useQuery("devices", (signal) => devicesApi.list({ signal }), {
    refreshMs: APP_CONFIG.refreshMs,
  });

  const all = useMemo(() => (devices.data ?? []) as DeviceRow[], [devices.data]);
  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return all;
    return all.filter((d) =>
      [d.name, d.device_id, d.ip, d.fw, d.location ?? ""].some((v) =>
        v.toLowerCase().includes(q),
      ),
    );
  }, [all, search]);

  const onlineCount = all.filter((d) => d.status === "online").length;

  const columns: TableColumn<DeviceRow>[] = [
    {
      key: "name",
      header: t("devices.columns.name"),
      width: proportional(2),
      renderCell: (d) => (
        <VStack gap={0}>
          <Link href={deviceDetailPath(d.device_id)}>{d.name || d.device_id}</Link>
          <Text type="supporting" color="secondary">
            {d.device_id}
          </Text>
        </VStack>
      ),
    },
    {
      key: "status",
      header: t("devices.columns.status"),
      width: pixel(140),
      renderCell: (d) => <DeviceStatusLabel status={d.status} />,
    },
    {
      key: "ip",
      header: t("devices.columns.ip"),
      width: pixel(150),
      renderCell: (d) => <Text hasTabularNumbers>{d.ip}</Text>,
    },
    { key: "fw", header: t("devices.columns.firmware"), width: pixel(150) },
    {
      key: "location",
      header: t("devices.columns.location"),
      width: proportional(1),
      renderCell: (d) =>
        d.location ?? (
          <Text color="secondary">{t("devices.noLocation")}</Text>
        ),
    },
    {
      key: "last_seen",
      header: t("devices.columns.lastSeen"),
      width: pixel(160),
      renderCell: (d) => <Text>{formatRelative(d.last_seen)}</Text>,
    },
  ];

  return (
    <Page>
      <PageHeader
        title={t("devices.title")}
        description={t("devices.description")}
        actions={
          <IconButton
            label={t("devices.refresh")}
            tooltip={t("devices.refresh")}
            icon={<ArrowPathIcon />}
            onClick={devices.refetch}
            isLoading={devices.isFetching}
          />
        }
      />

      {devices.error ? (
        <Banner
          status="error"
          title={t("devices.loadErrorTitle")}
          description={t("devices.loadErrorDescription", {
            message: devices.error.message,
          })}
          endContent={
            <Button label={t("devices.retry")} size="sm" onClick={devices.refetch} />
          }
        />
      ) : null}

      <Grid gap={3} columns={{ minWidth: 96, repeat: "fit" }}>
        <StatCard
          label={t("devices.kpi.total")}
          value={String(all.length)}
          isLoading={devices.isLoading}
        />
        <StatCard
          label={t("devices.kpi.online")}
          value={String(onlineCount)}
          isLoading={devices.isLoading}
        />
        <StatCard
          label={t("devices.kpi.offline")}
          value={String(all.length - onlineCount)}
          isLoading={devices.isLoading}
        />
      </Grid>

      <Card padding={0}>
        {devices.isLoading ? (
          <VStack gap={3} padding={4}>
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} height={40} index={i} />
            ))}
          </VStack>
        ) : (
          <DataTable
            data={rows}
            columns={columns}
            idKey="device_id"
            toolbar={
              <TableToolbar
                label={t("devices.toolbarLabel")}
                search={search}
                onSearchChange={setSearch}
                searchPlaceholder={t("devices.searchPlaceholder")}
                actions={
                  isMobile ? null : (
                    <Text type="supporting" color="secondary">
                      {t("devices.autoRefresh", {
                        seconds: APP_CONFIG.refreshMs / 1000,
                      })}
                    </Text>
                  )
                }
              />
            }
            emptyTitle={all.length === 0 ? t("devices.emptyTitle") : undefined}
            emptyDescription={
              all.length === 0 ? t("devices.emptyDescription") : undefined
            }
            emptyActions={
              search ? (
                <Button
                  label={t("common.clearFilters")}
                  onClick={() => setSearch("")}
                />
              ) : undefined
            }
          />
        )}
      </Card>
    </Page>
  );
}
