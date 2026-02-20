import { Link } from "react-router-dom";
import { lazy, Suspense, useEffect } from "react";
import { SurvivalPhase } from "@/components/game/SurvivalPhase";
import { LeaderboardPanel } from "@/components/game/LeaderboardPanel";
import { WhalePositionCard } from "@/components/game/WhalePositionCard";
import { SquadInfo } from "@/components/game/SquadInfo";
import { GameHeader } from "@/components/game/GameHeader";
import { useGame } from "@/context/GameContext";
import { Button } from "@/components/ui/button";

const PriceChart = lazy(() =>
  import("@/components/game/PriceChart").then((m) => ({
    default: m.PriceChart,
  })),
);

export default function Game() {
  const { state, clearRoom } = useGame();
  const room = state.currentRoom;
  const gameActive = room && room.state !== "finished";

  useEffect(() => {
    if (!gameActive) return;
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [gameActive]);

  if (state.forfeited) {
    return (
      <main className="min-h-screen bg-background flex items-center justify-center px-4">
        <div className="text-center space-y-4 max-w-sm">
          <div className="w-16 h-16 mx-auto rounded-full bg-destructive/10 border border-destructive/30 flex items-center justify-center">
            <span className="text-2xl">&#9760;</span>
          </div>
          <h1 className="text-2xl sm:text-3xl font-bold text-destructive">
            Forfeited
          </h1>
          <p className="text-muted-foreground">
            You left the room. Entry fee is lost.
          </p>
          <Link to="/rooms" onClick={clearRoom}>
            <Button className="bg-primary hover:bg-primary/90 min-h-[44px]">
              Back to Rooms
            </Button>
          </Link>
        </div>
      </main>
    );
  }

  if (state.selfEliminated && room) {
    return (
      <main className="min-h-screen bg-background flex items-center justify-center px-4">
        <div className="text-center space-y-4 max-w-sm">
          <div className="w-16 h-16 mx-auto rounded-full bg-destructive/10 border border-destructive/30 flex items-center justify-center">
            <span className="text-2xl">&#9889;</span>
          </div>
          <h1 className="text-2xl sm:text-3xl font-bold text-destructive">
            Eliminated
          </h1>
          <p className="text-muted-foreground">
            You were liquidated. Margin exceeded the threshold.
          </p>
          <div className="flex items-center justify-center gap-4 text-sm text-muted-foreground">
            <span>
              {room.alive}/{room.total} alive
            </span>
            <span>Pool: {room.pool}&#9733;</span>
          </div>
          <Link to="/rooms" onClick={clearRoom}>
            <Button className="bg-primary hover:bg-primary/90 min-h-[44px]">
              Back to Rooms
            </Button>
          </Link>
        </div>
      </main>
    );
  }

  if (!room) {
    return (
      <main className="min-h-screen bg-background flex items-center justify-center px-4">
        <div className="text-center space-y-4">
          <h1 className="text-2xl sm:text-3xl font-bold text-foreground">
            No game in progress
          </h1>
          <p className="text-muted-foreground">Join a room to start playing</p>
          <Link to="/rooms">
            <Button className="bg-primary hover:bg-primary/90 min-h-[44px]">
              Browse Rooms
            </Button>
          </Link>
        </div>
      </main>
    );
  }

  const isInDanger = room.margin_ratio >= 0.8;

  return (
    <main className="min-h-screen bg-background">
      <GameHeader />

      <div className="max-w-7xl mx-auto px-3 sm:px-4 py-4 sm:py-8">
        <div className="flex flex-col lg:grid lg:grid-cols-3 gap-4 sm:gap-6">
          <div className="lg:col-span-2 space-y-4 sm:space-y-6 order-1 lg:order-2">
            <div className="h-64 sm:h-80">
              <Suspense>
                <PriceChart
                  marginHistory={state.marginHistory}
                  volatilityMul={room.volatility_mul}
                  marginRatio={room.margin_ratio}
                  isInDanger={isInDanger}
                />
              </Suspense>
            </div>
            <WhalePositionCard />

            {room.state === "finished" && (
              <div className="rounded-lg border border-primary/50 bg-primary/5 p-6 text-center space-y-3">
                <h3 className="text-xl font-bold text-foreground">
                  Game Finished
                </h3>
                {room.winner_id ? (
                  <p className="text-muted-foreground">
                    Winner:{" "}
                    <span className="font-mono text-primary font-bold">
                      #{room.winner_id}
                    </span>
                  </p>
                ) : (
                  <p className="text-muted-foreground">No winner</p>
                )}
                <p className="text-sm text-muted-foreground">
                  Pool: {room.pool} Stars
                </p>
                <Link to="/rooms" onClick={clearRoom}>
                  <Button className="bg-primary hover:bg-primary/90 min-h-[44px]">
                    Play Again
                  </Button>
                </Link>
              </div>
            )}
          </div>

          <div className="space-y-4 sm:space-y-6 order-2 lg:order-1">
            <SquadInfo />
            <LeaderboardPanel />
          </div>
        </div>
      </div>

      <SurvivalPhase />
    </main>
  );
}
