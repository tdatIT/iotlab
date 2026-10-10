import { useTranslation } from "react-i18next";
import { HStack } from "@astryxdesign/core/Stack";
import type { StatusDotVariant } from "@astryxdesign/core/StatusDot";
import { Text } from "@astryxdesign/core/Text";

import { StatusLabel } from "@/components/common/StatusLabel";
import { enumLabel } from "@/i18n";
import { formatRelative } from "@/lib/format";
import type { Device, DeviceStatus } from "./types";

const STATUS_VARIANT: Record<string, StatusDotVariant> = {
  online: "success",
  offline: "error",
};

export function DeviceStatusLabel({ status }: { status: DeviceStatus }) {
  return (
    <StatusLabel
      variant={STATUS_VARIANT[status] ?? "neutral"}
      label={enumLabel("deviceStatus", status)}
      isPulsing={status === "online"}
    />
  );
}

/** Page-header line: status + "last seen 5 seconds ago". */
export function DevicePresence({ device }: { device: Device }) {
  const { t } = useTranslation();
  return (
    <HStack gap={3} vAlign="center" wrap="wrap">
      <DeviceStatusLabel status={device.status} />
      <Text type="supporting" color="secondary">
        {`${t("devices.detail.lastSeen")} ${formatRelative(device.last_seen)}`}
      </Text>
    </HStack>
  );
}
