package game

import (
	"math"
	"time"
)

// VolatilityMultiplier increases as the margin ratio approaches liquidation (1.0).
func VolatilityMultiplier(marginRatio float64) float64 {
	clamped := math.Max(0, math.Min(1, marginRatio))
	return 1.0 + 4.0*math.Pow(clamped, 3)
}

// SurvivalEfficiency = (TimeSurvivedSec * VolatilityMultiplier) / StarsSpent
func SurvivalEfficiency(timeSurvived time.Duration, volMul float64, starsSpent int64) float64 {
	if starsSpent <= 0 {
		return 0
	}
	return (timeSurvived.Seconds() * volMul) / float64(starsSpent)
}

// LatencyGraceTicks absorbs network jitter in the pulse window check.
// With 250ms ticks, 1 grace tick absorbs up to ~500ms of jitter.
const LatencyGraceTicks = 1

// RakeAmount computes the 12% pool rake.
func RakeAmount(pool int64) int64 {
	return pool * 12 / 100
}

// PlacementPayout describes a single placement's star reward.
type PlacementPayout struct {
	Place  int
	Amount int64
}

// PlacementPayouts returns top-3 payouts from the post-rake pool.
//
//	>=3 players → 1st 60%, 2nd 25%, 3rd 15%
//	 2 players → 1st 75%, 2nd 25%
//	 1 player  → 1st 100%
func PlacementPayouts(pool int64, numPlayers int) []PlacementPayout {
	postRake := pool - RakeAmount(pool)
	if numPlayers <= 1 {
		return []PlacementPayout{{1, postRake}}
	}
	if numPlayers == 2 {
		return []PlacementPayout{
			{1, postRake * 75 / 100},
			{2, postRake * 25 / 100},
		}
	}
	return []PlacementPayout{
		{1, postRake * 60 / 100},
		{2, postRake * 25 / 100},
		{3, postRake * 15 / 100},
	}
}

// WarChestContribution is 3% of total rake.
func WarChestContribution(rake int64) int64 {
	return rake * 3 / 100
}

// ShardsForLoser converts entry cost into Blitz Shards for non-placing players.
// 4th place gets 2x base, 5th gets 1.5x, everyone else gets base.
// Base ratio scales 40-60% with volatility.
func ShardsForLoser(entryCost int64, volMul float64, placement int) int64 {
	ratio := 0.4 + 0.2*math.Min(1, (volMul-1)/4)
	base := int64(float64(entryCost) * ratio)
	switch placement {
	case 4:
		return base * 2
	case 5:
		return base * 3 / 2
	default:
		return base
	}
}
