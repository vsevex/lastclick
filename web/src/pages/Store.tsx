import { NavBar } from "@/components/NavBar";
import { useGame } from "@/context/GameContext";

export default function Store() {
  const { state } = useGame();

  return (
    <main className="min-h-screen bg-background">
      <NavBar title="Cosmetic Store" showProfile={false} />

      <div className="max-w-7xl mx-auto px-3 sm:px-4 py-6 sm:py-12">
        <div className="mb-8 sm:mb-12">
          <h1 className="text-3xl sm:text-4xl md:text-5xl font-bold text-foreground mb-3 sm:mb-4">
            Cosmetic Store
          </h1>
          <p className="text-base sm:text-lg text-muted-foreground max-w-2xl">
            Customize your trader with exclusive cosmetics. Earn shards by
            surviving games.
          </p>
          {state.player && (
            <p className="text-sm text-accent font-bold mt-2">
              Your Shards: {state.player.ShardsBalance}
            </p>
          )}
        </div>

        <div className="text-center py-20 rounded-lg border border-border/50 bg-card/50">
          <p className="text-2xl font-bold text-foreground mb-2">Coming Soon</p>
          <p className="text-muted-foreground max-w-md mx-auto">
            The cosmetic store is being prepared. Play games and earn Blitz
            Shards now to be ready when it launches.
          </p>
        </div>

        <div className="mt-10 sm:mt-16 rounded-lg border border-border/50 bg-linear-to-br from-card via-card to-background p-5 sm:p-8">
          <h2 className="text-xl sm:text-2xl font-bold text-foreground mb-4 sm:mb-6">
            How to Earn Shards
          </h2>
          <div className="grid sm:grid-cols-2 gap-6">
            <div>
              <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                <span className="text-2xl font-bold text-primary">&#9889;</span>
              </div>
              <h3 className="font-semibold text-foreground mb-2">Star Burn</h3>
              <p className="text-sm text-muted-foreground">
                Stars spent during games convert to Blitz Shards. Higher
                volatility rooms give better conversion rates (40-60%).
              </p>
            </div>
            <div>
              <div className="w-12 h-12 rounded-lg bg-accent/10 flex items-center justify-center mb-4">
                <span className="text-2xl font-bold text-accent">
                  &#127942;
                </span>
              </div>
              <h3 className="font-semibold text-foreground mb-2">Win Games</h3>
              <p className="text-sm text-muted-foreground">
                Last player standing wins the pool minus 10% rake. The winner
                takes home up to 90% of all entry fees.
              </p>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
