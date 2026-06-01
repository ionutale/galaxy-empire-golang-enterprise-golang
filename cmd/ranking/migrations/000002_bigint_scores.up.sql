ALTER TABLE ranking.rankings
    ALTER COLUMN total_score TYPE BIGINT,
    ALTER COLUMN buildings_score TYPE BIGINT,
    ALTER COLUMN research_score TYPE BIGINT,
    ALTER COLUMN fleet_score TYPE BIGINT,
    ALTER COLUMN defense_score TYPE BIGINT;
