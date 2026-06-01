CREATE INDEX IF NOT EXISTS idx_notifications_player_unread ON notification.notifications(player_id, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_player_created ON notification.notifications(player_id, created_at DESC);
