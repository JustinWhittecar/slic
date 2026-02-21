CREATE TABLE IF NOT EXISTS external_ratings (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER REFERENCES variants(id),
    source TEXT NOT NULL,
    rating TEXT,
    url TEXT,
    notes TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_external_ratings_variant ON external_ratings(variant_id);
CREATE INDEX IF NOT EXISTS idx_external_ratings_source ON external_ratings(source);
