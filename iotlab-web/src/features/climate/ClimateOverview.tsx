import type { ComponentType, SVGProps } from "react";
import { Card } from "@astryxdesign/core/Card";
import { Grid } from "@astryxdesign/core/Grid";
import { Icon } from "@astryxdesign/core/Icon";
import { Skeleton } from "@astryxdesign/core/Skeleton";
import { HStack, VStack } from "@astryxdesign/core/Stack";
import { Heading, Text } from "@astryxdesign/core/Text";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  CloudIcon,
  FaceSmileIcon,
  FireIcon,
  SunIcon,
} from "@heroicons/react/24/outline";
import { StopIcon } from "@heroicons/react/24/solid";

import { STATUS_COLORS } from "@/config/colors";
import { formatDecimal } from "@/lib/format";
import { useIsMobile } from "@/lib/useMediaQuery";
import type { ClimateStatus } from "./climateStatus";
import { SnowflakeIcon } from "@/components/icons/SnowflakeIcon";

const STATUS_ICON: Record<ClimateStatus, ComponentType<SVGProps<SVGSVGElement>>> = {
  tooHot: FireIcon,
  hot: SunIcon,
  cold: SnowflakeIcon,
  humid: CloudIcon,
  comfortable: FaceSmileIcon,
};

export interface OverviewMetric {
  label: string;
  value: string;
  /** Change vs the reference (e.g. 24h average), same unit as `value`. */
  delta?: number;
  unit: string;
  color: string;
}

export interface OverviewStatus {
  status: ClimateStatus;
  /** Status name, e.g. "Hot". */
  value: string;
}

interface ClimateOverviewProps {
  title: string;
  subtitle: string;
  deltaLabel: string;
  temperature: OverviewMetric;
  humidity: OverviewMetric;
  statusLabel: string;
  /** Undefined until there is a reading. */
  status?: OverviewStatus;
  isLoading?: boolean;
}

/** Current conditions in three columns: temperature | humidity | status. */
export function ClimateOverview({
  title,
  subtitle,
  deltaLabel,
  temperature,
  humidity,
  statusLabel,
  status,
  isLoading,
}: ClimateOverviewProps) {
  const isMobile = useIsMobile();
  const valueType = isMobile ? "large" : "display-3";

  return (
    <Card>
      <VStack gap={4}>
        <VStack gap={0}>
          <Heading level={3}>{title}</Heading>
          <Text type="supporting" color="secondary">
            {subtitle}
          </Text>
        </VStack>
        <Grid gap={isMobile ? 3 : 4} columns={3}>
          {[temperature, humidity].map((m) => (
            <VStack key={m.label} gap={1}>
              <HStack gap={2} vAlign="center">
                <Icon icon={StopIcon} size="xsm" style={{ color: m.color }} />
                <Text type="supporting" color="secondary">
                  {m.label}
                </Text>
              </HStack>
              {isLoading ? (
                <Skeleton height={40} width="70%" />
              ) : (
                <>
                  <Text type={valueType} weight="bold" hasTabularNumbers>
                    {m.value}
                  </Text>
                  {m.delta !== undefined ? (
                    <HStack gap={1} vAlign="center">
                      <Icon
                        icon={m.delta >= 0 ? ArrowUpIcon : ArrowDownIcon}
                        size="xsm"
                        color="secondary"
                      />
                      <Text type="supporting" color="secondary" hasTabularNumbers>
                        {`${m.delta >= 0 ? "+" : "−"}${formatDecimal(Math.abs(m.delta))} ${m.unit}`}
                        {isMobile ? null : ` ${deltaLabel}`}
                      </Text>
                    </HStack>
                  ) : null}
                </>
              )}
            </VStack>
          ))}

          <VStack gap={1}>
            <Text type="supporting" color="secondary">
              {statusLabel}
            </Text>
            {isLoading ? (
              <Skeleton height={40} width="70%" />
            ) : status ? (
              <HStack gap={2} vAlign="center">
                <Icon
                  icon={STATUS_ICON[status.status]}
                  size={isMobile ? "md" : "lg"}
                  style={{ color: STATUS_COLORS[status.status] }}
                />
                <Text type={valueType} weight="bold">
                  {status.value}
                </Text>
              </HStack>
            ) : (
              <Text type={valueType} weight="bold">
                —
              </Text>
            )}
          </VStack>
        </Grid>
      </VStack>
    </Card>
  );
}
