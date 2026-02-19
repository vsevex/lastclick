package economy

import (
	"context"

	"github.com/lastclick/lastclick/internal/game"
	"github.com/lastclick/lastclick/internal/store"
)

// RakeService handles pool rake extraction and distribution.
type RakeService struct {
	txs    *store.TransactionStore
	squads *store.SquadStore
}

func NewRakeService(txs *store.TransactionStore, squads *store.SquadStore) *RakeService {
	return &RakeService{txs: txs, squads: squads}
}

// ProcessRake records the rake transaction and distributes war chest contributions.
func (r *RakeService) ProcessRake(ctx context.Context, pool int64, roomID string) (rake int64, warChest int64) {
	rake = game.RakeAmount(pool)
	warChest = game.WarChestContribution(rake)
	return rake, warChest
}
