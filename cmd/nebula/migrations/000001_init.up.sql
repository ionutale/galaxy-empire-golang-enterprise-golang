CREATE SCHEMA IF NOT EXISTS nebula;

CREATE TABLE IF NOT EXISTS nebula.expeditions (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    fleet_id INT NOT NULL DEFAULT 0,
    galaxy INT NOT NULL DEFAULT 0,
    system INT NOT NULL DEFAULT 0,
    position INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'travelling',
    ships_sent JSONB NOT NULL DEFAULT '{}',
    ships_lost JSONB NOT NULL DEFAULT '{}',
    ships_found JSONB NOT NULL DEFAULT '{}',
    resources_found JSONB NOT NULL DEFAULT '{}',
    dark_matter_found INT NOT NULL DEFAULT 0,
    outcome VARCHAR(50),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    travel_duration INT NOT NULL DEFAULT 300,
    explore_duration INT NOT NULL DEFAULT 1800,
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS nebula.player_dark_matter (
    player_id INT PRIMARY KEY,
    balance INT NOT NULL DEFAULT 0,
    total_earned INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS nebula.dm_transactions (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    amount INT NOT NULL,
    balance_after INT NOT NULL,
    reason VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS nebula.player_credits (
    player_id INT PRIMARY KEY,
    balance INT NOT NULL DEFAULT 0,
    total_earned INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS nebula.credits_transactions (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    amount INT NOT NULL,
    balance_after INT NOT NULL,
    reason VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS nebula.player_commanders (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    commander_type VARCHAR(20) NOT NULL,
    level INT NOT NULL DEFAULT 1,
    hired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE(player_id, commander_type)
);

CREATE TABLE IF NOT EXISTS nebula.daily_gift_streak (
    player_id INT PRIMARY KEY,
    streak_day INT NOT NULL DEFAULT 0,
    last_claim_date DATE NOT NULL DEFAULT CURRENT_DATE,
    consecutive_days INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS nebula.daily_tasks (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    target_amount INT NOT NULL,
    progress INT NOT NULL DEFAULT 0,
    reward_dm INT NOT NULL DEFAULT 0,
    reward_resources JSONB NOT NULL DEFAULT '{}',
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    claimed BOOLEAN NOT NULL DEFAULT FALSE,
    assigned_date DATE NOT NULL DEFAULT CURRENT_DATE,
    rerolls_used INT NOT NULL DEFAULT 0,
    UNIQUE(player_id, task_type, assigned_date)
);
