package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Squad struct {
	ID          string
	Name        string
	WarChest    int64
	MemberCount int
	SeasonRank  int
	CreatedAt   time.Time
}

type SquadStore struct {
	db *pgxpool.Pool
}

func NewSquadStore(db *pgxpool.Pool) *SquadStore {
	return &SquadStore{db: db}
}

func (s *SquadStore) Create(ctx context.Context, name string) (*Squad, error) {
	sq := &Squad{}
	err := s.db.QueryRow(ctx, `
		INSERT INTO squads (name) VALUES ($1)
		RETURNING id, name, war_chest, member_count, season_rank, created_at
	`, name).Scan(&sq.ID, &sq.Name, &sq.WarChest, &sq.MemberCount, &sq.SeasonRank, &sq.CreatedAt)
	return sq, err
}

func (s *SquadStore) Get(ctx context.Context, id string) (*Squad, error) {
	sq := &Squad{}
	err := s.db.QueryRow(ctx, `
		SELECT id, name, war_chest, member_count, season_rank, created_at
		FROM squads WHERE id = $1
	`, id).Scan(&sq.ID, &sq.Name, &sq.WarChest, &sq.MemberCount, &sq.SeasonRank, &sq.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return sq, err
}

func (s *SquadStore) AddMember(ctx context.Context, squadID string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE squads SET member_count = member_count + 1 WHERE id = $1
	`, squadID)
	return err
}

func (s *SquadStore) RemoveMember(ctx context.Context, squadID string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE squads SET member_count = GREATEST(member_count - 1, 0) WHERE id = $1
	`, squadID)
	return err
}

func (s *SquadStore) AddToWarChest(ctx context.Context, squadID string, amount int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE squads SET war_chest = war_chest + $2 WHERE id = $1
	`, squadID, amount)
	return err
}

func (s *SquadStore) ResetSeason(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `UPDATE squads SET season_rank = 0`)
	return err
}

func (s *SquadStore) TopByWarChest(ctx context.Context, limit int) ([]Squad, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, war_chest, member_count, season_rank, created_at
		FROM squads ORDER BY war_chest DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Squad
	for rows.Next() {
		var sq Squad
		if err := rows.Scan(&sq.ID, &sq.Name, &sq.WarChest, &sq.MemberCount, &sq.SeasonRank, &sq.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sq)
	}
	return out, rows.Err()
}
