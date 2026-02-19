package volatility

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// LiveFeed connects to a TON DEX / oracle API to track real whale position margin ratios.
// It polls an HTTP endpoint for position data and converts it to margin ratio updates.
type LiveFeed struct {
	OracleURL     string
	PositionID    string
	TickRate      time.Duration
	Logger        *slog.Logger
	LiquidPrice   float64 // position's liquidation price
	EntryPrice    float64 // position's entry price
	IsLong        bool
}

func NewLiveFeed(oracleURL string, positionID string, liquidPrice, entryPrice float64, isLong bool, logger *slog.Logger) *LiveFeed {
	return &LiveFeed{
		OracleURL:   oracleURL,
		PositionID:  positionID,
		TickRate:    500 * time.Millisecond,
		Logger:      logger,
		LiquidPrice: liquidPrice,
		EntryPrice:  entryPrice,
		IsLong:      isLong,
	}
}

type oracleResponse struct {
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
}

func (f *LiveFeed) Start(stop <-chan struct{}) <-chan Update {
	ch := make(chan Update, 32)
	go f.run(stop, ch)
	return ch
}

func (f *LiveFeed) run(stop <-chan struct{}, ch chan<- Update) {
	defer close(ch)

	ticker := time.NewTicker(f.TickRate)
	defer ticker.Stop()

	client := &http.Client{Timeout: 2 * time.Second}

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			price, err := f.fetchPrice(client)
			if err != nil {
				f.Logger.Warn("oracle fetch failed, skipping tick", "err", err)
				continue
			}

			ratio := f.computeMarginRatio(price)

			select {
			case ch <- Update{MarginRatio: ratio, Price: price}:
			default:
			}
		}
	}
}

func (f *LiveFeed) fetchPrice(client *http.Client) (float64, error) {
	if f.OracleURL == "" {
		return 0, fmt.Errorf("no oracle URL configured")
	}

	url := fmt.Sprintf("%s/price/%s", f.OracleURL, f.PositionID)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data oracleResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	return data.Price, nil
}

// computeMarginRatio converts current price to a 0..1 margin ratio.
// 0 = safe (far from liquidation), 1 = liquidated.
func (f *LiveFeed) computeMarginRatio(currentPrice float64) float64 {
	if f.EntryPrice == f.LiquidPrice {
		return 0
	}

	var ratio float64
	if f.IsLong {
		// Long: liquidation when price drops to LiquidPrice
		totalRange := f.EntryPrice - f.LiquidPrice
		distToLiquid := currentPrice - f.LiquidPrice
		ratio = 1.0 - (distToLiquid / totalRange)
	} else {
		// Short: liquidation when price rises to LiquidPrice
		totalRange := f.LiquidPrice - f.EntryPrice
		distToLiquid := f.LiquidPrice - currentPrice
		ratio = 1.0 - (distToLiquid / totalRange)
	}

	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return ratio
}
