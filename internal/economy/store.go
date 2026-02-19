package economy

import (
	"context"
	"fmt"

	"github.com/lastclick/lastclick/internal/store"
)

// CosmeticStore manages the shard-based cosmetic storefront.
type CosmeticStore struct {
	players *store.PlayerStore
	txs     *store.TransactionStore
}

func NewCosmeticStore(players *store.PlayerStore, txs *store.TransactionStore) *CosmeticStore {
	return &CosmeticStore{players: players, txs: txs}
}

// Purchase deducts shards and grants a cosmetic item to the player.
func (c *CosmeticStore) Purchase(ctx context.Context, playerID int64, shardCost int64) error {
	player, err := c.players.Get(ctx, playerID)
	if err != nil {
		return err
	}
	if player == nil {
		return fmt.Errorf("player not found")
	}
	if player.ShardsBalance < shardCost {
		return fmt.Errorf("insufficient shards")
	}
	if err := c.players.UpdateBalance(ctx, playerID, 0, -shardCost); err != nil {
		return err
	}
	return c.txs.Record(ctx, playerID, store.TxCosmetic, -shardCost, nil)
}
