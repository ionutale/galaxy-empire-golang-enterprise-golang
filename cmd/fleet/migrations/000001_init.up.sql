CREATE SCHEMA IF NOT EXISTS fleet;

CREATE TABLE IF NOT EXISTS fleet.fleets (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    origin_planet_id INT NOT NULL,
    target_galaxy INT NOT NULL,
    target_system INT NOT NULL,
    target_position INT NOT NULL,
    mission VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'stationed',
    speed_pct INT NOT NULL DEFAULT 100,
    ships JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE fleet.fleets ADD COLUMN IF NOT EXISTS arrives_at TIMESTAMPTZ;

ALTER TABLE fleet.fleets ADD COLUMN IF NOT EXISTS alliance_group_id INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS fleet.attack_cooldowns (
    id SERIAL PRIMARY KEY,
    attacker_id INT NOT NULL,
    target_galaxy INT NOT NULL,
    target_system INT NOT NULL,
    target_position INT NOT NULL,
    last_attack_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(attacker_id, target_galaxy, target_system, target_position)
);

CREATE TABLE IF NOT EXISTS fleet.debris_fields (
    id SERIAL PRIMARY KEY,
    galaxy INT NOT NULL,
    system INT NOT NULL,
    position INT NOT NULL,
    metal INT NOT NULL DEFAULT 0,
    crystal INT NOT NULL DEFAULT 0,
    UNIQUE(galaxy, system, position)
);
