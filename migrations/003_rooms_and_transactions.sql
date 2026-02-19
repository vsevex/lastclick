-- +goose Up
CREATE TYPE room_type AS ENUM ('alpha', 'blitz');
CREATE TYPE room_state AS ENUM ('waiting', 'active', 'survival', 'finished');
CREATE TYPE tx_type AS ENUM ('entry', 'pulse', 'rake', 'payout', 'shard_grant', 'cosmetic');

CREATE TABLE rooms (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type        room_type NOT NULL,
    tier        INT NOT NULL DEFAULT 1,
    entry_cost  INT NOT NULL,
    pool        BIGINT NOT NULL DEFAULT 0,
    state       room_state NOT NULL DEFAULT 'waiting',
    winner_id   BIGINT,
    started_at  TIMESTAMPTZ,
    ended_at    TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rooms_state ON rooms (state);

CREATE TABLE transactions (
    id          BIGSERIAL PRIMARY KEY,
    player_id   BIGINT NOT NULL REFERENCES players(id),
    type        tx_type NOT NULL,
    amount      BIGINT NOT NULL,
    room_id     UUID REFERENCES rooms(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tx_player ON transactions (player_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS rooms;
DROP TYPE IF EXISTS tx_type;
DROP TYPE IF EXISTS room_state;
DROP TYPE IF EXISTS room_type;
