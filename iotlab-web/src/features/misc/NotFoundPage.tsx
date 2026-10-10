import { useTranslation } from "react-i18next";
import { Center } from "@astryxdesign/core/Center";
import { EmptyState } from "@astryxdesign/core/EmptyState";
import { Button } from "@astryxdesign/core/Button";
import { Icon } from "@astryxdesign/core/Icon";
import { MapIcon } from "@heroicons/react/24/outline";

import { ROUTES } from "@/config/app";

export function NotFoundPage() {
  const { t } = useTranslation();
  return (
    <Center height="100%" minHeight={400}>
      <EmptyState
        icon={<Icon icon={MapIcon} size="lg" color="secondary" />}
        title={t("errors.notFoundTitle")}
        description={t("errors.notFoundDescription")}
        actions={
          <Button label={t("errors.backToDashboard")} variant="primary" href={ROUTES.devices} />
        }
      />
    </Center>
  );
}
