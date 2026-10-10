import type { ReactNode } from "react";
import { Card } from "@astryxdesign/core/Card";
import { HStack, VStack, StackItem } from "@astryxdesign/core/Stack";
import { Heading, Text } from "@astryxdesign/core/Text";

interface ChartCardProps {
  title: string;
  description?: string;
  /** Right side of the card header (legend, range picker, ...). */
  actions?: ReactNode;
  /** Fixed chart region height — recharts ResponsiveContainer needs one. */
  height?: number;
  children: ReactNode;
}

/** Dashboard widget container: header row + fixed-height chart region. */
export function ChartCard({
  title,
  description,
  actions,
  height = 280,
  children,
}: ChartCardProps) {
  return (
    <Card padding={4}>
      <VStack gap={4}>
        <HStack gap={4} vAlign="center" wrap="wrap">
          {/* fit-content floor: actions wrap below the title instead of squeezing it */}
          <StackItem size="fill" style={{ minWidth: "fit-content" }}>
            <VStack gap={1}>
              <Heading level={4}>{title}</Heading>
              {description ? (
                <Text type="supporting" color="secondary">
                  {description}
                </Text>
              ) : null}
            </VStack>
          </StackItem>
          {actions}
        </HStack>
        <VStack height={height}>{children}</VStack>
      </VStack>
    </Card>
  );
}
