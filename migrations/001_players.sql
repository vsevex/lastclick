-- +goose Up
CREATE TABLE players (
    id              BIGINT PRIMARY KEY,
    username        TEXT NOT NULL DEFAULT '',
    elo             INT NOT NULL DEFAULT 1200,
    lifetime_elo    INT NOT NULL DEFAULT 1200,
    efficiency_avg  DOUBLE PRECISION NOT NULL DEFAULT 0,
    stars_balance   BIGINT NOT NULL DEFAULT 0,
    shards_balance  BIGINT NOT NULL DEFAULT 0,
    squad_id        UUID,
    prestige_mult   DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_players_elo ON players (elo);
CREATE INDEX idx_players_squad ON players (squad_id);

-- +goose Down
DROP TABLE IF EXISTS players;
