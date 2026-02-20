import { Link } from "react-router-dom";
import { useGame } from "@/context/GameContext";
import { Button } from "@/components/ui/button";
import { TIERS } from "@/types/game";

export function GameHeader() {
  const { state, leaveRoom } = useGame();
  const room = state.currentRoom;
  if (!room) return null;

  const tier = TIERS[room.tier];
  const timerSec = Math.max(0, Math.ceil(room.timer_ms / 1000));
  const min = Math.floor(timerSec / 60);
  const sec = timerSec % 60;

  return (
    <div className="sticky top-0 z-40 border-b border-border/50 bg-background/80 backdrop-blur-sm safe-top">
      <div className="max-w-7xl mx-auto px-3 sm:px-4 py-3 sm:py-4 flex items-center justify-between">
        <div className="min-w-0 flex-1">
          <h1 className="font-bold text-foreground text-sm sm:text-base truncate">
            {room.type.toUpperCase()} T{room.tier}
          </h1>
          <p className="text-xs text-muted-foreground">
            {room.state} · {room.alive}/{room.total} alive
            {tier && ` · ${tier.entryCost}★`}
          </p>
        </div>

        <div className="flex items-center gap-3 sm:gap-4">
          <div className="text-center">
            <p className="text-[10px] sm:text-xs text-muted-foreground uppercase tracking-wide">
              Timer
            </p>
            <p
              className={`text-sm font-mono font-bold ${
                timerSec <= 10
                  ? "text-destructive animate-pulse"
                  : "text-primary"
              }`}
            >
              {String(min).padStart(2, "0")}:{String(sec).padStart(2, "0")}
            </p>
          </div>

          <Link to="/rooms" onClick={leaveRoom}>
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground hover:text-foreground min-h-[40px]"
            >
              Exit
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}
