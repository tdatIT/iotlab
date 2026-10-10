import { createElement, type ComponentType, type SVGProps } from "react";
import { CpuChipIcon } from "@heroicons/react/24/outline";
import { CpuChipIcon as CpuChipIconSolid } from "@heroicons/react/24/solid";

import { ThermometerIcon, ThermometerIconSolid } from "@/components/icons/ThermometerIcon";
import type { en } from "@/i18n/locales/en";
import { ROUTES } from "./app";
import { SERIES_COLORS } from "./colors";

export type NavIconType = ComponentType<SVGProps<SVGSVGElement>>;

/** Icon variant painted in a fixed color (inline style beats the nav's own icon color). */
function tinted(Icon: NavIconType, color: string): NavIconType {
  const Tinted = (props: SVGProps<SVGSVGElement>) =>
    createElement(Icon, { ...props, style: { ...props.style, color } });
  Tinted.displayName = `Tinted(${Icon.displayName ?? Icon.name})`;
  return Tinted;
}

/** Keys into the `nav` catalog namespace — labels are translated at render. */
export type NavItemKey = Exclude<
  keyof typeof en.nav,
  "sections" | "mainNavigation"
>;
export type NavSectionKey = keyof typeof en.nav.sections;

export interface NavItem {
  labelKey: NavItemKey;
  path: string;
  /** Outline variant, shown when the item is not selected. */
  icon: NavIconType;
  /** Filled (optionally tinted) variant for the selected state (falls back to `icon`). */
  iconSelected?: NavIconType;
}

export interface NavSection {
  titleKey: NavSectionKey;
  /** Hide the section title while keeping it for screen readers. */
  isHeaderHidden?: boolean;
  items: NavItem[];
}

/**
 * Single source of truth for the side navigation.
 * Add a page: route in app/router.tsx, label in i18n/locales/*, entry here.
 */
export const NAV_SECTIONS: NavSection[] = [
  {
    titleKey: "lab",
    items: [
      {
        labelKey: "devices",
        path: ROUTES.devices,
        icon: CpuChipIcon,
        // Selected page color: devices blue, temperature red-orange.
        iconSelected: tinted(CpuChipIconSolid, SERIES_COLORS.humidity),
      },
      {
        labelKey: "temperature",
        path: ROUTES.temperature,
        icon: ThermometerIcon,
        iconSelected: tinted(ThermometerIconSolid, SERIES_COLORS.temperature),
      },
    ],
  },
];
