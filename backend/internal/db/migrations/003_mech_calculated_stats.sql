ALTER TABLE variant_stats ADD COLUMN IF NOT EXISTS tmm INTEGER DEFAULT 0;
ALTER TABLE variant_stats ADD COLUMN IF NOT EXISTS armor_coverage_pct REAL DEFAULT 0;
ALTER TABLE variant_stats ADD COLUMN IF NOT EXISTS heat_neutral_damage REAL DEFAULT 0;
ALTER TABLE variant_stats ADD COLUMN IF NOT EXISTS heat_neutral_range TEXT DEFAULT '';
ALTER TABLE variant_stats ADD COLUMN IF NOT EXISTS max_damage REAL DEFAULT 0;
ALTER TABLE variant_stats ADD COLUMN IF NOT EXISTS effective_heat_neutral_damage REAL DEFAULT 0;
