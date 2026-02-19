package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TxType string

const (
	TxEntry      TxType = "entry"
	TxPulse      TxType = "pulse"
	TxRake       TxType = "rake"
	TxPayout     TxType = "payout"
	TxShardGrant TxType = "shard_grant"
	TxCosmetic   TxType = "cosmetic"
)

type Transaction struct {
	ID        int64
	PlayerID  int64
	Type      TxType
	Amount    int64
	RoomID    *string
	CreatedAt time.Time
}

type TransactionStore struct {
	db *pgxpool.Pool
}

func NewTransactionStore(db *pgxpool.Pool) *TransactionStore {
	return &TransactionStore{db: db}
}

func (s *TransactionStore) Record(ctx context.Context, playerID int64, txType TxType, amount int64, roomID *string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO transactions (player_id, type, amount, room_id) VALUES ($1, $2, $3, $4)
	`, playerID, txType, amount, roomID)
	return err
}

func (s *TransactionStore) PlayerHistory(ctx context.Context, playerID int64, limit int) ([]Transaction, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, player_id, type, amount, room_id, created_at
		FROM transactions WHERE player_id = $1
		ORDER BY created_at DESC LIMIT $2
	`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.PlayerID, &t.Type, &t.Amount, &t.RoomID, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
