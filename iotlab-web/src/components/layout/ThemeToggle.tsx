import { useTranslation } from "react-i18next";
import { IconButton } from "@astryxdesign/core/IconButton";
import { MoonIcon, SunIcon } from "@heroicons/react/24/outline";

import { useThemeMode } from "@/app/theme";

/** Light/dark switch for the TopNav. Persists across sessions. */
export function ThemeToggle() {
  const { t } = useTranslation();
  const { resolvedMode, toggle } = useThemeMode();
  const isDark = resolvedMode === "dark";

  return (
    <IconButton
      label={isDark ? t("theme.switchToLight") : t("theme.switchToDark")}
      tooltip={isDark ? t("theme.light") : t("theme.dark")}
      variant="ghost"
      icon={isDark ? <SunIcon /> : <MoonIcon />}
      onClick={toggle}
    />
  );
}
