import { useState } from "react";
import { useGame } from "@/context/GameContext";
import { Button } from "@/components/ui/button";

export function SimulationPanel() {
  const { engine, state } = useGame();
  const [pulseWindow, setPulseWindow] = useState(5);
  const [lagMs, setLagMs] = useState(0);
  const [massCount, setMassCount] = useState(5);
  const [collapsed, setCollapsed] = useState(false);

  if (!engine?.isPrototype) return null;

  const inRoom = !!state.currentRoom;
  const isSurvival = state.currentRoom?.state === "survival";

  return (
    <div className="fixed bottom-20 right-3 z-60 w-72 max-h-[70vh] overflow-y-auto">
      <div className="rounded-lg border border-amber-500/40 bg-background/95 backdrop-blur-sm shadow-lg">
        <button
          onClick={() => setCollapsed((c) => !c)}
          className="w-full flex items-center justify-between px-3 py-2 text-xs font-bold text-amber-400 uppercase tracking-wider border-b border-amber-500/20"
        >
          <span>SIM PANEL</span>
          <span>{collapsed ? "▲" : "▼"}</span>
        </button>

        {!collapsed && (
          <div className="p-3 space-y-3">
            {/* FSM state display */}
            <div className="space-y-1 text-xs">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Round</span>
                <span className="font-mono text-amber-300">
                  {engine.roundState ?? "—"}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Player</span>
                <span className="font-mono text-amber-300">
                  {engine.playerState ?? "—"}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Stars</span>
                <span className="font-mono text-primary">
                  {state.player?.StarsBalance ?? "—"}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Shards</span>
                <span className="font-mono text-accent">
                  {state.player?.ShardsBalance ?? "—"}
                </span>
              </div>
              {engine.payoutInfo && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Payout</span>
                  <span className="font-mono text-green-400">
                    #{engine.payoutInfo.rank} → {engine.payoutInfo.amount}★
                  </span>
                </div>
              )}
              {engine.shardCredit != null && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Shards earned</span>
                  <span className="font-mono text-accent">
                    +{engine.shardCredit}
                  </span>
                </div>
              )}
            </div>

            <div className="border-t border-border/30 pt-2 space-y-2">
              <p className="text-[10px] text-muted-foreground uppercase tracking-wide">
                Controls
              </p>

              <Button
                size="sm"
                variant="outline"
                className="w-full text-xs h-8 border-destructive/40 text-destructive hover:bg-destructive/10"
                disabled={!isSurvival}
                onClick={() =>
                  engine.debugCommand({ type: "FORCE_LIQUIDATION" })
                }
              >
                Force Liquidation
              </Button>

              <Button
                size="sm"
                variant="outline"
                className="w-full text-xs h-8 border-amber-500/40 text-amber-400 hover:bg-amber-500/10"
                disabled={!isSurvival}
                onClick={() => engine.debugCommand({ type: "FORCE_TOP3" })}
              >
                Force Top-3 Finish
              </Button>

              <Button
                size="sm"
                variant="outline"
                className="w-full text-xs h-8"
                disabled={!inRoom}
                onClick={() => engine.simulateDisconnect()}
              >
                Simulate Disconnect
              </Button>

              <Button
                size="sm"
                variant="outline"
                className="w-full text-xs h-8"
                disabled={!inRoom}
                onClick={() => engine.simulateReconnect()}
              >
                Simulate Reconnect
              </Button>

              <Button
                size="sm"
                variant="outline"
                className="w-full text-xs h-8"
                disabled={!inRoom}
                onClick={() =>
                  engine.debugCommand({ type: "FORCE_COUNTDOWN_END" })
                }
              >
                Skip Countdown
              </Button>

              {/* Mass elimination */}
              <div className="flex gap-1.5 items-center">
                <input
                  type="number"
                  min={1}
                  max={50}
                  value={massCount}
                  onChange={(e) => setMassCount(Number(e.target.value))}
                  className="w-14 h-8 rounded border border-border/50 bg-background text-xs text-center font-mono text-foreground"
                />
                <Button
                  size="sm"
                  variant="outline"
                  className="flex-1 text-xs h-8"
                  disabled={!isSurvival}
                  onClick={() =>
                    engine.debugCommand({
                      type: "MASS_ELIMINATION",
                      value: massCount,
                    })
                  }
                >
                  Mass Eliminate
                </Button>
              </div>

              {/* Pulse window */}
              <div className="space-y-1">
                <label className="text-[10px] text-muted-foreground">
                  Pulse Window: {pulseWindow}s
                </label>
                <input
                  type="range"
                  min={1}
                  max={15}
                  step={0.5}
                  value={pulseWindow}
                  onChange={(e) => {
                    const v = Number(e.target.value);
                    setPulseWindow(v);
                    engine.debugCommand({
                      type: "SET_PULSE_WINDOW",
                      value: v,
                    });
                  }}
                  className="w-full accent-amber-500"
                />
              </div>

              {/* Lag injection */}
              <div className="space-y-1">
                <label className="text-[10px] text-muted-foreground">
                  Inject Lag: {lagMs}ms
                </label>
                <input
                  type="range"
                  min={0}
                  max={2000}
                  step={50}
                  value={lagMs}
                  onChange={(e) => {
                    const v = Number(e.target.value);
                    setLagMs(v);
                    engine.debugCommand({ type: "INJECT_LAG", value: v });
                  }}
                  className="w-full accent-amber-500"
                />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
