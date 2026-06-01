CREATE SCHEMA IF NOT EXISTS chat;

CREATE TABLE IF NOT EXISTS chat.messages (
    id SERIAL PRIMARY KEY,
    channel VARCHAR(20) NOT NULL,
    channel_id INT NOT NULL DEFAULT 0,
    sender_id INT NOT NULL,
    sender_name VARCHAR(100) NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_channel ON chat.messages(channel, channel_id, created_at DESC);

CREATE TABLE IF NOT EXISTS chat.private_messages (
    id SERIAL PRIMARY KEY,
    sender_id INT NOT NULL,
    receiver_id INT NOT NULL,
    content TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pm_receiver ON chat.private_messages(receiver_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pm_sender ON chat.private_messages(sender_id, created_at DESC);
