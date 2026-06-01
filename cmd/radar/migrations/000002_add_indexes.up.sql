CREATE INDEX IF NOT EXISTS idx_radar_events_player_id ON radar.radar_events(player_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_radar_events_unresolved ON radar.radar_events(player_id) WHERE resolved = FALSE;
