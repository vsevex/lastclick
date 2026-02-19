package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Season struct {
	ID        int
	StartDate time.Time
	EndDate   time.Time
	IsActive  bool
}

type SeasonStore struct {
	db *pgxpool.Pool
}

func NewSeasonStore(db *pgxpool.Pool) *SeasonStore {
	return &SeasonStore{db: db}
}

func (s *SeasonStore) Active(ctx context.Context) (*Season, error) {
	se := &Season{}
	err := s.db.QueryRow(ctx, `
		SELECT id, start_date, end_date, is_active FROM seasons WHERE is_active = TRUE
	`).Scan(&se.ID, &se.StartDate, &se.EndDate, &se.IsActive)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return se, err
}

func (s *SeasonStore) Create(ctx context.Context, start, end time.Time) (*Season, error) {
	// Deactivate current season
	_, _ = s.db.Exec(ctx, `UPDATE seasons SET is_active = FALSE WHERE is_active = TRUE`)

	se := &Season{}
	err := s.db.QueryRow(ctx, `
		INSERT INTO seasons (start_date, end_date, is_active) VALUES ($1, $2, TRUE)
		RETURNING id, start_date, end_date, is_active
	`, start, end).Scan(&se.ID, &se.StartDate, &se.EndDate, &se.IsActive)
	return se, err
}
