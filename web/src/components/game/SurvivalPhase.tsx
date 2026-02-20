import { useGame } from "@/context/GameContext";
import { useTelegram } from "@/context/TelegramProvider";
import { TIERS } from "@/types/game";
import { Button } from "@/components/ui/button";
import { useState, useEffect, useRef } from "react";

/** Cooldown after accepted pulse (matches server rate limit). No client authority. */
const PULSE_COOLDOWN_MS = 500;
/** How long to show "Pulsed!" after ack. */
const PULSE_CONFIRM_DURATION_MS = 1200;

export function SurvivalPhase() {
  const { state, pulse } = useGame();
  const { userId } = useTelegram();
  const room = state.currentRoom;
  const [cooldownUntil, setCooldownUntil] = useState(0);
  const [confirmUntil, setConfirmUntil] = useState(0);
  const lastHandledAckRef = useRef<string | null>(null);

  const ack = state.lastPulseAck;
  const isMyAck = ack && userId != null && ack.player_id === userId;
  const ackKey = ack
    ? `${ack.player_id}-${ack.server_time_ms ?? ack.timer_ms}`
    : null;

  useEffect(() => {
    if (!isMyAck || !ackKey) return;
    if (lastHandledAckRef.current === ackKey) return;
    lastHandledAckRef.current = ackKey;
    const now = Date.now();
    setCooldownUntil(now + PULSE_COOLDOWN_MS);
    setConfirmUntil(now + PULSE_CONFIRM_DURATION_MS);
    try {
      window.Telegram?.WebApp?.HapticFeedback?.impactOccurred?.("light");
    } catch {
      if (navigator.vibrate) navigator.vibrate(10);
    }
  }, [isMyAck, ackKey]);

  const now = Date.now();
  const onCooldown = now < cooldownUntil;
  const showConfirmed = now < confirmUntil && isMyAck;

  if (!room || room.state !== "survival") return null;

  const wasEliminated = userId ? state.eliminated.includes(userId) : false;
  if (wasEliminated) return null;

  const tier = TIERS[room.tier];
  const timerSec = Math.max(0, Math.ceil(room.timer_ms / 1000));
  const isUrgent = timerSec <= 5;

  const handlePulse = () => {
    if (onCooldown) return;
    pulse();
  };

  return (
    <div className="fixed inset-0 z-50 pointer-events-none flex items-center justify-center px-4">
      <div className="absolute inset-0 bg-destructive/10 backdrop-blur-sm pointer-events-auto" />

      <div className="relative pointer-events-auto space-y-6 sm:space-y-8 text-center w-full max-w-sm">
        <div className="space-y-3 sm:space-y-4">
          <p className="text-xs sm:text-sm text-destructive/80 uppercase tracking-widest font-semibold">
            SURVIVAL PHASE
          </p>
          <div className="flex justify-center">
            <div
              className={`text-6xl sm:text-7xl md:text-8xl font-bold font-mono ${
                isUrgent ? "text-destructive animate-pulse" : "text-accent"
              }`}
            >
              {timerSec}
            </div>
          </div>
          <p className="text-muted-foreground text-sm">
            {room.alive} players alive · Vol {room.volatility_mul.toFixed(1)}x
          </p>
        </div>

        <Button
          onClick={handlePulse}
          disabled={onCooldown}
          size="lg"
          className={`w-full max-w-xs mx-auto py-6 text-lg font-bold min-h-[56px] transition-all duration-200 ${
            showConfirmed
              ? "bg-green-600 hover:bg-green-600 text-white"
              : onCooldown
                ? "opacity-70 cursor-not-allowed bg-primary/80"
                : "bg-primary hover:bg-primary/90 text-primary-foreground shadow-lg shadow-primary/50 active:scale-95"
          }`}
        >
          {showConfirmed ? "Pulsed!" : onCooldown ? "..." : "PULSE NOW"}
        </Button>

        <p className="text-xs text-muted-foreground">
          Pulse within {tier?.pulseWindowSec ?? 5}s or get eliminated
        </p>

        {isUrgent && (
          <div className="pt-3 sm:pt-4 border-t border-destructive/30">
            <p className="text-destructive font-bold animate-pulse text-sm sm:text-base">
              CRITICAL - PULSE NOW
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
