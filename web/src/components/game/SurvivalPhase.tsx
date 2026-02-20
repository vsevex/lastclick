import { useGame } from "@/context/GameContext";
import { useTelegram } from "@/context/TelegramProvider";
import { TIERS } from "@/types/game";
import { Button } from "@/components/ui/button";
import { useState, useEffect, useRef } from "react";

/** Cooldown after accepted pulse (matches server rate limit). No client authority. */
const PULSE_COOLDOWN_MS = 500;
/** How long to show "Pulsed!" after ack. */
const PULSE_CONFIRM_DURATION_MS = 1200;
/** How long to show pulse impact line (+1 Stability, efficiency). */
const PULSE_IMPACT_DURATION_MS = 2200;

/** Risk zones from margin_ratio (no raw decimals). */
type RiskZone = "safe" | "risk" | "critical";
function riskZone(marginRatio: number): RiskZone {
  if (marginRatio >= 0.8) return "critical";
  if (marginRatio >= 0.5) return "risk";
  return "safe";
}

/** Client-side efficiency proxy: (timeSurvived * volMul) / entryCost. */
function efficiencyProxy(
  survivalTimeSec: number,
  timerMs: number,
  volMul: number,
  entryCost: number,
): number {
  const timeSurvivedSec = Math.max(0, survivalTimeSec - timerMs / 1000);
  if (entryCost <= 0) return 0;
  return (timeSurvivedSec * volMul) / entryCost;
}

/** Efficiency delta from one pulse extension. */
function efficiencyDelta(
  extensionSec: number,
  volMul: number,
  entryCost: number,
): number {
  if (entryCost <= 0) return 0;
  return (extensionSec * volMul) / entryCost;
}

export function SurvivalPhase() {
  const { state, pulse } = useGame();
  const { userId } = useTelegram();
  const room = state.currentRoom;
  const [cooldownUntil, setCooldownUntil] = useState(0);
  const [confirmUntil, setConfirmUntil] = useState(0);
  const [pulseImpactUntil, setPulseImpactUntil] = useState(0);
  const [lastPulseEfficiencyDelta, setLastPulseEfficiencyDelta] = useState(0);
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
    setPulseImpactUntil(now + PULSE_IMPACT_DURATION_MS);
    const tier = room ? TIERS[room.tier] : null;
    if (ack.extension_ms != null && tier) {
      const extSec = ack.extension_ms / 1000;
      const delta = efficiencyDelta(
        extSec,
        room!.volatility_mul,
        tier.entryCost,
      );
      setLastPulseEfficiencyDelta(delta);
    }
    try {
      window.Telegram?.WebApp?.HapticFeedback?.impactOccurred?.("light");
    } catch {
      if (navigator.vibrate) navigator.vibrate(10);
    }
  }, [isMyAck, ackKey, ack?.extension_ms, room?.volatility_mul, room?.tier]);

  const now = Date.now();
  const onCooldown = now < cooldownUntil;
  const showConfirmed = now < confirmUntil && isMyAck;
  const showPulseImpact = now < pulseImpactUntil && isMyAck;

  if (!room || room.state !== "survival") return null;

  const wasEliminated = userId ? state.eliminated.includes(userId) : false;
  if (wasEliminated) return null;

  const tier = TIERS[room.tier];
  const timerSec = Math.max(0, Math.ceil(room.timer_ms / 1000));
  const timerDisplaySec = Math.max(0, room.timer_ms / 1000);
  const isUrgent = timerSec <= 5;
  const zone = riskZone(room.margin_ratio);
  const efficiency =
    tier &&
    efficiencyProxy(
      tier.survivalTimeSec,
      room.timer_ms,
      room.volatility_mul,
      tier.entryCost,
    );

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

        {/* A. Personal survival status — competitive, not reactive */}
        <div className="space-y-1.5 text-sm text-muted-foreground">
          <p>
            Next required pulse in:{" "}
            <span className="font-mono text-foreground">
              {timerDisplaySec.toFixed(1)}s
            </span>
          </p>
          {efficiency != null && (
            <p>
              Your Efficiency:{" "}
              <span className="font-mono text-foreground">
                {efficiency.toFixed(1)}
              </span>
            </p>
          )}
          <p>
            You are one of{" "}
            <span className="font-mono text-foreground">{room.alive}</span> left
          </p>
        </div>

        {/* B. Risk indicator — pressure without raw decimals */}
        <div
          className={`rounded-lg px-3 py-2 text-sm font-semibold ${
            zone === "safe"
              ? "bg-green-500/20 text-green-700 dark:text-green-400"
              : zone === "risk"
                ? "bg-yellow-500/20 text-yellow-700 dark:text-yellow-400"
                : "bg-red-500/20 text-red-700 dark:text-red-400"
          }`}
        >
          {zone === "safe" && "Safe Zone"}
          {zone === "risk" && "Risk Zone"}
          {zone === "critical" && "Critical Zone"}
        </div>

        {/* C. Pulse impact — brief feedback on my pulse */}
        {showPulseImpact && (
          <div className="rounded-lg bg-green-500/15 border border-green-500/30 px-3 py-2 text-sm space-y-0.5 animate-in fade-in duration-200">
            <p className="text-green-700 dark:text-green-400 font-medium">
              +1 Stability
            </p>
            {lastPulseEfficiencyDelta > 0 && (
              <p className="text-green-600/90 dark:text-green-500/90 text-xs">
                Efficiency adjusted: +{lastPulseEfficiencyDelta.toFixed(1)}
              </p>
            )}
          </div>
        )}

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
