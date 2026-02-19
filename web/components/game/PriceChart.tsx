"use client";

import { useMemo } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface Props {
  marginHistory: number[];
  volatilityMul: number;
  marginRatio: number;
  isInDanger: boolean;
}

export function PriceChart({
  marginHistory,
  volatilityMul,
  marginRatio,
  isInDanger,
}: Props) {
  const chartData = useMemo(
    () =>
      marginHistory.map((v, i) => ({ t: i, margin: +(v * 100).toFixed(2) })),
    [marginHistory],
  );

  return (
    <div
      className={`w-full h-full rounded-lg border transition-colors duration-300 p-3 sm:p-4 ${
        isInDanger
          ? "border-destructive/50 bg-destructive/5 pulse-danger"
          : "border-border/50 bg-background/50"
      }`}
    >
      <div className="mb-3 sm:mb-4">
        <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">
          Margin Ratio
        </p>
        <div className="flex items-baseline gap-2">
          <p className="text-xl sm:text-2xl md:text-3xl font-bold text-foreground">
            {(marginRatio * 100).toFixed(1)}%
          </p>
          <p className="text-xs font-semibold text-muted-foreground">
            Vol: {volatilityMul.toFixed(2)}x
          </p>
        </div>
      </div>

      <ResponsiveContainer width="100%" height="55%">
        <LineChart
          data={chartData}
          margin={{ top: 5, right: 5, left: -25, bottom: 0 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" opacity={0.3} />
          <XAxis dataKey="t" tick={false} axisLine={{ stroke: "#2D3748" }} />
          <YAxis
            domain={[0, 100]}
            tick={false}
            axisLine={{ stroke: "#2D3748" }}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: "#1A1F2E",
              border: "1px solid #2D3748",
              borderRadius: "8px",
              fontSize: "12px",
            }}
            formatter={(value: number) => [`${value}%`, "Margin"]}
          />
          <Line
            type="monotone"
            dataKey="margin"
            stroke={isInDanger ? "#EF4444" : "#10B981"}
            strokeWidth={2}
            dot={false}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>

      <div className="mt-3 sm:mt-4 flex items-center justify-between text-xs">
        <div className="flex items-center gap-1.5">
          <div className="w-2 h-2 rounded-full bg-destructive" />
          <span className="text-muted-foreground">Liquidation at 100%</span>
        </div>
        <span
          className={`font-semibold ${
            marginRatio >= 0.8
              ? "text-destructive"
              : marginRatio >= 0.5
                ? "text-accent"
                : "text-primary"
          }`}
        >
          {(marginRatio * 100).toFixed(1)}%
        </span>
      </div>
    </div>
  );
}
