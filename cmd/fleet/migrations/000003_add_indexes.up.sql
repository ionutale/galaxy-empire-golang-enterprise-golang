CREATE INDEX IF NOT EXISTS idx_fleet_fleets_player_id ON fleet.fleets(player_id);
CREATE INDEX IF NOT EXISTS idx_fleet_fleets_status ON fleet.fleets(status) WHERE status NOT IN ('arrived', 'completed');
