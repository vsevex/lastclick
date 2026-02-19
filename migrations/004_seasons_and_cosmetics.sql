-- +goose Up
CREATE TABLE seasons (
    id          SERIAL PRIMARY KEY,
    start_date  TIMESTAMPTZ NOT NULL,
    end_date    TIMESTAMPTZ NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX idx_seasons_active ON seasons (is_active) WHERE is_active = TRUE;

CREATE TABLE cosmetics (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    shard_cost  INT NOT NULL,
    season_id   INT REFERENCES seasons(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE player_cosmetics (
    player_id   BIGINT NOT NULL REFERENCES players(id),
    cosmetic_id INT NOT NULL REFERENCES cosmetics(id),
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (player_id, cosmetic_id)
);

-- +goose Down
DROP TABLE IF EXISTS player_cosmetics;
DROP TABLE IF EXISTS cosmetics;
DROP TABLE IF EXISTS seasons;
