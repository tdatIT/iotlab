import { Card } from "@astryxdesign/core/Card";
import { Skeleton } from "@astryxdesign/core/Skeleton";
import { VStack } from "@astryxdesign/core/Stack";
import { Heading, Text } from "@astryxdesign/core/Text";

interface StatCardProps {
  label: string;
  value: string;
  /** Show a skeleton instead of the value. */
  isLoading?: boolean;
}

/** KPI tile for dashboard grids. Compose inside a Grid with repeat: 'fit'. */
export function StatCard({ label, value, isLoading }: StatCardProps) {
  return (
    <Card padding={3}>
      <VStack gap={1}>
        <Text type="supporting" color="secondary">
          {label}
        </Text>
        {isLoading ? <Skeleton height={28} width="60%" /> : <Heading level={3}>{value}</Heading>}
      </VStack>
    </Card>
  );
}
