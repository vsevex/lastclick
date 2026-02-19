package volatility

// Feed provides margin ratio updates to a game room.
// MarginRatio goes from 0 (safe) to 1 (liquidation).
type Feed interface {
	// Start begins the feed. It sends updates on the returned channel.
	// Close the stop channel to terminate.
	Start(stop <-chan struct{}) <-chan Update
}

type Update struct {
	MarginRatio float64
	Price       float64
}
