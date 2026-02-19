package matchmaking

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/lastclick/lastclick/internal/cache"
	"github.com/redis/go-redis/v9"
)

// Queue manages Redis-backed matchmaking queues per tier band.
type Queue struct {
	rdb *redis.Client
}

func NewQueue(rdb *redis.Client) *Queue {
	return &Queue{rdb: rdb}
}

// Enqueue adds a player to their tier's matchmaking queue.
func (q *Queue) Enqueue(ctx context.Context, playerID int64, elo int) error {
	band := Band(elo)
	key := fmt.Sprintf(cache.KeyMatchQueue, band)
	return q.rdb.ZAdd(ctx, key, redis.Z{
		Score:  float64(time.Now().UnixMilli()),
		Member: strconv.FormatInt(playerID, 10),
	}).Err()
}

// Dequeue removes a player from their tier's queue.
func (q *Queue) Dequeue(ctx context.Context, playerID int64, elo int) error {
	band := Band(elo)
	key := fmt.Sprintf(cache.KeyMatchQueue, band)
	return q.rdb.ZRem(ctx, key, strconv.FormatInt(playerID, 10)).Err()
}

// PeekBand returns up to `count` players from the specified tier band (oldest first).
func (q *Queue) PeekBand(ctx context.Context, band int, count int64) ([]int64, error) {
	key := fmt.Sprintf(cache.KeyMatchQueue, band)
	members, err := q.rdb.ZRange(ctx, key, 0, count-1).Result()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		id, err := strconv.ParseInt(m, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// PopBand atomically removes and returns up to `count` players from a band.
func (q *Queue) PopBand(ctx context.Context, band int, count int64) ([]int64, error) {
	key := fmt.Sprintf(cache.KeyMatchQueue, band)
	members, err := q.rdb.ZPopMin(ctx, key, count).Result()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if s, ok := m.Member.(string); ok {
			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				continue
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// QueueSize returns the number of players waiting in a tier band.
func (q *Queue) QueueSize(ctx context.Context, band int) (int64, error) {
	key := fmt.Sprintf(cache.KeyMatchQueue, band)
	return q.rdb.ZCard(ctx, key).Result()
}
