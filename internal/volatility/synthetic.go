package volatility

import (
	"math"
	"math/rand"
	"time"
)

// SyntheticFeed simulates a whale position margin ratio using a random walk
// with mean-reversion toward liquidation. Used for Blitz rooms.
type SyntheticFeed struct {
	Duration   time.Duration // target session duration
	TickRate   time.Duration
	Volatility float64 // step size scaling (default 0.02)
	Drift      float64 // upward drift toward liquidation (default 0.005)
}

func NewSyntheticFeed(duration time.Duration) *SyntheticFeed {
	return &SyntheticFeed{
		Duration:   duration,
		TickRate:   250 * time.Millisecond,
		Volatility: 0.02,
		Drift:      0.005,
	}
}

func (f *SyntheticFeed) Start(stop <-chan struct{}) <-chan Update {
	ch := make(chan Update, 32)
	go f.run(stop, ch)
	return ch
}

func (f *SyntheticFeed) run(stop <-chan struct{}, ch chan<- Update) {
	defer close(ch)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	ratio := 0.1 + rng.Float64()*0.2 // start between 0.1 and 0.3

	ticker := time.NewTicker(f.TickRate)
	defer ticker.Stop()

	elapsed := time.Duration(0)

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			elapsed += f.TickRate
			progress := float64(elapsed) / float64(f.Duration)

			// Noise component
			noise := rng.NormFloat64() * f.Volatility

			// Mean-reversion toward a target that increases over time
			target := 0.3 + 0.7*math.Pow(progress, 1.5)
			reversion := (target - ratio) * 0.05

			// Entropy injection: occasional spikes
			spike := 0.0
			if rng.Float64() < 0.03 {
				spike = (rng.Float64() - 0.3) * 0.15
			}

			ratio += f.Drift + noise + reversion + spike
			ratio = math.Max(0.01, math.Min(1.0, ratio))

			// Simulated price (arbitrary base)
			price := 100.0 * (1.0 - ratio*0.5)

			select {
			case ch <- Update{MarginRatio: ratio, Price: price}:
			default:
			}
		}
	}
}
