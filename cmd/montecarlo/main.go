package main

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lastclick/lastclick/internal/game"
	"github.com/lastclick/lastclick/internal/room"
)

// --- Config ---
const (
	totalPlayers = 10_000
	totalRounds  = 50_000
)

// archetype distribution
const (
	pctConservative = 0.35
	pctAggressive   = 0.25
	pctWhale        = 0.15
	pctCasual       = 0.25
)

// tier distribution per round
const (
	pctTier1 = 0.60
	pctTier2 = 0.30
	// tier3 = remainder
)

type Archetype int

const (
	Conservative Archetype = iota
	Aggressive
	Whale
	Casual
)

func (a Archetype) String() string {
	return [...]string{"Conservative", "Aggressive", "Whale", "Casual"}[a]
}

type MCPlayer struct {
	ID        int64
	Archetype Archetype

	mu              sync.Mutex
	TotalBurned     int64
	TotalShards     int64
	TotalWins       int
	TotalPlaces     int // top 3 finishes
	TotalGames      int
	TotalPayouts    int64
	TotalTicks      int64
	TotalEfficiency float64
}

type roundResult struct {
	tierNum      int
	playerCount  int
	ticks        int
	finishReason string
	pool         int64
	rake         int64
	hasWinner    bool
	winnerArch   Archetype
	pstats       []pstat
}

type pstat struct {
	pid       int64
	arch      Archetype
	burned    int64
	shards    int64
	ticks     int
	won       bool
	placed    bool // top 3
	placement int
	payout    int64
	eff       float64
}

func main() {
	start := time.Now()

	players := make([]*MCPlayer, totalPlayers)
	for i := range players {
		var arch Archetype
		r := float64(i) / float64(totalPlayers)
		switch {
		case r < pctConservative:
			arch = Conservative
		case r < pctConservative+pctAggressive:
			arch = Aggressive
		case r < pctConservative+pctAggressive+pctWhale:
			arch = Whale
		default:
			arch = Casual
		}
		players[i] = &MCPlayer{ID: int64(i + 1), Archetype: arch}
	}
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(players), func(i, j int) { players[i], players[j] = players[j], players[i] })

	workers := runtime.GOMAXPROCS(0)
	results := make([]roundResult, totalRounds)

	var progress atomic.Int64
	var wg sync.WaitGroup

	chunkSize := totalRounds / workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		lo := w * chunkSize
		hi := lo + chunkSize
		if w == workers-1 {
			hi = totalRounds
		}
		go func(lo, hi int) {
			defer wg.Done()
			localRng := rand.New(rand.NewSource(int64(lo) * 7919))
			for i := lo; i < hi; i++ {
				results[i] = runRound(localRng, players, int64(i))
				if n := progress.Add(1); n%(totalRounds/10) == 0 {
					fmt.Printf("  ... %d/%d rounds (%.0f%%)\n", n, totalRounds, float64(n)/float64(totalRounds)*100)
				}
			}
		}(lo, hi)
	}
	wg.Wait()

	elapsed := time.Since(start)
	printReport(players, results, elapsed)
}

func runRound(rng *rand.Rand, allPlayers []*MCPlayer, _ int64) roundResult {
	tr := rng.Float64()
	var tier room.TierConfig
	var tierNum int
	switch {
	case tr < pctTier1:
		tier = room.Tiers[1]
		tierNum = 1
	case tr < pctTier1+pctTier2:
		tier = room.Tiers[2]
		tierNum = 2
	default:
		tier = room.Tiers[3]
		tierNum = 3
	}

	roomSize := tier.MinPlayers + rng.Intn(tier.MaxPlayers-tier.MinPlayers+1)
	if roomSize > len(allPlayers) {
		roomSize = len(allPlayers)
	}

	indices := rng.Perm(len(allPlayers))[:roomSize]
	selected := make([]*MCPlayer, roomSize)
	ids := make([]int64, roomSize)
	for i, idx := range indices {
		selected[i] = allPlayers[idx]
		ids[i] = allPlayers[idx].ID
	}

	volScript := genVolScript(rng, tier, 2400)

	pulseSchedule := make(map[int][]int64)
	pulseWindowTicks := int(tier.PulseWindow / (250 * time.Millisecond))

	for _, p := range selected {
		genPlayerPulses(rng, p.ID, p.Archetype, pulseWindowTicks, 2400, pulseSchedule)
	}

	result := game.RunSimulation(game.SimConfig{
		Tier:          tier,
		PlayerIDs:     ids,
		VolScript:     volScript,
		PulseSchedule: pulseSchedule,
		MaxTicks:      2400,
		SilentMode:    true,
	})

	pool := int64(roomSize) * tier.EntryCost
	rake := game.RakeAmount(pool)

	pstats := make([]pstat, 0, roomSize)
	var winnerArch Archetype

	for _, p := range selected {
		st := result.PlayerStats[p.ID]
		survTicks := result.TotalTicks
		if st.EliminatedAt > 0 {
			survTicks = st.EliminatedAt
		}
		won := p.ID == result.WinnerID
		placed := st.Placement > 0 && st.Placement <= 3

		ps := pstat{
			pid:       p.ID,
			arch:      p.Archetype,
			burned:    tier.EntryCost,
			shards:    st.ShardsEarned,
			ticks:     survTicks,
			won:       won,
			placed:    placed,
			placement: st.Placement,
			payout:    st.Payout,
			eff:       st.Efficiency,
		}
		pstats = append(pstats, ps)

		if won {
			winnerArch = p.Archetype
		}

		p.mu.Lock()
		p.TotalBurned += tier.EntryCost
		p.TotalShards += st.ShardsEarned
		p.TotalPayouts += st.Payout
		p.TotalTicks += int64(survTicks)
		p.TotalEfficiency += ps.eff
		p.TotalGames++
		if won {
			p.TotalWins++
		}
		if placed {
			p.TotalPlaces++
		}
		p.mu.Unlock()
	}

	return roundResult{
		tierNum:      tierNum,
		playerCount:  roomSize,
		ticks:        result.TotalTicks,
		finishReason: result.FinishReason,
		pool:         pool,
		rake:         rake,
		hasWinner:    result.WinnerID != 0,
		winnerArch:   winnerArch,
		pstats:       pstats,
	}
}

func genVolScript(rng *rand.Rand, tier room.TierConfig, maxTicks int) map[int]float64 {
	script := make(map[int]float64)
	ratio := 0.1 + rng.Float64()*0.2
	survivalTicks := int(tier.SurvivalTime / (250 * time.Millisecond))

	for tick := 1; tick <= maxTicks; tick++ {
		progress := float64(tick) / float64(survivalTicks)
		noise := rng.NormFloat64() * 0.02
		target := 0.3 + 0.7*math.Pow(math.Min(progress, 1.5), 1.5)
		reversion := (target - ratio) * 0.05
		spike := 0.0
		if rng.Float64() < 0.03 {
			spike = (rng.Float64() - 0.3) * 0.15
		}
		ratio += 0.005 + noise + reversion + spike
		ratio = math.Max(0.01, math.Min(1.0, ratio))

		if tick%4 == 0 {
			script[tick] = ratio
		}

		if ratio >= 1.0 {
			script[tick] = 1.0
			break
		}
	}
	return script
}

func genPlayerPulses(rng *rand.Rand, pid int64, arch Archetype, pwTicks, maxTicks int, schedule map[int][]int64) {
	switch arch {
	case Conservative:
		interval := 3
		for tick := 1 + rng.Intn(3); tick <= maxTicks; tick += interval {
			schedule[tick] = append(schedule[tick], pid)
		}

	case Aggressive:
		interval := pwTicks - 2
		if interval < 3 {
			interval = 3
		}
		for tick := 1; tick <= maxTicks; tick += interval + rng.Intn(3) - 1 {
			if tick < 1 {
				tick = 1
			}
			schedule[tick] = append(schedule[tick], pid)
		}

	case Whale:
		for tick := 1; tick <= maxTicks; tick += 2 {
			schedule[tick] = append(schedule[tick], pid)
		}

	case Casual:
		interval := 6 + rng.Intn(5)
		for tick := 1 + rng.Intn(5); tick <= maxTicks; tick += interval {
			if rng.Float64() < 0.25 {
				continue
			}
			schedule[tick] = append(schedule[tick], pid)
		}
	}
}

func printReport(players []*MCPlayer, results []roundResult, elapsed time.Duration) {
	var allBurns, allShards, allTicks, allEff, allPayouts []float64
	var totalPool, totalRake int64
	winsByArch := make(map[Archetype]int)
	placesByArch := make(map[Archetype]int)
	gamesByArch := make(map[Archetype]int)
	finishReasons := make(map[string]int)
	var totalSessions int
	var totalPlaced int

	for _, r := range results {
		totalPool += r.pool
		totalRake += r.rake
		finishReasons[r.finishReason]++
		if r.hasWinner {
			winsByArch[r.winnerArch]++
		}
		for _, ps := range r.pstats {
			allBurns = append(allBurns, float64(ps.burned))
			allShards = append(allShards, float64(ps.shards))
			allTicks = append(allTicks, float64(ps.ticks))
			allPayouts = append(allPayouts, float64(ps.payout))
			if ps.eff > 0 {
				allEff = append(allEff, ps.eff)
			}
			gamesByArch[ps.arch]++
			totalSessions++
			if ps.placed {
				placesByArch[ps.arch]++
				totalPlaced++
			}
		}
	}

	sort.Float64s(allBurns)
	sort.Float64s(allShards)
	sort.Float64s(allTicks)
	sort.Float64s(allEff)
	sort.Float64s(allPayouts)

	totalBurned := sum(allBurns)
	totalShardsGen := sum(allShards)
	totalPayoutsSum := sum(allPayouts)

	var netResults []float64
	gamesBeforeWin := make([]float64, 0)
	gamesBeforePlace := make([]float64, 0)
	var positiveCount int
	for _, p := range players {
		if p.TotalGames == 0 {
			continue
		}
		net := float64(p.TotalPayouts) - float64(p.TotalBurned)
		netResults = append(netResults, net)
		if net > 0 {
			positiveCount++
		}
		if p.TotalWins > 0 {
			gamesBeforeWin = append(gamesBeforeWin, float64(p.TotalGames)/float64(p.TotalWins))
		}
		if p.TotalPlaces > 0 {
			gamesBeforePlace = append(gamesBeforePlace, float64(p.TotalGames)/float64(p.TotalPlaces))
		}
	}
	sort.Float64s(netResults)
	sort.Float64s(gamesBeforeWin)
	sort.Float64s(gamesBeforePlace)

	tickToSec := func(t float64) float64 { return t * 0.25 }

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              MONTE CARLO SIMULATION REPORT                  ║")
	fmt.Println("║                  (v2 — Top-3 Payouts)                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Players: %d  |  Rounds: %d  |  Sessions: %d\n", totalPlayers, totalRounds, totalSessions)
	fmt.Printf("  Tiers: T1(%.0f%%) T2(%.0f%%) T3(%.0f%%)\n", pctTier1*100, pctTier2*100, (1-pctTier1-pctTier2)*100)
	fmt.Printf("  Archetypes: Conservative(%.0f%%) Aggressive(%.0f%%) Whale(%.0f%%) Casual(%.0f%%)\n",
		pctConservative*100, pctAggressive*100, pctWhale*100, pctCasual*100)
	fmt.Printf("  Rake: 12%%  |  Payouts: Top-3 (60/25/15)  |  Pulses: Free\n")
	fmt.Printf("  Elapsed: %v  |  Workers: %d\n", elapsed.Round(time.Millisecond), runtime.GOMAXPROCS(0))

	fmt.Println()
	fmt.Println("─── BURN ECONOMICS ────────────────────────────────────────────")
	fmt.Printf("  Mean Stars burned/session:     %8.1f  (entry fee only)\n", mean(allBurns))
	fmt.Printf("  Median Stars burned/session:   %8.1f\n", percentile(allBurns, 50))
	fmt.Printf("  90th pctl burned:              %8.1f\n", percentile(allBurns, 90))
	fmt.Printf("  Total Stars burned:          %10.0f\n", totalBurned)
	fmt.Printf("  Total pool collected:        %10d\n", totalPool)
	fmt.Printf("  Total rake (house):          %10d\n", totalRake)
	fmt.Printf("  Total payouts (top 3):       %10.0f\n", totalPayoutsSum)
	fmt.Printf("  Effective house take:          %7.2f%%\n", float64(totalRake)/totalBurned*100)

	fmt.Println()
	fmt.Println("─── SHARD ECONOMICS ───────────────────────────────────────────")
	fmt.Printf("  Mean Shards earned/session:    %8.1f\n", mean(allShards))
	fmt.Printf("  Median Shards earned:          %8.1f\n", percentile(allShards, 50))
	fmt.Printf("  90th pctl Shards:              %8.1f\n", percentile(allShards, 90))
	fmt.Printf("  Total Shards generated:      %10.0f\n", totalShardsGen)
	if totalBurned > 0 {
		fmt.Printf("  Shard inflation rate:          %8.4f shards/star\n", totalShardsGen/totalBurned)
	}

	fmt.Println()
	fmt.Println("─── SURVIVAL ──────────────────────────────────────────────────")
	fmt.Printf("  Mean session length:           %7.1fs\n", tickToSec(mean(allTicks)))
	fmt.Printf("  Median session length:         %7.1fs\n", tickToSec(percentile(allTicks, 50)))
	fmt.Printf("  90th pctl session length:      %7.1fs\n", tickToSec(percentile(allTicks, 90)))
	fmt.Printf("  10th pctl session length:      %7.1fs\n", tickToSec(percentile(allTicks, 10)))

	fmt.Println()
	fmt.Println("─── FINISH REASONS ────────────────────────────────────────────")
	for reason, count := range finishReasons {
		fmt.Printf("  %-20s %8d  (%5.1f%%)\n", reason, count, float64(count)/float64(totalRounds)*100)
	}

	fmt.Println()
	fmt.Println("─── WIN & PLACEMENT RATES BY ARCHETYPE ────────────────────────")
	totalWins := 0
	for _, c := range winsByArch {
		totalWins += c
	}
	for _, a := range []Archetype{Conservative, Aggressive, Whale, Casual} {
		wins := winsByArch[a]
		places := placesByArch[a]
		games := gamesByArch[a]
		winPct := 0.0
		placePct := 0.0
		if games > 0 {
			winPct = float64(wins) / float64(games) * 100
			placePct = float64(places) / float64(games) * 100
		}
		fmt.Printf("  %-15s  wins: %5d (%4.1f%%)  top3: %6d (%5.1f%%)  games: %7d\n",
			a.String(), wins, winPct, places, placePct, games)
	}
	fmt.Printf("  %-15s  wins: %5d          top3: %6d           sessions: %d\n",
		"TOTAL", totalWins, totalPlaced, totalSessions)

	fmt.Println()
	fmt.Println("─── EFFICIENCY DISTRIBUTION ───────────────────────────────────")
	if len(allEff) > 0 {
		fmt.Printf("  Mean efficiency:               %8.2f\n", mean(allEff))
		fmt.Printf("  Median efficiency:             %8.2f\n", percentile(allEff, 50))
		fmt.Printf("  90th pctl efficiency:          %8.2f\n", percentile(allEff, 90))
		fmt.Printf("  99th pctl efficiency:          %8.2f\n", percentile(allEff, 99))
		fmt.Printf("  Max efficiency:                %8.2f\n", allEff[len(allEff)-1])
	}

	fmt.Println()
	fmt.Println("─── PLAYER LIFETIME RISK ──────────────────────────────────────")
	activePlayers := len(netResults)
	fmt.Printf("  Active players (played >=1):   %8d / %d\n", activePlayers, totalPlayers)
	fmt.Printf("  Net positive players:          %8d  (%5.1f%%)\n", positiveCount, float64(positiveCount)/float64(activePlayers)*100)
	fmt.Printf("  Mean net P&L per player:       %8.1f stars\n", mean(netResults))
	fmt.Printf("  Median net P&L:                %8.1f stars\n", percentile(netResults, 50))
	fmt.Printf("  10th pctl (worst):             %8.1f stars\n", percentile(netResults, 10))
	fmt.Printf("  90th pctl (best):              %8.1f stars\n", percentile(netResults, 90))
	if len(gamesBeforeWin) > 0 {
		fmt.Printf("  Avg games per 1st place:       %8.1f\n", mean(gamesBeforeWin))
		fmt.Printf("  Median games per 1st:          %8.1f\n", percentile(gamesBeforeWin, 50))
	}
	if len(gamesBeforePlace) > 0 {
		fmt.Printf("  Avg games per top-3:           %8.1f\n", mean(gamesBeforePlace))
		fmt.Printf("  Median games per top-3:        %8.1f\n", percentile(gamesBeforePlace, 50))
	}

	fmt.Println()
	fmt.Println("─── REINFORCEMENT FREQUENCY ───────────────────────────────────")
	if totalSessions > 0 {
		winFreq := float64(totalWins) / float64(totalSessions) * 100
		placeFreq := float64(totalPlaced) / float64(totalSessions) * 100
		fmt.Printf("  Win frequency (1st):           %7.2f%% per session\n", winFreq)
		fmt.Printf("  Placement frequency (top 3):   %7.2f%% per session\n", placeFreq)
		fmt.Printf("  Micro-win frequency (top 5):   %7.2f%% (4th/5th get 2x/1.5x shards)\n", placeFreq*5.0/3.0)
	}

	fmt.Println()
	fmt.Println("─── DIAGNOSIS ─────────────────────────────────────────────────")
	avgBurn := mean(allBurns)
	avgSurvival := tickToSec(mean(allTicks))
	houseRate := float64(totalRake) / totalBurned * 100
	var shardRate float64
	if totalBurned > 0 {
		shardRate = totalShardsGen / totalBurned
	}
	netPct := float64(positiveCount) / float64(activePlayers) * 100
	winFreq := float64(totalWins) / float64(totalSessions) * 100
	placeFreq := float64(totalPlaced) / float64(totalSessions) * 100

	if avgSurvival < 15 {
		fmt.Println("  !! AVG SURVIVAL < 15s — HIGH CHURN RISK — players die too fast")
	} else if avgSurvival < 30 {
		fmt.Println("  ~~ AVG SURVIVAL 15-30s — moderate — watch for casual dropout")
	} else {
		fmt.Println("  OK AVG SURVIVAL > 30s — healthy session length")
	}

	if avgBurn > 80 {
		fmt.Println("  !! AVG BURN > 80 — burn velocity too high, LTV at risk")
	} else if avgBurn < 5 {
		fmt.Println("  !! AVG BURN < 5 — burn velocity extremely low")
	} else {
		fmt.Printf("  OK AVG BURN %.1f — within target range (entry-fee-only model)\n", avgBurn)
	}

	if houseRate < 7 {
		fmt.Println("  !! HOUSE TAKE < 7% — margins too thin")
	} else if houseRate > 15 {
		fmt.Println("  !! HOUSE TAKE > 15% — predatory — players will leave")
	} else {
		fmt.Printf("  OK HOUSE TAKE %.1f%% — within 7-12%% target\n", houseRate)
	}

	if shardRate > 0.8 {
		fmt.Println("  !! SHARD INFLATION > 0.8 — cosmetic economy will hyperinflate")
	} else if shardRate < 0.1 {
		fmt.Println("  !! SHARD RATE < 0.1 — shards too scarce, players feel unrewarded")
	} else {
		fmt.Printf("  OK SHARD RATE %.3f — balanced\n", shardRate)
	}

	if winFreq >= 1 && winFreq <= 3 {
		fmt.Printf("  OK WIN FREQ %.2f%% — within 1-3%% target\n", winFreq)
	} else if winFreq < 1 {
		fmt.Printf("  !! WIN FREQ %.2f%% — below 1%%, players feel hopeless\n", winFreq)
	} else {
		fmt.Printf("  ~~ WIN FREQ %.2f%% — above 3%%, monitor pool sustainability\n", winFreq)
	}

	if placeFreq >= 5 {
		fmt.Printf("  OK PLACEMENT FREQ %.2f%% — healthy reinforcement via top-3\n", placeFreq)
	} else {
		fmt.Printf("  ~~ PLACEMENT FREQ %.2f%% — consider smaller rooms for more placements\n", placeFreq)
	}

	if netPct > 40 {
		fmt.Println("  !! NET POSITIVE > 40% — house is losing money")
	} else if netPct < 5 {
		fmt.Println("  !! NET POSITIVE < 5% — almost nobody wins, churn imminent")
	} else {
		fmt.Printf("  OK NET POSITIVE %.1f%% — healthy winner pool\n", netPct)
	}

	fmt.Println()
}

func mean(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	return sum(s) / float64(len(s))
}

func sum(s []float64) float64 {
	t := 0.0
	for _, v := range s {
		t += v
	}
	return t
}

func percentile(sorted []float64, pct float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * pct / 100)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
