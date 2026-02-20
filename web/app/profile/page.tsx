"use client";

import { useEffect, useState } from "react";
import { ProfileHeader } from "@/components/profile/ProfileHeader";
import { NavBar } from "@/components/NavBar";
import { useGame } from "@/context/GameContext";
import { useTelegram } from "@/context/TelegramProvider";
import { getLeaderboardPlayers, getPlayerRank } from "@/lib/api";
import type { LeaderboardEntry } from "@/types/game";
import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function ProfilePage() {
  const { state, refreshPlayer } = useGame();
  const { userId } = useTelegram();
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [myRank, setMyRank] = useState<LeaderboardEntry | null>(null);

  useEffect(() => {
    if (userId && !state.player) refreshPlayer();
  }, [userId, state.player, refreshPlayer]);

  useEffect(() => {
    getLeaderboardPlayers(10)
      .then(setLeaderboard)
      .catch(() => {});
    if (userId) {
      getPlayerRank(userId)
        .then(setMyRank)
        .catch(() => {});
    }
  }, [userId]);

  return (
    <main className="min-h-screen bg-background">
      <NavBar title="Your Profile" />

      <div className="max-w-7xl mx-auto px-3 sm:px-4 py-6 sm:py-12">
        {state.player ? (
          <ProfileHeader />
        ) : (
          <div className="rounded-lg border border-border/50 bg-card/50 p-8 text-center">
            <p className="text-muted-foreground">Loading profile...</p>
          </div>
        )}

        <div className="grid sm:grid-cols-2 gap-4 sm:gap-6 mt-6 sm:mt-8">
          {/* Rank Card */}
          <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-5 sm:p-6">
            <h2 className="text-lg sm:text-xl font-bold text-foreground mb-5 sm:mb-6">
              Your Rank
            </h2>
            {myRank ? (
              <div className="space-y-4">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Season Rank</span>
                  <span className="font-bold text-primary">#{myRank.Rank}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">
                    Efficiency Score
                  </span>
                  <span className="font-bold text-accent">
                    {myRank.Score.toFixed(1)}
                  </span>
                </div>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                Play a game to get ranked this season
              </p>
            )}
            <Link href="/rooms" className="w-full mt-4 block">
              <Button className="w-full bg-primary/80 hover:bg-primary min-h-[44px]">
                Play Now
              </Button>
            </Link>
          </div>

          {/* Quick Stats */}
          <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-5 sm:p-6">
            <h2 className="text-lg sm:text-xl font-bold text-foreground mb-5 sm:mb-6">
              Account
            </h2>
            <div className="space-y-3">
              <div className="p-3 rounded-lg bg-background/50 border border-border/30">
                <p className="text-xs text-muted-foreground mb-1 uppercase tracking-wide">
                  Prestige Multiplier
                </p>
                <p className="font-semibold text-foreground">
                  {state.player?.PrestigeMult?.toFixed(1) ?? "1.0"}x
                </p>
              </div>
              <div className="p-3 rounded-lg bg-background/50 border border-border/30">
                <p className="text-xs text-muted-foreground mb-1 uppercase tracking-wide">
                  Squad
                </p>
                <p className="font-semibold text-foreground">
                  {state.player?.SquadID ?? "None"}
                </p>
              </div>
            </div>
            <Link href="/store" className="w-full mt-4 block">
              <Button className="w-full bg-primary/80 hover:bg-primary min-h-[44px]">
                Visit Store
              </Button>
            </Link>
          </div>
        </div>

        {/* Leaderboard */}
        <div className="mt-8 sm:mt-12">
          <h2 className="text-xl sm:text-2xl font-bold text-foreground mb-4 sm:mb-6">
            Season Leaderboard
          </h2>

          {leaderboard.length > 0 ? (
            <>
              {/* Mobile */}
              <div className="sm:hidden space-y-3">
                {leaderboard.map((entry) => (
                  <div
                    key={entry.PlayerID}
                    className="flex items-center gap-3 p-3 rounded-lg border border-border/50 bg-card/50"
                  >
                    <span className="text-sm font-bold text-primary w-8">
                      #{entry.Rank}
                    </span>
                    <div className="flex-1 min-w-0">
                      <p className="font-semibold text-foreground text-sm truncate font-mono">
                        #{entry.PlayerID}
                      </p>
                    </div>
                    <span className="text-sm font-mono text-accent font-bold">
                      {entry.Score.toFixed(1)}
                    </span>
                  </div>
                ))}
              </div>

              {/* Desktop */}
              <div className="hidden sm:block rounded-lg border border-border/50 bg-card/50 overflow-hidden">
                <table className="w-full">
                  <thead className="border-b border-border/30 bg-background/50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wide">
                        Rank
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wide">
                        Player
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wide">
                        Efficiency
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border/30">
                    {leaderboard.map((entry) => (
                      <tr
                        key={entry.PlayerID}
                        className={`hover:bg-background/50 transition-colors ${
                          entry.PlayerID === userId ? "bg-primary/5" : ""
                        }`}
                      >
                        <td className="px-6 py-4 text-sm font-bold text-primary">
                          #{entry.Rank}
                        </td>
                        <td className="px-6 py-4 text-sm font-mono text-foreground">
                          #{entry.PlayerID}
                          {entry.PlayerID === userId && (
                            <span className="ml-2 text-xs text-primary">
                              (you)
                            </span>
                          )}
                        </td>
                        <td className="px-6 py-4 text-sm font-mono text-accent font-bold">
                          {entry.Score.toFixed(1)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          ) : (
            <div className="text-center py-12 rounded-lg border border-border/50 bg-card/50">
              <p className="text-muted-foreground">
                No leaderboard data yet this season
              </p>
            </div>
          )}
        </div>
      </div>
    </main>
  );
}
