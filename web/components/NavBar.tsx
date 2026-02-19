"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useGame } from "@/context/GameContext";
import { useSocket } from "@/context/SocketContext";

interface NavBarProps {
  showProfile?: boolean;
  showStore?: boolean;
  title?: string;
}

export function NavBar({
  showProfile = true,
  showStore = true,
  title,
}: NavBarProps) {
  const { state } = useGame();
  const { connected } = useSocket();

  return (
    <div className="sticky top-0 z-50 border-b border-border/50 bg-background/80 backdrop-blur-sm safe-top">
      <div className="max-w-7xl mx-auto px-3 sm:px-4 py-3 flex items-center justify-between">
        <Link href="/" className="flex items-center gap-2 group">
          <div className="w-8 h-8 rounded bg-linear-to-br from-primary to-accent flex items-center justify-center group-hover:shadow-lg group-hover:shadow-primary/20 transition-all">
            <span className="text-xs font-bold text-primary-foreground">
              LC
            </span>
          </div>
          <span className="font-bold text-foreground hidden sm:inline">
            {title || "Last Click"}
          </span>
        </Link>

        <nav className="flex items-center gap-1 sm:gap-2">
          <div
            className={`w-2 h-2 rounded-full mr-1 ${connected ? "bg-primary" : "bg-destructive"}`}
          />
          {showProfile && (
            <Link href="/profile">
              <Button
                variant="ghost"
                size="sm"
                className="text-muted-foreground hover:text-foreground min-h-[40px] min-w-[40px]"
              >
                Profile
              </Button>
            </Link>
          )}
          {showStore && (
            <Link href="/store">
              <Button
                variant="ghost"
                size="sm"
                className="text-muted-foreground hover:text-foreground min-h-[40px] min-w-[40px]"
              >
                Store
              </Button>
            </Link>
          )}
          {state.player && (
            <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-card/50 border border-border/30 ml-1 sm:ml-2">
              <span className="text-xs text-muted-foreground">
                &#9733; {state.player.StarsBalance}
              </span>
              <span className="text-xs text-accent font-bold">
                {state.player.ShardsBalance}
              </span>
            </div>
          )}
        </nav>
      </div>
    </div>
  );
}
