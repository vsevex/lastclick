-- +goose Up
CREATE TABLE squads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    war_chest       BIGINT NOT NULL DEFAULT 0,
    member_count    INT NOT NULL DEFAULT 0,
    season_rank     INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE players
    ADD CONSTRAINT fk_players_squad FOREIGN KEY (squad_id) REFERENCES squads(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE players DROP CONSTRAINT IF EXISTS fk_players_squad;
DROP TABLE IF EXISTS squads;
