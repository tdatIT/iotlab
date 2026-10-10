import { useEffect, useRef, useState, type ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { AppShell } from "@astryxdesign/core/AppShell";
import { TopNav, TopNavHeading } from "@astryxdesign/core/TopNav";
import { NavIcon } from "@astryxdesign/core/NavIcon";
import {
  SideNav,
  SideNavItem,
  SideNavSection,
} from "@astryxdesign/core/SideNav";
import { CpuChipIcon } from "@heroicons/react/24/solid";

import { HStack, VStack } from "@astryxdesign/core/Stack";
import { Section } from "@astryxdesign/core/Section";
import { Text } from "@astryxdesign/core/Text";

import { APP_CONFIG, ROUTES } from "@/config/app";
import { NAV_SECTIONS } from "@/config/navigation";
import { useMediaQuery } from "@/lib/useMediaQuery";
import { LanguageMenu } from "./LanguageMenu";
import { ThemeToggle } from "./ThemeToggle";

// Responsive contract:
//   > 768px   top nav 48 | 48px icon rail that expands over the content on
//             hover/focus (overlay styles in styles/global.css)
//   <= 768px  AppShell moves the side nav into the mobile drawer (always expanded)
interface AdminShellProps {
  children: ReactNode;
}

function isPathSelected(pathname: string, itemPath: string): boolean {
  return pathname === itemPath || pathname.startsWith(`${itemPath}/`);
}

/**
 * The single app frame: TopNav (identity + user menu) and SideNav driven by
 * config/navigation.ts. Pages never touch the shell — they render inside it
 * via the router's <Outlet>.
 */
export function AdminShell({ children }: AdminShellProps) {
  const { pathname } = useLocation();
  const { t } = useTranslation();
  const isDrawer = useMediaQuery("(max-width: 768px)");
  // Rail stays collapsed; pointer hover or keyboard focus expands it.
  const [isNavActive, setIsNavActive] = useState(false);
  const isNavCollapsed = !isDrawer && !isNavActive;
  const navRef = useRef<HTMLElement>(null);

  // SideNav has no pointer/focus props; listen on its root element.
  useEffect(() => {
    const nav = navRef.current;
    if (!nav) return;
    const open = () => setIsNavActive(true);
    const close = () => setIsNavActive(false);
    const onFocusOut = (e: FocusEvent) => {
      if (!nav.contains(e.relatedTarget as Node | null)) close();
    };
    nav.addEventListener("mouseenter", open);
    nav.addEventListener("mouseleave", close);
    nav.addEventListener("focusin", open);
    nav.addEventListener("focusout", onFocusOut);
    return () => {
      nav.removeEventListener("mouseenter", open);
      nav.removeEventListener("mouseleave", close);
      nav.removeEventListener("focusin", open);
      nav.removeEventListener("focusout", onFocusOut);
    };
  }, []);

  return (
    <AppShell
      contentPadding={0}
      topNav={
        <TopNav
          label={t("nav.mainNavigation")}
          heading={
            <TopNavHeading
              heading={APP_CONFIG.name}
              headingHref={ROUTES.devices}
              logo={
                <NavIcon
                  icon={<CpuChipIcon style={{ width: 16, height: 16 }} />}
                />
              }
            />
          }
          endContent={
            <HStack gap={1} vAlign="center">
              <LanguageMenu />
              <ThemeToggle />
            </HStack>
          }
        />
      }
      sideNav={
        <SideNav
          ref={navRef}
          collapsible={{ isCollapsed: isNavCollapsed, hasButton: false }}
          footer={
            isNavCollapsed ? undefined : (
              <Section variant="transparent" padding={3}>
                <VStack hAlign="center">
                  <Text type="supporting" color="secondary">
                    {APP_CONFIG.name} v{APP_CONFIG.version}
                  </Text>
                </VStack>
              </Section>
            )
          }
        >
          {NAV_SECTIONS.map((section) => (
            <SideNavSection
              key={section.titleKey}
              title={t(`nav.sections.${section.titleKey}`)}
              isHeaderHidden={section.isHeaderHidden}
            >
              {section.items.map((item) => (
                <SideNavItem
                  key={item.path}
                  label={t(`nav.${item.labelKey}`)}
                  href={item.path}
                  icon={item.icon}
                  selectedIcon={item.iconSelected}
                  isSelected={isPathSelected(pathname, item.path)}
                />
              ))}
            </SideNavSection>
          ))}
        </SideNav>
      }
    >
      {children}
    </AppShell>
  );
}
