import type { ReactNode } from "react";
import { HStack, VStack, StackItem } from "@astryxdesign/core/Stack";
import { Heading, Text } from "@astryxdesign/core/Text";

interface PageHeaderProps {
  title: string;
  /** One-line summary under the title. */
  description?: string;
  /** Extra line under the description (status, timestamps, ...). */
  meta?: ReactNode;
  /** Right-aligned actions (Buttons, filters, date pickers, ...). */
  actions?: ReactNode;
}

/**
 * Consistent page header: title + optional description on the left, actions
 * on the right. On narrow screens the actions wrap onto their own row.
 */
export function PageHeader({ title, description, meta, actions }: PageHeaderProps) {
  return (
    <HStack gap={3} vAlign="center" wrap="wrap">
      {/* 18rem basis: below that the actions drop under the title instead of squeezing it */}
      <StackItem size="fill" style={{ flexBasis: "18rem" }}>
        <VStack gap={1}>
          <Heading level={2}>{title}</Heading>
          {description ? (
            <Text type="supporting" color="secondary">
              {description}
            </Text>
          ) : null}
          {meta}
        </VStack>
      </StackItem>
      {actions ? (
        <HStack gap={2} vAlign="center">
          {actions}
        </HStack>
      ) : null}
    </HStack>
  );
}
