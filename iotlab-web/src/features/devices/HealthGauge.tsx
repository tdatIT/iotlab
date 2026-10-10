import type { ReactNode } from "react";
import { Card } from "@astryxdesign/core/Card";
import { Skeleton } from "@astryxdesign/core/Skeleton";
import { HStack, VStack } from "@astryxdesign/core/Stack";
import { Text } from "@astryxdesign/core/Text";
import { Cell, Label, Pie, PieChart } from "recharts";

import { useIsMobile } from "@/lib/useMediaQuery";

export type GaugeTone = "success" | "warning" | "error" | "accent" | "neutral";

const TONE_COLOR: Record<GaugeTone, string> = {
  success: "var(--color-success, #0B991F)",
  warning: "var(--color-warning, #D97706)",
  error: "var(--color-error, #DC2626)",
  accent: "var(--color-text-blue, #0064E0)",
  neutral: "var(--color-text-secondary, #56606B)",
};
const TRACK_COLOR = "var(--color-border, #E5E8EB)";

interface HealthGaugeProps {
  label: string;
  icon?: ReactNode;
  /** Filled share of the ring, 0–100. */
  percent: number;
  tone: GaugeTone;
  /** Big text in the middle of the ring. */
  value: string;
  /** Small text under the value, inside the ring. */
  caption?: string;
  /** Line under the chart. */
  hint?: ReactNode;
  isLoading?: boolean;
}

/** Ring diameter: desktop / mobile (two gauges per row). */
const SIZE = { desktop: 148, mobile: 120 };

/** Health KPI as a donut: ring = percent, centre = value + caption. */
export function HealthGauge({
  label,
  icon,
  percent,
  tone,
  value,
  caption,
  hint,
  isLoading,
}: HealthGaugeProps) {
  const isMobile = useIsMobile();
  const size = isMobile ? SIZE.mobile : SIZE.desktop;
  const filled = Math.max(0, Math.min(100, percent));
  const data = [
    { key: "value", v: filled },
    { key: "rest", v: 100 - filled },
  ];

  return (
    <Card padding={isMobile ? 3 : 4}>
      <VStack gap={2} hAlign="center">
        <HStack gap={2} vAlign="center" hAlign="center">
          {icon}
          <Text type="supporting" color="secondary">
            {label}
          </Text>
        </HStack>
        {isLoading ? (
          <Skeleton width={size} height={size} radius="rounded" />
        ) : (
          <VStack width={size} height={size}>
            <PieChart width={size} height={size}>
              <Pie
                data={data}
                dataKey="v"
                startAngle={90}
                endAngle={-270}
                innerRadius={isMobile ? "80%" : "74%"}
                outerRadius="100%"
                stroke="none"
                isAnimationActive={false}
              >
                <Cell fill={TONE_COLOR[tone]} />
                <Cell fill={TRACK_COLOR} />
                <Label
                  position="center"
                  // Fixed-size chart, so the centre is known — recharts 3 passes no polar viewBox here.
                  content={() => {
                    const cx = size / 2;
                    const cy = size / 2;
                    return (
                      <text x={cx} y={cy} textAnchor="middle">
                        <tspan
                          x={cx}
                          dy={caption ? "-0.1em" : "0.35em"}
                          fontSize={isMobile ? 15 : 18}
                          fontWeight={600}
                          fill="var(--color-text-primary, #1C2024)"
                        >
                          {value}
                        </tspan>
                        {caption ? (
                          <tspan
                            x={cx}
                            dy="1.5em"
                            fontSize={isMobile ? 10 : 11}
                            fill="var(--color-text-secondary, #56606B)"
                          >
                            {caption}
                          </tspan>
                        ) : null}
                      </text>
                    );
                  }}
                />
              </Pie>
            </PieChart>
        </VStack>
      )}
        {hint && !isLoading ? (
          <Text type="supporting" color="secondary" justify="center">
            {hint}
          </Text>
        ) : null}
      </VStack>
    </Card>
  );
}
