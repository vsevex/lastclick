package squad

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lastclick/lastclick/internal/store"
)

const maxSquadSize = 50

type Service struct {
	squads  *store.SquadStore
	players *store.PlayerStore
	logger  *slog.Logger
}

func NewService(squads *store.SquadStore, players *store.PlayerStore, logger *slog.Logger) *Service {
	return &Service{squads: squads, players: players, logger: logger}
}

func (s *Service) Create(ctx context.Context, name string, founderID int64) (*store.Squad, error) {
	player, err := s.players.Get(ctx, founderID)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return nil, fmt.Errorf("player not found")
	}
	if player.SquadID != nil {
		return nil, fmt.Errorf("already in a squad")
	}

	sq, err := s.squads.Create(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("create squad: %w", err)
	}

	if err := s.players.SetSquad(ctx, founderID, &sq.ID); err != nil {
		return nil, err
	}
	if err := s.squads.AddMember(ctx, sq.ID); err != nil {
		return nil, err
	}

	return sq, nil
}

func (s *Service) Join(ctx context.Context, playerID int64, squadID string) error {
	player, err := s.players.Get(ctx, playerID)
	if err != nil {
		return err
	}
	if player == nil {
		return fmt.Errorf("player not found")
	}
	if player.SquadID != nil {
		return fmt.Errorf("already in a squad")
	}

	sq, err := s.squads.Get(ctx, squadID)
	if err != nil {
		return err
	}
	if sq == nil {
		return fmt.Errorf("squad not found")
	}
	if sq.MemberCount >= maxSquadSize {
		return fmt.Errorf("squad is full")
	}

	if err := s.players.SetSquad(ctx, playerID, &squadID); err != nil {
		return err
	}
	return s.squads.AddMember(ctx, squadID)
}

func (s *Service) Leave(ctx context.Context, playerID int64) error {
	player, err := s.players.Get(ctx, playerID)
	if err != nil {
		return err
	}
	if player == nil || player.SquadID == nil {
		return fmt.Errorf("not in a squad")
	}

	squadID := *player.SquadID
	if err := s.players.SetSquad(ctx, playerID, nil); err != nil {
		return err
	}
	return s.squads.RemoveMember(ctx, squadID)
}

func (s *Service) Get(ctx context.Context, squadID string) (*store.Squad, error) {
	return s.squads.Get(ctx, squadID)
}
