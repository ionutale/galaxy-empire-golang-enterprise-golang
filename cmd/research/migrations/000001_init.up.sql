CREATE SCHEMA IF NOT EXISTS research;

CREATE TABLE IF NOT EXISTS research.research_queue (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    tech_type VARCHAR(50) NOT NULL,
    target_level INT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completes_at TIMESTAMPTZ NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    cancelled BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE(player_id, tech_type)
);
