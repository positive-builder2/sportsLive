CREATE TABLE IF NOT EXISTS matches (
    id         TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'live',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS balls (
    id            BIGSERIAL PRIMARY KEY,
    event_id      TEXT NOT NULL UNIQUE,
    match_id      TEXT NOT NULL REFERENCES matches(id),
    innings_id    TEXT NOT NULL,
    over_number   INT NOT NULL,
    ball_in_over  INT NOT NULL,
    batsman_id    TEXT NOT NULL,
    batsman_name  TEXT NOT NULL,
    bowler_id     TEXT NOT NULL,
    bowler_name   TEXT NOT NULL,
    runs_scored   INT NOT NULL DEFAULT 0,
    extras_wides  INT NOT NULL DEFAULT 0,
    extras_noballs INT NOT NULL DEFAULT 0,
    extras_byes   INT NOT NULL DEFAULT 0,
    extras_legbyes INT NOT NULL DEFAULT 0,
    wicket_type   TEXT,
    player_out_id TEXT,
    delivery_type TEXT,
    delivered_at  TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS balls_match_idx ON balls (match_id, delivered_at);
