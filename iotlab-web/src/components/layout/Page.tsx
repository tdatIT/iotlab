import type { ReactNode } from "react";
import { VStack } from "@astryxdesign/core/Stack";
import { Section } from "@astryxdesign/core/Section";

interface PageProps {
  children: ReactNode;
  /**
   * Content width policy:
   * - "full": stretch (dashboards, tables)
   * - "narrow": readable column, max 720px (settings, forms)
   */
  width?: "full" | "narrow";
}

/**
 * Standard page body: consistent padding + vertical rhythm for every page.
 * AppShell contentPadding is 0 (tables need edge-to-edge), so pages own
 * their padding through this wrapper. Compose with <PageHeader> as first child.
 */
export function Page({ children, width = "full" }: PageProps) {
  return (
    <Section variant="transparent" padding={6}>
      <VStack gap={6} maxWidth={width === "narrow" ? 720 : undefined}>
        {children}
      </VStack>
    </Section>
  );
}
