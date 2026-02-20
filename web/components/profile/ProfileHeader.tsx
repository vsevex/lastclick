"use client";

import { useGame } from "@/context/GameContext";

export function ProfileHeader() {
  const { state } = useGame();
  const player = state.player;
  if (!player) return null;

  return (
    <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-5 sm:p-8">
      <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4 sm:gap-6">
        <div className="w-16 h-16 sm:w-24 sm:h-24 rounded-xl bg-linear-to-br from-primary/50 to-accent/50 flex items-center justify-center text-2xl sm:text-4xl font-bold text-foreground border-2 border-border/50">
          {player.Username[0]?.toUpperCase() ?? "?"}
        </div>

        <div className="flex-1">
          <h1 className="text-2xl sm:text-3xl md:text-4xl font-bold text-foreground mb-1 sm:mb-2">
            {player.Username}
          </h1>
          <p className="text-muted-foreground mb-3 sm:mb-4">
            Elo {player.Elo} · Lifetime {player.LifetimeElo}
          </p>

          <div className="flex flex-wrap gap-4 sm:gap-6">
            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide">
                Stars
              </p>
              <p className="text-xl sm:text-2xl font-bold text-primary">
                {player.StarsBalance}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide">
                Shards
              </p>
              <p className="text-xl sm:text-2xl font-bold text-accent">
                {player.ShardsBalance}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide">
                Avg Efficiency
              </p>
              <p className="text-xl sm:text-2xl font-bold text-foreground">
                {player.EfficiencyAvg.toFixed(1)}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
