import { HStack } from "@astryxdesign/core/Stack";
import { StatusDot, type StatusDotVariant } from "@astryxdesign/core/StatusDot";
import { Text } from "@astryxdesign/core/Text";

interface StatusLabelProps {
  variant: StatusDotVariant;
  label: string;
  isPulsing?: boolean;
}

/**
 * StatusDot with a visible text label — StatusDot alone only exposes its
 * label to screen readers. Use in table status columns and detail views.
 */
export function StatusLabel({ variant, label, isPulsing }: StatusLabelProps) {
  return (
    <HStack gap={2} vAlign="center">
      <StatusDot variant={variant} label={label} isPulsing={isPulsing} />
      <Text type="supporting">{label}</Text>
    </HStack>
  );
}
