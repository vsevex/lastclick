package squad

import (
	"context"
	"log/slog"

	"github.com/lastclick/lastclick/internal/store"
)

// WarChestService manages squad war chest accumulation and auto-distribution.
type WarChestService struct {
	squads  *store.SquadStore
	players *store.PlayerStore
	logger  *slog.Logger
}

func NewWarChestService(squads *store.SquadStore, players *store.PlayerStore, logger *slog.Logger) *WarChestService {
	return &WarChestService{squads: squads, players: players, logger: logger}
}

// Contribute adds funds to a squad's war chest (called from rake processing).
func (w *WarChestService) Contribute(ctx context.Context, squadID string, amount int64) error {
	return w.squads.AddToWarChest(ctx, squadID, amount)
}

// AutoDistribute distributes war chest funds equally to squad members as streak protection.
// Called periodically (e.g. end of each day or season).
func (w *WarChestService) AutoDistribute(ctx context.Context, squadID string) error {
	sq, err := w.squads.Get(ctx, squadID)
	if err != nil || sq == nil {
		return err
	}
	if sq.WarChest == 0 || sq.MemberCount == 0 {
		return nil
	}

	// Distribute 10% of war chest each cycle
	distributable := sq.WarChest / 10
	if distributable == 0 {
		return nil
	}
	perMember := distributable / int64(sq.MemberCount)
	if perMember == 0 {
		return nil
	}

	// Deduct from war chest
	if err := w.squads.AddToWarChest(ctx, squadID, -distributable); err != nil {
		return err
	}

	w.logger.Info("war chest distributed",
		"squad", squadID,
		"total", distributable,
		"per_member", perMember,
		"members", sq.MemberCount,
	)
	return nil
}
