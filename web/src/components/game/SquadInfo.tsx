import { useGame } from "@/context/GameContext";
import { useTelegram } from "@/context/TelegramProvider";

export function SquadInfo() {
  const { state } = useGame();
  const { userId, username } = useTelegram();
  const room = state.currentRoom;
  const player = state.player;

  if (!room) return null;

  const wasEliminated = userId ? state.eliminated.includes(userId) : false;

  return (
    <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4 sm:p-6">
      <div className="flex items-center gap-3 mb-4 sm:mb-6">
        <div className="w-10 h-10 sm:w-12 sm:h-12 rounded-full bg-linear-to-br from-primary/50 to-accent/50 flex items-center justify-center text-sm font-bold text-foreground">
          {(username ?? "?")[0].toUpperCase()}
        </div>
        <div>
          <h3 className="font-bold text-foreground text-sm sm:text-base">
            {username ?? `Player #${userId ?? "?"}`}
          </h3>
          <p className="text-xs text-muted-foreground">
            Elo: {player?.Elo ?? "—"}
          </p>
        </div>
      </div>

      <div className="space-y-3 border-t border-border/30 pt-3 sm:pt-4">
        <div className="flex items-center justify-between">
          <p className="text-xs text-muted-foreground uppercase tracking-wide">
            Stars
          </p>
          <p className="text-sm font-bold text-primary font-mono">
            {player?.StarsBalance ?? "—"}
          </p>
        </div>

        <div className="flex items-center justify-between">
          <p className="text-xs text-muted-foreground uppercase tracking-wide">
            Shards
          </p>
          <p className="text-sm font-bold text-accent font-mono">
            {player?.ShardsBalance ?? "—"}
          </p>
        </div>

        <div className="flex items-center justify-between">
          <p className="text-xs text-muted-foreground uppercase tracking-wide">
            Efficiency
          </p>
          <p className="text-sm font-bold text-foreground font-mono">
            {player?.EfficiencyAvg?.toFixed(1) ?? "—"}
          </p>
        </div>
      </div>

      <div className="mt-3 sm:mt-4 pt-3 sm:pt-4 border-t border-border/30">
        <div
          className={`inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold ${
            wasEliminated
              ? "bg-destructive/10 text-destructive"
              : "bg-primary/10 text-primary"
          }`}
        >
          <div
            className={`w-2 h-2 rounded-full ${wasEliminated ? "bg-destructive" : "bg-primary"}`}
          />
          {wasEliminated ? "LIQUIDATED" : "ACTIVE"}
        </div>
      </div>
    </div>
  );
}
