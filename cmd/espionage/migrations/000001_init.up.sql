CREATE SCHEMA IF NOT EXISTS espionage;

CREATE TABLE IF NOT EXISTS espionage.espionage_reports (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    target_player_id INT NOT NULL,
    target_galaxy INT NOT NULL,
    target_system INT NOT NULL,
    target_position INT NOT NULL,
    detail_level INT NOT NULL DEFAULT 0,
    resources JSONB,
    fleet JSONB,
    defense JSONB,
    tech JSONB,
    report_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours'
);
