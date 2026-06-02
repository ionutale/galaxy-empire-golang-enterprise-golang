ALTER TABLE ranking.player_scores
    ALTER COLUMN total_score TYPE INTEGER,
    ALTER COLUMN buildings_score TYPE INTEGER,
    ALTER COLUMN research_score TYPE INTEGER,
    ALTER COLUMN fleet_score TYPE INTEGER,
    ALTER COLUMN defense_score TYPE INTEGER;
