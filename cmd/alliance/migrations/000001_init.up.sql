CREATE SCHEMA IF NOT EXISTS alliance;

CREATE TABLE IF NOT EXISTS alliance.alliances (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    tag VARCHAR(10) NOT NULL UNIQUE,
    founder_id INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS alliance.members (
    id SERIAL PRIMARY KEY,
    alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE,
    player_id INT NOT NULL UNIQUE,
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(alliance_id, player_id)
);

ALTER TABLE alliance.members ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS alliance.bank (
    id SERIAL PRIMARY KEY,
    alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE UNIQUE,
    metal BIGINT NOT NULL DEFAULT 0,
    crystal BIGINT NOT NULL DEFAULT 0,
    gas BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS alliance.audit_log (
    id SERIAL PRIMARY KEY,
    alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE,
    player_id INT NOT NULL,
    action VARCHAR(100) NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS alliance.bulletins (
    id SERIAL PRIMARY KEY,
    alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE,
    author_player_id INT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS alliance.shared_reports (
    id SERIAL PRIMARY KEY,
    alliance_id INT NOT NULL REFERENCES alliance.alliances(id) ON DELETE CASCADE,
    report_id INT NOT NULL,
    shared_by INT NOT NULL,
    shared_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
