ALTER TABLE research.research_queue
    ADD COLUMN IF NOT EXISTS planet_id INT NOT NULL DEFAULT 0;
