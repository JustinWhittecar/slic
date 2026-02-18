-- Efficiency score: monte carlo sim results
ALTER TABLE variant_stats ADD COLUMN IF NOT EXISTS efficiency_score real DEFAULT 0;
ALTER TABLE variant_stats ADD COLUMN IF NOT EXISTS offense_turns real DEFAULT 0;
ALTER TABLE variant_stats ADD COLUMN IF NOT EXISTS defense_turns real DEFAULT 0;
