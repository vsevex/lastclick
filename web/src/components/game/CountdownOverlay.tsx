import { useGame } from "@/context/GameContext";

/** Shown during COUNTDOWN: "Now it's real" — chart reset, join locked at 0. */
export function CountdownOverlay() {
  const { state } = useGame();
  const room = state.currentRoom;
  if (!room || room.state !== "active") return null;

  const countdownSec = Math.max(0, Math.ceil(room.timer_ms / 1000));

  return (
    <div
      className="fixed inset-0 z-40 pointer-events-none flex items-center justify-center px-4 bg-background/60 backdrop-blur-[2px] animate-in fade-in duration-300"
      aria-live="polite"
    >
      <div className="rounded-xl border border-primary/30 bg-card/95 px-6 py-8 text-center shadow-lg">
        <p className="text-xs uppercase tracking-widest text-muted-foreground mb-2">
          Round starting
        </p>
        <p className="text-5xl sm:text-6xl font-bold font-mono text-primary tabular-nums">
          {countdownSec}
        </p>
        <p className="text-sm text-muted-foreground mt-3">
          Margin set to baseline · Join locked at 0
        </p>
      </div>
    </div>
  );
}
