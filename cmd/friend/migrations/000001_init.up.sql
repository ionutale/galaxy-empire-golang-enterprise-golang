CREATE SCHEMA IF NOT EXISTS friend;

CREATE TABLE IF NOT EXISTS friend.friendships (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    friend_id INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(player_id, friend_id)
);

ALTER TABLE friend.friendships ADD COLUMN IF NOT EXISTS last_active TIMESTAMPTZ;
