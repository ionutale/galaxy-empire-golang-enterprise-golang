CREATE INDEX IF NOT EXISTS idx_espionage_reports_player ON espionage.espionage_reports(player_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_espionage_reports_target ON espionage.espionage_reports(target_player_id);
