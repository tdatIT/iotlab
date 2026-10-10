import { useTranslation } from "react-i18next";
import { Center } from "@astryxdesign/core/Center";
import { EmptyState } from "@astryxdesign/core/EmptyState";
import { Button } from "@astryxdesign/core/Button";
import { Icon } from "@astryxdesign/core/Icon";
import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";

import { ROUTES } from "@/config/app";

interface ServerErrorPageProps {
  /** Shown under the title when the crash carries a message (dev aid). */
  detail?: string;
}

/**
 * 500 page. Rendered standalone (no shell) — the shell itself may be what
 * crashed. Used by the app ErrorBoundary and previewable at /500.
 */
export function ServerErrorPage({ detail }: ServerErrorPageProps) {
  const { t } = useTranslation();
  return (
    <Center height="100dvh" style={{ background: "var(--color-background-body)" }}>
      <EmptyState
        icon={<Icon icon={ExclamationTriangleIcon} size="lg" color="error" />}
        title={t("errors.serverErrorTitle")}
        description={detail ?? t("errors.serverErrorDescription")}
        actions={
          <>
            <Button
              label={t("errors.reload")}
              variant="primary"
              onClick={() => window.location.reload()}
            />
            <Button label={t("errors.backToDashboard")} href={ROUTES.devices} />
          </>
        }
      />
    </Center>
  );
}
