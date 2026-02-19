package leaderboard

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lastclick/lastclick/internal/cache"
	"github.com/redis/go-redis/v9"
)

type Entry struct {
	PlayerID int64
	Score    float64
	Rank     int64
}

type Service struct {
	rdb *redis.Client
}

func NewService(rdb *redis.Client) *Service {
	return &Service{rdb: rdb}
}

// UpdateEfficiency sets a player's efficiency score for the current season.
func (s *Service) UpdateEfficiency(ctx context.Context, seasonID int, playerID int64, efficiency float64) error {
	key := fmt.Sprintf(cache.KeyLeaderboard, seasonID)
	return s.rdb.ZAdd(ctx, key, redis.Z{
		Score:  efficiency,
		Member: strconv.FormatInt(playerID, 10),
	}).Err()
}

// TopEfficiency returns the top N players by efficiency for a season.
func (s *Service) TopEfficiency(ctx context.Context, seasonID int, count int64) ([]Entry, error) {
	key := fmt.Sprintf(cache.KeyLeaderboard, seasonID)
	return s.topFromSortedSet(ctx, key, count)
}

// UpdateSquadRank sets a squad's rank score.
func (s *Service) UpdateSquadRank(ctx context.Context, seasonID int, squadID string, score float64) error {
	key := fmt.Sprintf(cache.KeySquadBoard, seasonID)
	return s.rdb.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: squadID,
	}).Err()
}

// TopSquads returns the top N squads by rank for a season.
func (s *Service) TopSquads(ctx context.Context, seasonID int, count int64) ([]Entry, error) {
	key := fmt.Sprintf(cache.KeySquadBoard, seasonID)
	return s.topFromSortedSet(ctx, key, count)
}

// PlayerRank returns a player's rank and score for a season.
func (s *Service) PlayerRank(ctx context.Context, seasonID int, playerID int64) (*Entry, error) {
	key := fmt.Sprintf(cache.KeyLeaderboard, seasonID)
	member := strconv.FormatInt(playerID, 10)

	rank, err := s.rdb.ZRevRank(ctx, key, member).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	score, err := s.rdb.ZScore(ctx, key, member).Result()
	if err != nil {
		return nil, err
	}

	return &Entry{PlayerID: playerID, Score: score, Rank: rank + 1}, nil
}

// ResetSeason removes leaderboard data for a given season.
func (s *Service) ResetSeason(ctx context.Context, seasonID int) error {
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, fmt.Sprintf(cache.KeyLeaderboard, seasonID))
	pipe.Del(ctx, fmt.Sprintf(cache.KeySquadBoard, seasonID))
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Service) topFromSortedSet(ctx context.Context, key string, count int64) ([]Entry, error) {
	results, err := s.rdb.ZRevRangeWithScores(ctx, key, 0, count-1).Result()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(results))
	for i, z := range results {
		member, _ := z.Member.(string)
		id, _ := strconv.ParseInt(member, 10, 64)
		entries = append(entries, Entry{
			PlayerID: id,
			Score:    z.Score,
			Rank:     int64(i + 1),
		})
	}
	return entries, nil
}
