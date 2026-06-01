CREATE SCHEMA IF NOT EXISTS notification;

CREATE TABLE IF NOT EXISTS notification.notifications (
    id SERIAL PRIMARY KEY,
    player_id INT NOT NULL,
    category VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_player ON notification.notifications(player_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notification.notifications(player_id, is_read, created_at DESC);
