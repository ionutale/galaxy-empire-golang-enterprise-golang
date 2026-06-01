CREATE SCHEMA IF NOT EXISTS planet;

CREATE TABLE IF NOT EXISTS planet.planets (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL DEFAULT 'Homeworld',
    galaxy INTEGER NOT NULL DEFAULT 1,
    system INTEGER NOT NULL DEFAULT 1,
    position INTEGER NOT NULL DEFAULT 7,
    metal INTEGER NOT NULL DEFAULT 500,
    crystal INTEGER NOT NULL DEFAULT 300,
    gas INTEGER NOT NULL DEFAULT 200,
    energy INTEGER NOT NULL DEFAULT 50,
    max_fields INTEGER NOT NULL DEFAULT 40,
    resources_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS planet.player_progress (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES planet.planets(user_id) ON DELETE CASCADE UNIQUE,
    vip_points INTEGER NOT NULL DEFAULT 0,
    total_resources_produced BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS planet.buildings (
    id SERIAL PRIMARY KEY,
    planet_id INTEGER NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    level INTEGER NOT NULL DEFAULT 0,
    UNIQUE(planet_id, type)
);

CREATE TABLE IF NOT EXISTS planet.construction_queue (
    id SERIAL PRIMARY KEY,
    planet_id INTEGER NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE,
    building_type VARCHAR(50) NOT NULL,
    target_level INTEGER NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completes_at TIMESTAMPTZ NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'upgrade'
);

CREATE TABLE IF NOT EXISTS planet.player_technologies (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES planet.planets(user_id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    level INTEGER NOT NULL DEFAULT 0,
    UNIQUE(user_id, type)
);

CREATE TABLE IF NOT EXISTS planet.player_ships (
    id SERIAL PRIMARY KEY,
    planet_id INT NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE,
    ship_type VARCHAR(50) NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    UNIQUE(planet_id, ship_type)
);

CREATE TABLE IF NOT EXISTS planet.player_defenses (
    id SERIAL PRIMARY KEY,
    planet_id INT NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE,
    defense_type VARCHAR(50) NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    UNIQUE(planet_id, defense_type)
);

CREATE TABLE IF NOT EXISTS planet.moon_buildings (
    id SERIAL PRIMARY KEY,
    moon_galaxy INT NOT NULL,
    moon_system INT NOT NULL,
    moon_position INT NOT NULL,
    type VARCHAR(50) NOT NULL,
    level INT NOT NULL DEFAULT 0,
    UNIQUE(moon_galaxy, moon_system, moon_position, type)
);

CREATE TABLE IF NOT EXISTS planet.wormhole_generators (
    id SERIAL PRIMARY KEY,
    moon_galaxy INT NOT NULL,
    moon_system INT NOT NULL,
    moon_position INT NOT NULL,
    level INT NOT NULL DEFAULT 1,
    linked_galaxy INT,
    linked_system INT,
    linked_position INT,
    cooldown_until TIMESTAMPTZ,
    UNIQUE(moon_galaxy, moon_system, moon_position)
);

CREATE TABLE IF NOT EXISTS planet.stargate_links (
    id SERIAL PRIMARY KEY,
    planet_id INT NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE UNIQUE,
    target_planet_id INT NOT NULL REFERENCES planet.planets(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
