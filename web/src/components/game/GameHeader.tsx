import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useGame } from "@/context/GameContext";
import { Button } from "@/components/ui/button";
import { TIERS } from "@/types/game";

export function GameHeader() {
  const { state, forfeit, clearRoom, engine } = useGame();
  const navigate = useNavigate();
  const room = state.currentRoom;
  if (!room) return null;

  const tier = TIERS[room.tier];
  const timerSec = Math.max(0, Math.ceil(room.timer_ms / 1000));
  const min = Math.floor(timerSec / 60);
  const sec = timerSec % 60;
  const isSurvival = room.state === "survival";
  const isSpectator =
    state.selfEliminated && engine?.voluntaryExit && room.state === "survival";

  const handleExit = useCallback(() => {
    if (room.state === "finished" || isSpectator) {
      clearRoom();
      navigate("/rooms");
      return;
    }

    const msg = isSurvival
      ? "Leaving during survival = immediate elimination. Entry fee is lost. Are you sure?"
      : "Leaving the room will forfeit your entry fee. Are you sure?";

    if (window.confirm(msg)) {
      forfeit();
    }
  }, [forfeit, clearRoom, navigate, room.state, isSurvival, isSpectator]);

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

          <Button
            variant="ghost"
            size="sm"
            onClick={handleExit}
            className={`min-h-[40px] ${
              isSpectator
                ? "text-muted-foreground hover:text-foreground"
                : isSurvival
                  ? "text-destructive hover:text-destructive hover:bg-destructive/10"
                  : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {isSpectator ? "Back to rooms" : isSurvival ? "Forfeit" : "Exit"}
          </Button>
        </div>
      </div>
    </div>
  );
}
