import { Link } from "react-router-dom";
import { lazy, Suspense, useEffect, useState } from "react";
import { SurvivalPhase } from "@/components/game/SurvivalPhase";
import { TIERS } from "@/types/game";
import { TutorialTooltips } from "@/components/game/TutorialTooltips";
import { CountdownOverlay } from "@/components/game/CountdownOverlay";
import { LeaderboardPanel } from "@/components/game/LeaderboardPanel";
import { WhalePositionCard } from "@/components/game/WhalePositionCard";
import { SquadInfo } from "@/components/game/SquadInfo";
import { GameHeader } from "@/components/game/GameHeader";
import { SimulationPanel } from "@/components/game/SimulationPanel";
import { useGame } from "@/context/GameContext";
import { Button } from "@/components/ui/button";

const PriceChart = lazy(() =>
  import("@/components/game/PriceChart").then((m) => ({
    default: m.PriceChart,
  })),
);

const NEXT_ROUND_DELAY_SEC = 12;

export default function Game() {
  const { state, clearRoom, engine, joinRoom } = useGame();
  const room = state.currentRoom;
  const gameActive = room && room.state !== "finished";
  const [nextRoundSec, setNextRoundSec] = useState(NEXT_ROUND_DELAY_SEC);

  // Countdown for "Next Round Starts in Xs" when round is finished
  useEffect(() => {
    if (room?.state !== "finished" || room?.tier === 0) return;
    setNextRoundSec(NEXT_ROUND_DELAY_SEC);
    const id = setInterval(() => {
      setNextRoundSec((s) => (s <= 0 ? 0 : s - 1));
    }, 1000);
    return () => clearInterval(id);
  }, [room?.state, room?.tier, room?.room_id]);

  useEffect(() => {
    if (!gameActive) return;
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [gameActive]);

  if (state.forfeited) {
    const shardInfo = engine?.shardCredit;
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
          {shardInfo != null && shardInfo > 0 && (
            <p className="text-sm text-accent font-semibold">
              +{shardInfo} Shards earned
            </p>
          )}
          <Link to="/rooms" onClick={clearRoom}>
            <Button className="bg-primary hover:bg-primary/90 min-h-[44px]">
              Back to Rooms
            </Button>
          </Link>
        </div>
        <SimulationPanel />
      </main>
    );
  }

  // Voluntary exit during survival → spectator (stay on game view). Liquidated → full eliminated screen.
  const isSpectator =
    state.selfEliminated && engine?.voluntaryExit && room?.state === "survival";
  if (state.selfEliminated && room && !isSpectator) {
    const shardInfo = engine?.shardCredit;
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
          {shardInfo != null && shardInfo > 0 && (
            <p className="text-sm text-accent font-semibold">
              +{shardInfo} Shards earned
            </p>
          )}
          <Link to="/rooms" onClick={clearRoom}>
            <Button className="bg-primary hover:bg-primary/90 min-h-[44px]">
              Back to Rooms
            </Button>
          </Link>
        </div>
        <SimulationPanel />
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
        <SimulationPanel />
      </main>
    );
  }

  const isInDanger = room.margin_ratio >= 0.8;
  const payoutInfo = engine?.payoutInfo;

  return (
    <main className="min-h-screen bg-background">
      <GameHeader />
      {isSpectator && (
        <div className="sticky top-[57px] z-30 mx-auto max-w-7xl px-3 sm:px-4 py-2 bg-muted/80 border-b border-border/50 text-center text-sm text-muted-foreground">
          You forfeited. Spectating until round ends. Rejoin allowed next round
          only.
        </div>
      )}

      <div className="max-w-7xl mx-auto px-3 sm:px-4 py-4 sm:py-8">
        <div className="flex flex-col lg:grid lg:grid-cols-3 gap-4 sm:gap-6">
          <div className="lg:col-span-2 space-y-4 sm:space-y-6 order-1 lg:order-2">
            <div
              className={`h-64 sm:h-80 transition-colors duration-300 ${
                room.state === "active"
                  ? "rounded-lg border border-primary/20 bg-primary/5"
                  : ""
              }`}
            >
              <Suspense>
                <PriceChart
                  key={`${room.room_id}-${room.state === "waiting" ? "waiting" : "round"}`}
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
                {room.tier === 0 ? (
                  <>
                    <h3 className="text-xl font-bold text-foreground">
                      Demo complete
                    </h3>
                    <p className="text-lg font-bold text-green-400">
                      You placed 2nd.
                    </p>
                    <p className="text-2xl font-bold text-primary font-mono">
                      +12 &#9733;{" "}
                      <span className="text-sm font-normal text-muted-foreground">
                        (example payout)
                      </span>
                    </p>
                    <p className="text-xs text-muted-foreground max-w-sm mx-auto">
                      Efficiency = survival time × volatility ÷ entry. Higher =
                      better payout potential.
                    </p>
                    <Link
                      to="/rooms"
                      onClick={() => {
                        try {
                          localStorage.setItem("lastclick_tutorial_done", "1");
                        } catch {}
                        clearRoom();
                      }}
                    >
                      <Button className="w-full bg-primary hover:bg-primary/90 min-h-[44px]">
                        Play real rooms
                      </Button>
                    </Link>
                  </>
                ) : (
                  <>
                    <h3 className="text-xl font-bold text-foreground">
                      Game Finished
                    </h3>
                    {(state.roundResult?.placement ?? payoutInfo?.rank) !=
                      null && (
                      <p className="text-lg font-bold text-green-400">
                        #{state.roundResult?.placement ?? payoutInfo?.rank}{" "}
                        Place
                      </p>
                    )}
                    {payoutInfo && (
                      <p className="text-2xl font-bold text-primary font-mono">
                        +{payoutInfo.amount} &#9733;
                      </p>
                    )}
                    {(state.roundResult?.shards ?? engine?.shardCredit ?? 0) >
                      0 && (
                      <p className="text-sm text-accent font-semibold">
                        +{state.roundResult?.shards ?? engine?.shardCredit}{" "}
                        Shards earned
                      </p>
                    )}
                    <p className="text-sm text-muted-foreground">
                      Next Round Starts in {nextRoundSec}s
                    </p>
                    <div className="flex flex-col sm:flex-row gap-2 justify-center">
                      <Button
                        className="bg-primary hover:bg-primary/90 min-h-[44px]"
                        disabled={nextRoundSec > 0}
                        onClick={() => {
                          if (!room) return;
                          engine?.resetRound?.();
                          joinRoom(room.room_id);
                        }}
                      >
                        Re-enter ({TIERS[room?.tier ?? 1]?.entryCost ?? 5}⭐)
                      </Button>
                      <Link to="/rooms" onClick={clearRoom}>
                        <Button variant="outline" className="min-h-[44px]">
                          Back to rooms
                        </Button>
                      </Link>
                    </div>
                  </>
                )}
              </div>
            )}
          </div>

          <div className="space-y-4 sm:space-y-6 order-2 lg:order-1">
            <SquadInfo />
            <LeaderboardPanel />
          </div>
        </div>
      </div>

      <CountdownOverlay />
      <TutorialTooltips />
      <SurvivalPhase />
      <SimulationPanel />
    </main>
  );
}
