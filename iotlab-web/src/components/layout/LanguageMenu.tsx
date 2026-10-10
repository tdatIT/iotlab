import { useTranslation } from "react-i18next";
import { DropdownMenu } from "@astryxdesign/core/DropdownMenu";
import { Icon } from "@astryxdesign/core/Icon";
import { CheckIcon, LanguageIcon } from "@heroicons/react/24/outline";

import { SUPPORTED_LANGUAGES, type AppLanguage } from "@/i18n";

/** en/vi switcher in the TopNav. Selection persists via i18n's languageChanged hook. */
export function LanguageMenu() {
  const { t, i18n } = useTranslation();
  const current = (i18n.language as AppLanguage) ?? "en";

  return (
    <DropdownMenu
      button={{
        label: t("language.label"),
        variant: "ghost",
        icon: <LanguageIcon style={{ width: 16, height: 16 }} />,
        isIconOnly: true,
      }}
      menuWidth={180}
      items={SUPPORTED_LANGUAGES.map((lang) => ({
        label: t(`language.${lang}`),
        icon:
          current === lang ? (
            <Icon icon={CheckIcon} size="sm" color="accent" />
          ) : undefined,
        onClick: () => void i18n.changeLanguage(lang),
      }))}
    />
  );
}
