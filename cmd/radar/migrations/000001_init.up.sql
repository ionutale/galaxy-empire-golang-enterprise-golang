CREATE SCHEMA IF NOT EXISTS radar;

CREATE TABLE IF NOT EXISTS radar.radar_events (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    event_type VARCHAR(30) NOT NULL,
    source_player_id INT,
    fleet_id INT,
    target_galaxy INT NOT NULL,
    target_system INT NOT NULL,
    target_position INT NOT NULL,
    origin_galaxy INT,
    origin_system INT,
    origin_position INT,
    arrival_time TIMESTAMPTZ,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS radar.eu_x_radars (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL UNIQUE,
    moon_galaxy INT NOT NULL,
    moon_system INT NOT NULL,
    moon_position INT NOT NULL,
    level INT NOT NULL DEFAULT 1
);
