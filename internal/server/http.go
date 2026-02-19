package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lastclick/lastclick/internal/config"
	"github.com/lastclick/lastclick/internal/leaderboard"
	"github.com/lastclick/lastclick/internal/squad"
	"github.com/lastclick/lastclick/internal/store"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg         *config.Config
	db          *pgxpool.Pool
	rdb         *redis.Client
	hub         *Hub
	logger      *slog.Logger
	mux         *http.ServeMux
	players     *store.PlayerStore
	squadSvc    *squad.Service
	leaderboard *leaderboard.Service
	seasons     *store.SeasonStore
	metrics     *Metrics
}

func New(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client, hub *Hub, logger *slog.Logger) *Server {
	s := &Server{
		cfg:         cfg,
		db:          db,
		rdb:         rdb,
		hub:         hub,
		logger:      logger,
		mux:         http.NewServeMux(),
		leaderboard: leaderboard.NewService(rdb),
		seasons:     store.NewSeasonStore(db),
		metrics:     NewMetrics(),
	}
	s.routes()
	return s
}

func (s *Server) SetPlayerStore(ps *store.PlayerStore) {
	s.players = ps
}

func (s *Server) SetSquadService(svc *squad.Service) {
	s.squadSvc = svc
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /metrics", s.metrics.ServeHTTP)
	s.mux.Handle("GET /ws", s.hub)

	// Squad endpoints
	s.mux.HandleFunc("POST /api/squads", s.handleCreateSquad)
	s.mux.HandleFunc("POST /api/squads/join", s.handleJoinSquad)
	s.mux.HandleFunc("POST /api/squads/leave", s.handleLeaveSquad)
	s.mux.HandleFunc("GET /api/squads/{id}", s.handleGetSquad)

	// Player endpoint
	s.mux.HandleFunc("GET /api/player/{id}", s.handleGetPlayer)

	// Leaderboard endpoints
	s.mux.HandleFunc("GET /api/leaderboard/players", s.handlePlayerLeaderboard)
	s.mux.HandleFunc("GET /api/leaderboard/squads", s.handleSquadLeaderboard)
	s.mux.HandleFunc("GET /api/leaderboard/rank/{playerID}", s.handlePlayerRank)

	// Static files for Mini App
	s.mux.Handle("GET /", http.FileServer(http.Dir("web")))
}

func (s *Server) handleGetPlayer(w http.ResponseWriter, r *http.Request) {
	if s.players == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	pidStr := r.PathValue("id")
	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		http.Error(w, "bad player id", http.StatusBadRequest)
		return
	}
	player, err := s.players.Get(r.Context(), pid)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if player == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, player)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status := map[string]string{"status": "ok"}

	if err := s.db.Ping(ctx); err != nil {
		status["db"] = "down"
		status["status"] = "degraded"
	} else {
		status["db"] = "ok"
	}

	if err := s.rdb.Ping(ctx).Err(); err != nil {
		status["redis"] = "down"
		status["status"] = "degraded"
	} else {
		status["redis"] = "ok"
	}

	w.Header().Set("Content-Type", "application/json")
	if status["status"] != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if err := json.NewEncoder(w).Encode(status); err != nil {
		s.logger.Error("write json", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleCreateSquad(w http.ResponseWriter, r *http.Request) {
	if s.squadSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Name      string `json:"name"`
		FounderID int64  `json:"founder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	sq, err := s.squadSvc.Create(r.Context(), req.Name, req.FounderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, sq)
}

func (s *Server) handleJoinSquad(w http.ResponseWriter, r *http.Request) {
	if s.squadSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		PlayerID int64  `json:"player_id"`
		SquadID  string `json:"squad_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.squadSvc.Join(r.Context(), req.PlayerID, req.SquadID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"status": "joined"})
}

func (s *Server) handleLeaveSquad(w http.ResponseWriter, r *http.Request) {
	if s.squadSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		PlayerID int64 `json:"player_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.squadSvc.Leave(r.Context(), req.PlayerID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"status": "left"})
}

func (s *Server) handleGetSquad(w http.ResponseWriter, r *http.Request) {
	if s.squadSvc == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	id := r.PathValue("id")
	sq, err := s.squadSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if sq == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, sq)
}

func (s *Server) handlePlayerLeaderboard(w http.ResponseWriter, r *http.Request) {
	season, err := s.seasons.Active(r.Context())
	if err != nil || season == nil {
		writeJSON(w, []any{})
		return
	}
	count := int64(50)
	if c := r.URL.Query().Get("count"); c != "" {
		if n, err := strconv.ParseInt(c, 10, 64); err == nil && n > 0 && n <= 100 {
			count = n
		}
	}
	entries, err := s.leaderboard.TopEfficiency(r.Context(), season.ID, count)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, entries)
}

func (s *Server) handleSquadLeaderboard(w http.ResponseWriter, r *http.Request) {
	season, err := s.seasons.Active(r.Context())
	if err != nil || season == nil {
		writeJSON(w, []any{})
		return
	}
	count := int64(50)
	if c := r.URL.Query().Get("count"); c != "" {
		if n, err := strconv.ParseInt(c, 10, 64); err == nil && n > 0 && n <= 100 {
			count = n
		}
	}
	entries, err := s.leaderboard.TopSquads(r.Context(), season.ID, count)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, entries)
}

func (s *Server) handlePlayerRank(w http.ResponseWriter, r *http.Request) {
	season, err := s.seasons.Active(r.Context())
	if err != nil || season == nil {
		http.Error(w, "no active season", http.StatusNotFound)
		return
	}
	pidStr := r.PathValue("playerID")
	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		http.Error(w, "bad player id", http.StatusBadRequest)
		return
	}
	entry, err := s.leaderboard.PlayerRank(r.Context(), season.ID, pid)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if entry == nil {
		http.Error(w, "not ranked", http.StatusNotFound)
		return
	}
	writeJSON(w, entry)
}

func (s *Server) Handler() http.Handler {
	limiter := NewRateLimiter(30, 60)
	return ChainMiddleware(s.mux,
		RecoveryMiddleware(s.logger),
		LoggingMiddleware(s.logger),
		RateLimitMiddleware(limiter, s.logger),
	)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
