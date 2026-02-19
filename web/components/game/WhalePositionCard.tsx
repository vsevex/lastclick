"use client";

import { useGame } from "@/context/GameContext";
import { TIERS } from "@/types/game";

export function WhalePositionCard() {
  const { state } = useGame();
  const room = state.currentRoom;
  if (!room) return null;

  const tier = TIERS[room.tier];
  const isInDanger = room.margin_ratio >= 0.8;
  const timerSec = Math.max(0, Math.ceil(room.timer_ms / 1000));

  return (
    <div
      className={`rounded-lg border transition-colors duration-300 p-4 sm:p-6 ${
        isInDanger
          ? "border-destructive/50 bg-destructive/5"
          : "border-border/50 bg-card/50 backdrop-blur-sm"
      }`}
    >
      <div className="grid grid-cols-2 gap-4 sm:gap-6">
        <div>
          <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1 sm:mb-2">
            Margin Ratio
          </p>
          <p className="text-xl sm:text-3xl font-bold text-foreground font-mono">
            {(room.margin_ratio * 100).toFixed(1)}%
          </p>
        </div>

        <div>
          <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1 sm:mb-2">
            Volatility Mul
          </p>
          <p className="text-xl sm:text-3xl font-bold text-accent font-mono">
            {room.volatility_mul.toFixed(2)}x
          </p>
        </div>

        <div>
          <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1 sm:mb-2">
            Timer
          </p>
          <p
            className={`text-lg sm:text-2xl font-bold font-mono ${
              timerSec <= 10 ? "text-destructive" : "text-primary"
            }`}
          >
            {timerSec}s
          </p>
        </div>

        <div>
          <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1 sm:mb-2">
            Pool
          </p>
          <p className="text-lg sm:text-2xl font-bold text-foreground font-mono">
            {room.pool} &#9733;
          </p>
        </div>
      </div>

      {isInDanger && (
        <div className="mt-3 sm:mt-4 pt-3 sm:pt-4 border-t border-destructive/30">
          <p className="text-xs sm:text-sm text-destructive font-semibold animate-pulse text-center">
            APPROACHING LIQUIDATION
            {tier && ` - PULSE WITHIN ${tier.pulseWindowSec}s OR ELIMINATE`}
          </p>
        </div>
      )}
    </div>
  );
}
