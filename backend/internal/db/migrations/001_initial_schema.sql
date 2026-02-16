CREATE TABLE chassis (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    tonnage INTEGER NOT NULL CHECK (tonnage >= 20 AND tonnage <= 100),
    tech_base TEXT NOT NULL CHECK (tech_base IN ('Inner Sphere', 'Clan', 'Mixed')),
    sarna_url TEXT
);

CREATE TABLE eras (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    start_year INTEGER NOT NULL,
    end_year INTEGER
);

CREATE TABLE factions (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    abbreviation TEXT NOT NULL UNIQUE
);

CREATE TABLE equipment (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    damage REAL,
    heat INTEGER,
    min_range INTEGER,
    short_range INTEGER,
    medium_range INTEGER,
    long_range INTEGER,
    tonnage REAL NOT NULL,
    slots INTEGER NOT NULL
);

CREATE TABLE variants (
    id SERIAL PRIMARY KEY,
    chassis_id INTEGER NOT NULL REFERENCES chassis(id) ON DELETE CASCADE,
    model_code TEXT NOT NULL,
    name TEXT NOT NULL,
    battle_value INTEGER,
    intro_year INTEGER,
    era TEXT,
    role TEXT
);

CREATE INDEX idx_variants_chassis ON variants(chassis_id);
CREATE INDEX idx_variants_intro_year ON variants(intro_year);
CREATE INDEX idx_variants_role ON variants(role);

CREATE TABLE variant_equipment (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    equipment_id INTEGER NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
    location TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE variant_stats (
    variant_id INTEGER PRIMARY KEY REFERENCES variants(id) ON DELETE CASCADE,
    walk_mp INTEGER NOT NULL,
    run_mp INTEGER NOT NULL,
    jump_mp INTEGER NOT NULL DEFAULT 0,
    armor_total INTEGER NOT NULL,
    internal_structure_total INTEGER NOT NULL,
    heat_sink_count INTEGER NOT NULL,
    heat_sink_type TEXT NOT NULL DEFAULT 'Single',
    engine_type TEXT NOT NULL,
    engine_rating INTEGER NOT NULL
);

CREATE TABLE variant_era_factions (
    variant_id INTEGER NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    era_id INTEGER NOT NULL REFERENCES eras(id) ON DELETE CASCADE,
    faction_id INTEGER NOT NULL REFERENCES factions(id) ON DELETE CASCADE,
    PRIMARY KEY (variant_id, era_id, faction_id)
);

CREATE TABLE model_sources (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL CHECK (source_type IN ('iwm', 'forcepack')),
    name TEXT NOT NULL,
    url TEXT
);

-- Seed eras
INSERT INTO eras (name, start_year, end_year) VALUES
    ('Age of War', 2005, 2570),
    ('Star League', 2571, 2780),
    ('Early Succession Wars', 2781, 2900),
    ('Late Succession Wars', 2901, 3049),
    ('Clan Invasion', 3049, 3061),
    ('Civil War', 3062, 3067),
    ('Jihad', 3067, 3081),
    ('Dark Age', 3081, 3150),
    ('ilClan', 3151, NULL);
