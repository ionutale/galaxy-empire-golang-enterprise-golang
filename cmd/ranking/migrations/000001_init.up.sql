CREATE SCHEMA IF NOT EXISTS ranking;

CREATE TABLE IF NOT EXISTS ranking.player_scores (
    id SERIAL PRIMARY KEY,
    player_id INTEGER UNIQUE NOT NULL,
    player_name VARCHAR(100) NOT NULL DEFAULT '',
    total_score INTEGER NOT NULL DEFAULT 0,
    fleet_score INTEGER NOT NULL DEFAULT 0,
    buildings_score INTEGER NOT NULL DEFAULT 0,
    research_score INTEGER NOT NULL DEFAULT 0,
    defense_score INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_player_scores_total
ON ranking.player_scores (total_score DESC, player_id ASC);
