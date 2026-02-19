package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lastclick/lastclick/internal/cache"
	"github.com/lastclick/lastclick/internal/config"
	"github.com/lastclick/lastclick/internal/game"
	"github.com/lastclick/lastclick/internal/room"
	"github.com/lastclick/lastclick/internal/server"
	"github.com/lastclick/lastclick/internal/squad"
	"github.com/lastclick/lastclick/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	rdb, err := cache.NewRedis(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Error("connect redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Stores
	playerStore := store.NewPlayerStore(db)
	txStore := store.NewTransactionStore(db)
	squadStore := store.NewSquadStore(db)

	// Room manager
	rooms := room.NewManager()

	// End-of-round callback
	onEnd := func(r *room.Room) {
		endCtx, endCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer endCancel()

		rake := game.RakeAmount(r.Pool)
		payout := game.WinnerPayout(r.Pool)

		if r.WinnerID != 0 {
			_ = playerStore.UpdateBalance(endCtx, r.WinnerID, payout, 0)
			roomID := r.ID
			_ = txStore.Record(endCtx, r.WinnerID, store.TxPayout, payout, &roomID)
		}

		for _, p := range r.Players {
			if p.ID == r.WinnerID {
				continue
			}
			shards := game.ShardsFromBurn(p.StarsSpent, r.VolatilityMul)
			if shards > 0 {
				_ = playerStore.UpdateBalance(endCtx, p.ID, 0, shards)
				roomID := r.ID
				_ = txStore.Record(endCtx, p.ID, store.TxShardGrant, shards, &roomID)
			}
		}

		warChest := game.WarChestContribution(rake)
		if warChest > 0 {
			for _, p := range r.Players {
				player, err := playerStore.Get(endCtx, p.ID)
				if err != nil || player == nil || player.SquadID == nil {
					continue
				}
				share := warChest / int64(r.PlayerCount())
				_ = squadStore.AddToWarChest(endCtx, *player.SquadID, share)
			}
		}

		logger.Info("room finished",
			"room", r.ID,
			"winner", r.WinnerID,
			"pool", r.Pool,
			"rake", rake,
			"payout", payout,
		)
	}

	// Wire engine and hub (circular dependency resolved via SetHub)
	engine := game.NewEngine(rooms, nil, logger, onEnd)
	hub := server.NewHub(cfg.BotToken, cfg.Env == "development", engine, logger)
	engine.SetHub(hub)

	srv := server.New(cfg, db, rdb, hub, logger)
	srv.SetPlayerStore(playerStore)

	// Squad service
	squadSvc := squad.NewService(squadStore, playerStore, logger)
	srv.SetSquadService(squadSvc)

	httpSrv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      srv.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := httpSrv.Shutdown(shutCtx); err != nil {
		logger.Error("shutdown", "err", err)
	}
}
