package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Player struct {
	ID            int64
	Username      string
	Elo           int
	LifetimeElo   int
	EfficiencyAvg float64
	StarsBalance  int64
	ShardsBalance int64
	SquadID       *string
	PrestigeMult  float64
	CreatedAt     time.Time
}

type PlayerStore struct {
	db *pgxpool.Pool
}

func NewPlayerStore(db *pgxpool.Pool) *PlayerStore {
	return &PlayerStore{db: db}
}

func (s *PlayerStore) Upsert(ctx context.Context, id int64, username string) (*Player, error) {
	p := &Player{}
	err := s.db.QueryRow(ctx, `
		INSERT INTO players (id, username) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username
		RETURNING id, username, elo, lifetime_elo, efficiency_avg,
		          stars_balance, shards_balance, squad_id, prestige_mult, created_at
	`, id, username).Scan(
		&p.ID, &p.Username, &p.Elo, &p.LifetimeElo, &p.EfficiencyAvg,
		&p.StarsBalance, &p.ShardsBalance, &p.SquadID, &p.PrestigeMult, &p.CreatedAt,
	)
	return p, err
}

func (s *PlayerStore) Get(ctx context.Context, id int64) (*Player, error) {
	p := &Player{}
	err := s.db.QueryRow(ctx, `
		SELECT id, username, elo, lifetime_elo, efficiency_avg,
		       stars_balance, shards_balance, squad_id, prestige_mult, created_at
		FROM players WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Username, &p.Elo, &p.LifetimeElo, &p.EfficiencyAvg,
		&p.StarsBalance, &p.ShardsBalance, &p.SquadID, &p.PrestigeMult, &p.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *PlayerStore) UpdateBalance(ctx context.Context, id int64, starsDelta, shardsDelta int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE players
		SET stars_balance = stars_balance + $2,
		    shards_balance = shards_balance + $3
		WHERE id = $1
	`, id, starsDelta, shardsDelta)
	return err
}

func (s *PlayerStore) UpdateElo(ctx context.Context, id int64, newElo int) error {
	_, err := s.db.Exec(ctx, `
		UPDATE players
		SET elo = $2,
		    lifetime_elo = GREATEST(lifetime_elo, $2)
		WHERE id = $1
	`, id, newElo)
	return err
}

func (s *PlayerStore) UpdateEfficiency(ctx context.Context, id int64, avg float64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE players SET efficiency_avg = $2 WHERE id = $1
	`, id, avg)
	return err
}

func (s *PlayerStore) SetSquad(ctx context.Context, playerID int64, squadID *string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE players SET squad_id = $2 WHERE id = $1
	`, playerID, squadID)
	return err
}

func (s *PlayerStore) ResetSeasonalStats(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		UPDATE players SET elo = 1200, efficiency_avg = 0, prestige_mult = 1.0
	`)
	return err
}
