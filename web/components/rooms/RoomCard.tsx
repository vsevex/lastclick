"use client";

import Link from "next/link";
import type { RoomInfo } from "@/types/game";
import { TIERS } from "@/types/game";
import { Button } from "@/components/ui/button";
import { useGame } from "@/context/GameContext";

export function RoomCard({ room }: { room: RoomInfo }) {
  const { joinRoom } = useGame();
  const tier = TIERS[room.tier];
  const canJoin = room.state === "waiting";

  const stateColor =
    room.state === "waiting"
      ? "text-primary"
      : room.state === "survival"
        ? "text-destructive"
        : "text-accent";

  return (
    <div className="h-full flex flex-col p-4 sm:p-6 rounded-xl border border-border/50 bg-card/50 backdrop-blur-sm hover:border-primary/50 hover:bg-card/70 transition-all duration-300 group">
      <div className="flex items-start justify-between mb-3 sm:mb-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-xs font-bold uppercase text-muted-foreground">
              {room.type}
            </span>
            <span className="text-xs font-bold text-accent">T{room.tier}</span>
          </div>
          <p className="text-xs text-muted-foreground truncate">
            {room.id.slice(0, 8)}...
          </p>
        </div>
        <div
          className={`px-2.5 py-1 rounded-full text-xs font-semibold ${stateColor} bg-background/50 capitalize`}
        >
          {room.state}
        </div>
      </div>

      <div className="space-y-2.5 mb-4 sm:mb-6 flex-1">
        <div className="flex justify-between items-center">
          <span className="text-xs text-muted-foreground uppercase tracking-wide">
            Players
          </span>
          <span className="text-sm font-semibold text-foreground">
            {room.players} / {tier?.maxPlayers ?? "?"}
          </span>
        </div>
        <div className="w-full bg-background/50 rounded-full h-2 overflow-hidden">
          <div
            className="h-full bg-linear-to-r from-primary to-accent transition-all duration-300"
            style={{
              width: `${tier ? (room.players / tier.maxPlayers) * 100 : 0}%`,
            }}
          />
        </div>

        <div className="flex justify-between items-center pt-1">
          <span className="text-xs text-muted-foreground uppercase tracking-wide">
            Entry Fee
          </span>
          <span className="text-sm font-semibold text-primary">
            {tier?.entryCost ?? "?"} Stars
          </span>
        </div>

        <div className="flex justify-between items-center">
          <span className="text-xs text-muted-foreground uppercase tracking-wide">
            Pool
          </span>
          <span className="text-sm font-mono text-foreground">
            {room.pool} Stars
          </span>
        </div>
      </div>

      {canJoin ? (
        <Link
          href="/game"
          className="w-full block"
          onClick={() => joinRoom(room.id)}
        >
          <Button
            className="w-full bg-primary/80 hover:bg-primary text-primary-foreground font-semibold min-h-[44px] transition-all duration-300 group-hover:shadow-lg group-hover:shadow-primary/20"
            size="sm"
          >
            Join Room
          </Button>
        </Link>
      ) : (
        <Button
          disabled
          className="w-full min-h-[44px]"
          variant="outline"
          size="sm"
        >
          {room.state === "finished" ? "Finished" : "In Progress"}
        </Button>
      )}
    </div>
  );
}
