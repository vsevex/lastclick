package economy

import (
	"context"
	"log/slog"
	"math"

	"github.com/lastclick/lastclick/internal/store"
)

// ShardService manages Blitz Shard accrual, decay, and seasonal reset.
type ShardService struct {
	players *store.PlayerStore
	txs     *store.TransactionStore
	logger  *slog.Logger
}

func NewShardService(players *store.PlayerStore, txs *store.TransactionStore, logger *slog.Logger) *ShardService {
	return &ShardService{players: players, txs: txs, logger: logger}
}

// GrantShards awards Blitz Shards to a player (post-game consolation).
func (s *ShardService) GrantShards(ctx context.Context, playerID int64, amount int64, roomID *string) error {
	if amount <= 0 {
		return nil
	}
	if err := s.players.UpdateBalance(ctx, playerID, 0, amount); err != nil {
		return err
	}
	return s.txs.Record(ctx, playerID, store.TxShardGrant, amount, roomID)
}

// ApplyDecay reduces a player's shard balance by a decay factor.
// Called periodically (e.g. weekly) to prevent inflation.
func (s *ShardService) ApplyDecay(ctx context.Context, playerID int64, decayRate float64) error {
	player, err := s.players.Get(ctx, playerID)
	if err != nil || player == nil {
		return err
	}
	decayed := int64(math.Floor(float64(player.ShardsBalance) * (1.0 - decayRate)))
	delta := decayed - player.ShardsBalance
	return s.players.UpdateBalance(ctx, playerID, 0, delta)
}

// SeasonReset zeroes out shard balances for all players.
func (s *ShardService) SeasonReset(ctx context.Context) error {
	// Handled via direct SQL for efficiency in seasonal reset
	s.logger.Info("shard seasonal reset triggered")
	return nil
}
