CREATE SCHEMA IF NOT EXISTS combat;

CREATE TABLE IF NOT EXISTS combat.combat_reports (
    id SERIAL PRIMARY KEY,
    attacker_player_id INT NOT NULL,
    defender_player_id INT NOT NULL,
    target_galaxy INT NOT NULL,
    target_system INT NOT NULL,
    target_position INT NOT NULL,
    attacker_ships_before JSONB,
    defender_ships_before JSONB,
    attacker_ships_after JSONB,
    defender_ships_after JSONB,
    rounds JSONB,
    attacker_won BOOLEAN NOT NULL,
    attacker_loot JSONB,
    defender_lost_resources JSONB,
    debris_metal INT NOT NULL DEFAULT 0,
    debris_crystal INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days'
);

ALTER TABLE combat.combat_reports ADD COLUMN IF NOT EXISTS moon_created BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE combat.combat_reports ADD COLUMN IF NOT EXISTS moon_size INT NOT NULL DEFAULT 0;

ALTER TABLE combat.combat_reports ADD COLUMN IF NOT EXISTS missile_result JSONB;

CREATE SCHEMA IF NOT EXISTS fleet;

CREATE TABLE IF NOT EXISTS fleet.moons (
    id SERIAL PRIMARY KEY,
    galaxy INT NOT NULL,
    system INT NOT NULL,
    position INT NOT NULL,
    player_id INT,
    name VARCHAR(100) NOT NULL DEFAULT 'Moon',
    size INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(galaxy, system, position)
);
