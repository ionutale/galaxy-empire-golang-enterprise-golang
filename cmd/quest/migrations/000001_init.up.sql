CREATE SCHEMA IF NOT EXISTS quest;

CREATE TABLE IF NOT EXISTS quest.quest_definitions (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(20) NOT NULL,
    reward_dm INT NOT NULL DEFAULT 0,
    reward_metal INT NOT NULL DEFAULT 0,
    reward_crystal INT NOT NULL DEFAULT 0,
    reward_gas INT NOT NULL DEFAULT 0,
    requirements JSONB NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS quest.player_quests (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    quest_id VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    progress_current INT NOT NULL DEFAULT 0,
    progress_target INT NOT NULL DEFAULT 1,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    claimed_at TIMESTAMPTZ,
    UNIQUE(player_id, quest_id)
);

CREATE INDEX IF NOT EXISTS idx_player_quests_player_id ON quest.player_quests(player_id);
