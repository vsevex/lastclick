"use client";

import { useGame } from "@/context/GameContext";
import { useTelegram } from "@/context/TelegramProvider";

export function LeaderboardPanel() {
  const { state } = useGame();
  const { userId } = useTelegram();
  const room = state.currentRoom;

  if (!room) return null;

  const wasEliminated = userId ? state.eliminated.includes(userId) : false;

  return (
    <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4 sm:p-6">
      <div className="mb-4 sm:mb-6">
        <h2 className="text-lg sm:text-xl font-bold text-foreground mb-1 sm:mb-2">
          Survivors
        </h2>
        <p className="text-sm text-muted-foreground">
          {room.alive} alive / {room.total} total
        </p>
      </div>

      <div className="mb-4 sm:mb-6">
        <div className="w-full bg-background/50 rounded-full h-3 overflow-hidden border border-border/30">
          <div
            className="h-full bg-linear-to-r from-primary via-accent to-destructive transition-all duration-300"
            style={{
              width: `${room.total > 0 ? (room.alive / room.total) * 100 : 100}%`,
            }}
          />
        </div>
        <p className="text-xs text-muted-foreground mt-2">
          {room.total > 0 ? Math.round((room.alive / room.total) * 100) : 0}%
          survival rate
        </p>
      </div>

      {wasEliminated && (
        <div className="p-3 rounded-lg bg-destructive/10 border border-destructive/30 mb-4">
          <p className="text-sm font-semibold text-destructive text-center">
            YOU WERE ELIMINATED
          </p>
        </div>
      )}

      {state.eliminated.length > 0 && (
        <div className="space-y-1.5">
          <p className="text-xs text-muted-foreground uppercase tracking-wide mb-2">
            Recent Eliminations
          </p>
          {state.eliminated
            .slice(-5)
            .reverse()
            .map((pid) => (
              <div
                key={pid}
                className="flex items-center gap-2 p-2 rounded bg-background/50 border border-border/30"
              >
                <div className="w-2 h-2 rounded-full bg-destructive" />
                <span className="text-xs text-muted-foreground font-mono">
                  #{pid}
                </span>
                <span className="text-xs text-destructive ml-auto">
                  eliminated
                </span>
              </div>
            ))}
        </div>
      )}
    </div>
  );
}
