package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedis(ctx context.Context, addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return rdb, nil
}

const (
	KeyRoomState   = "room:%s:state"
	KeyRoomPlayers = "room:%s:players"
	KeyMatchQueue  = "matchmaking:tier:%d"
	KeyLeaderboard = "leaderboard:efficiency:season:%d"
	KeySquadBoard  = "leaderboard:squad:season:%d"
)
