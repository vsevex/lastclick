import { Link } from "react-router-dom";
import { useEffect, useState } from "react";
import { useSocket } from "@/context/SocketContext";
import { Button } from "@/components/ui/button";

interface LiveMetrics {
  active_rooms: number;
  ws_connections: number;
  total_pulses: number;
  total_rooms: number;
}

export function HeroSection() {
  const { connected } = useSocket();
  const [metrics, setMetrics] = useState<LiveMetrics | null>(null);

  useEffect(() => {
    const fetchMetrics = () =>
      fetch("/metrics")
        .then((r) => r.json())
        .then(setMetrics)
        .catch(() => {});

    fetchMetrics();
    const id = setInterval(fetchMetrics, 10_000);
    return () => clearInterval(id);
  }, []);

  return (
    <section className="min-h-[calc(100vh-60px)] bg-linear-to-b from-background via-background to-card flex items-center justify-center px-3 sm:px-4 py-8">
      <div className="max-w-4xl w-full mx-auto">
        <div className="mb-8 sm:mb-12 text-center space-y-4 sm:space-y-6">
          <div className="inline-block px-4 py-2 rounded-full border border-primary/30 bg-primary/5">
            <p className="text-xs sm:text-sm font-medium text-primary">
              LEVERAGE THE VOLATILITY
            </p>
          </div>

          <h1 className="text-4xl sm:text-5xl md:text-7xl font-bold text-foreground leading-tight text-balance">
            Last Click
          </h1>

          <p className="text-lg sm:text-xl md:text-2xl text-muted-foreground max-w-2xl mx-auto leading-relaxed">
            A high-fidelity volatility survival game where precision, timing,
            and nerve determine who walks away with Blitz Shards.
          </p>
        </div>

        <div className="flex flex-col sm:flex-row gap-3 sm:gap-4 justify-center mb-10 sm:mb-16">
          <Link to="/rooms" className="w-full sm:w-auto">
            <Button
              size="lg"
              className="w-full bg-primary text-primary-foreground hover:bg-primary/90 font-semibold min-h-[48px]"
            >
              {connected ? "Enter The Game" : "Connecting..."}
            </Button>
          </Link>
          <Link to="/profile" className="w-full sm:w-auto">
            <Button
              size="lg"
              variant="outline"
              className="w-full border-border hover:bg-card min-h-[48px]"
            >
              View Profile
            </Button>
          </Link>
        </div>

        <div className="grid grid-cols-3 gap-3 sm:gap-6">
          <div className="p-4 sm:p-6 rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm hover:border-primary/50 transition-colors">
            <div className="text-xl sm:text-3xl font-bold text-primary mb-1 sm:mb-2">
              50v1
            </div>
            <h3 className="font-semibold text-foreground text-sm sm:text-base mb-1 sm:mb-2">
              Survival
            </h3>
            <p className="text-xs sm:text-sm text-muted-foreground hidden sm:block">
              Up to 50 traders per room, only one walks away with the pool
            </p>
          </div>

          <div className="p-4 sm:p-6 rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm hover:border-primary/50 transition-colors">
            <div className="text-xl sm:text-3xl font-bold text-accent mb-1 sm:mb-2">
              Infinite
            </div>
            <h3 className="font-semibold text-foreground text-sm sm:text-base mb-1 sm:mb-2">
              Rounds
            </h3>
            <p className="text-xs sm:text-sm text-muted-foreground hidden sm:block">
              Play as many rounds as you want and climb the seasonal
              leaderboards
            </p>
          </div>

          <div className="p-4 sm:p-6 rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm hover:border-primary/50 transition-colors">
            <div className="text-xl sm:text-3xl font-bold text-destructive mb-1 sm:mb-2">
              Dynamic
            </div>
            <h3 className="font-semibold text-foreground text-sm sm:text-base mb-1 sm:mb-2">
              Cosmetics
            </h3>
            <p className="text-xs sm:text-sm text-muted-foreground hidden sm:block">
              Customize your avatar with rare cosmetics earned through gameplay
            </p>
          </div>
        </div>

        <div className="mt-8 sm:mt-12 p-5 sm:p-8 rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm">
          <h2 className="text-lg sm:text-xl font-bold text-foreground mb-5 sm:mb-6">
            Live Right Now
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 sm:gap-6">
            <div className="p-3 sm:p-4 rounded-lg border border-border/30 bg-background/50">
              <p className="text-[10px] sm:text-xs text-muted-foreground uppercase tracking-wider mb-1">
                Active Games
              </p>
              <p className="text-xl sm:text-2xl font-bold text-primary font-mono">
                {metrics?.active_rooms ?? "—"}
              </p>
            </div>
            <div className="p-3 sm:p-4 rounded-lg border border-border/30 bg-background/50">
              <p className="text-[10px] sm:text-xs text-muted-foreground uppercase tracking-wider mb-1">
                Players Online
              </p>
              <p className="text-xl sm:text-2xl font-bold text-accent font-mono">
                {metrics?.ws_connections ?? "—"}
              </p>
            </div>
            <div className="p-3 sm:p-4 rounded-lg border border-border/30 bg-background/50">
              <p className="text-[10px] sm:text-xs text-muted-foreground uppercase tracking-wider mb-1">
                Total Pulses
              </p>
              <p className="text-xl sm:text-2xl font-bold text-accent font-mono">
                {metrics?.total_pulses ?? "—"}
              </p>
            </div>
            <div className="p-3 sm:p-4 rounded-lg border border-border/30 bg-background/50">
              <p className="text-[10px] sm:text-xs text-muted-foreground uppercase tracking-wider mb-1">
                Rooms Played
              </p>
              <p className="text-xl sm:text-2xl font-bold text-primary font-mono">
                {metrics?.total_rooms ?? "—"}
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
