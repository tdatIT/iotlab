import { lazy, Suspense } from "react";
import { Navigate, Outlet, Route, Routes } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Center } from "@astryxdesign/core/Center";
import { Spinner } from "@astryxdesign/core/Spinner";

import { ROUTES } from "@/config/app";
import { AdminShell } from "@/components/layout/AdminShell";

// Pages are lazy-loaded so heavy dependencies (e.g. recharts on the
// temperature page) stay out of the initial bundle.
const DevicesPage = lazy(() =>
  import("@/features/devices/DevicesPage").then((m) => ({
    default: m.DevicesPage,
  })),
);
const DeviceDetailPage = lazy(() =>
  import("@/features/devices/DeviceDetailPage").then((m) => ({
    default: m.DeviceDetailPage,
  })),
);
const ClimatePage = lazy(() =>
  import("@/features/climate/ClimatePage").then((m) => ({
    default: m.ClimatePage,
  })),
);
const NotFoundPage = lazy(() =>
  import("@/features/misc/NotFoundPage").then((m) => ({
    default: m.NotFoundPage,
  })),
);
const ServerErrorPage = lazy(() =>
  import("@/features/misc/ServerErrorPage").then((m) => ({
    default: m.ServerErrorPage,
  })),
);

function PageFallback() {
  const { t } = useTranslation();
  return (
    <Center height="100%" minHeight={320}>
      <Spinner label={t("common.loadingPage")} />
    </Center>
  );
}

/** App pages render inside the shell; the 500 page renders bare. */
function ShellLayout() {
  return (
    <AdminShell>
      <Suspense fallback={<PageFallback />}>
        <Outlet />
      </Suspense>
    </AdminShell>
  );
}

export function AppRouter() {
  return (
    <Routes>
      {/* Standalone 500 page (preview at /500; the ErrorBoundary renders it on crashes). */}
      <Route
        path={ROUTES.serverError}
        element={
          <Suspense fallback={<PageFallback />}>
            <ServerErrorPage />
          </Suspense>
        }
      />

      <Route element={<ShellLayout />}>
        <Route index element={<Navigate to={ROUTES.devices} replace />} />
        <Route path={ROUTES.devices} element={<DevicesPage />} />
        <Route path={`${ROUTES.devices}/:deviceId`} element={<DeviceDetailPage />} />
        <Route path={ROUTES.temperature} element={<ClimatePage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
