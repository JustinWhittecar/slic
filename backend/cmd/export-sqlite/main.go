package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://slic:slic@localhost:5432/slic?sslmode=disable"
	}
	pg, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("pg connect: %v", err)
	}
	defer pg.Close()

	outPath := "slic.db"
	if len(os.Args) > 1 {
		outPath = os.Args[1]
	}
	os.Remove(outPath)
	sl, err := sql.Open("sqlite", outPath)
	if err != nil {
		log.Fatalf("sqlite open: %v", err)
	}
	defer sl.Close()

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	} {
		sl.Exec(pragma)
	}

	// Create tables
	for _, ddl := range []string{
		`CREATE TABLE chassis (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			tonnage INTEGER NOT NULL,
			tech_base TEXT NOT NULL,
			sarna_url TEXT,
			alternate_name TEXT
		)`,
		`CREATE TABLE variants (
			id INTEGER PRIMARY KEY,
			chassis_id INTEGER NOT NULL REFERENCES chassis(id) ON DELETE CASCADE,
			model_code TEXT NOT NULL,
			name TEXT NOT NULL,
			battle_value INTEGER,
			intro_year INTEGER,
			era TEXT,
			role TEXT,
			mul_id INTEGER,
			config TEXT,
			source TEXT,
			rules_level INTEGER
		)`,
		`CREATE TABLE variant_stats (
			variant_id INTEGER PRIMARY KEY REFERENCES variants(id) ON DELETE CASCADE,
			walk_mp INTEGER NOT NULL,
			run_mp INTEGER NOT NULL,
			jump_mp INTEGER NOT NULL DEFAULT 0,
			armor_total INTEGER NOT NULL,
			internal_structure_total INTEGER NOT NULL,
			heat_sink_count INTEGER NOT NULL,
			heat_sink_type TEXT NOT NULL DEFAULT 'Single',
			engine_type TEXT NOT NULL,
			engine_rating INTEGER NOT NULL,
			cockpit_type TEXT,
			gyro_type TEXT,
			myomer_type TEXT,
			structure_type TEXT,
			armor_type TEXT,
			tmm INTEGER DEFAULT 0,
			armor_coverage_pct REAL DEFAULT 0,
			heat_neutral_damage REAL DEFAULT 0,
			heat_neutral_range TEXT DEFAULT '',
			max_damage REAL DEFAULT 0,
			effective_heat_neutral_damage REAL DEFAULT 0,
			tonnage INTEGER DEFAULT 0,
			game_damage REAL DEFAULT 0,
			has_targeting_computer INTEGER DEFAULT 0,
			combat_rating REAL DEFAULT 0,
			offense_turns REAL DEFAULT 0,
			defense_turns REAL DEFAULT 0
		)`,
		`CREATE TABLE equipment (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			damage REAL,
			heat INTEGER,
			min_range INTEGER,
			short_range INTEGER,
			medium_range INTEGER,
			long_range INTEGER,
			tonnage REAL NOT NULL,
			slots INTEGER NOT NULL,
			internal_name TEXT,
			bv INTEGER,
			rack_size INTEGER DEFAULT 0,
			expected_damage REAL DEFAULT 0,
			damage_per_ton REAL DEFAULT 0,
			damage_per_heat REAL DEFAULT 0,
			extreme_range INTEGER DEFAULT 0,
			tech_base TEXT,
			to_hit_modifier INTEGER DEFAULT 0,
			damage_short INTEGER DEFAULT 0,
			damage_medium INTEGER DEFAULT 0,
			damage_long INTEGER DEFAULT 0,
			effective_damage_short REAL DEFAULT 0,
			effective_damage_medium REAL DEFAULT 0,
			effective_damage_long REAL DEFAULT 0,
			effective_dps_ton REAL DEFAULT 0,
			effective_dps_heat REAL DEFAULT 0
		)`,
		`CREATE TABLE variant_equipment (
			id INTEGER PRIMARY KEY,
			variant_id INTEGER NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
			equipment_id INTEGER NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
			location TEXT NOT NULL,
			quantity INTEGER NOT NULL DEFAULT 1,
			UNIQUE(variant_id, equipment_id, location)
		)`,
		`CREATE TABLE eras (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			start_year INTEGER NOT NULL,
			end_year INTEGER
		)`,
		`CREATE TABLE factions (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			abbreviation TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE variant_era_factions (
			variant_id INTEGER NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
			era_id INTEGER NOT NULL REFERENCES eras(id) ON DELETE CASCADE,
			faction_id INTEGER NOT NULL REFERENCES factions(id) ON DELETE CASCADE,
			PRIMARY KEY (variant_id, era_id, faction_id)
		)`,
		`CREATE TABLE equipment_lookup (
			equipment_id INTEGER NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
			lookup_name TEXT NOT NULL PRIMARY KEY
		)`,
		`CREATE TABLE model_sources (
			id INTEGER PRIMARY KEY,
			variant_id INTEGER NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
			source_type TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT
		)`,
		// Indexes
		`CREATE INDEX idx_variants_chassis ON variants(chassis_id)`,
		`CREATE INDEX idx_variants_intro_year ON variants(intro_year)`,
		`CREATE INDEX idx_variants_mul_id ON variants(mul_id)`,
		`CREATE INDEX idx_variants_role ON variants(role)`,
		`CREATE INDEX idx_equipment_internal_name ON equipment(internal_name)`,
	} {
		if _, err := sl.Exec(ddl); err != nil {
			log.Fatalf("DDL error: %v\n%s", err, ddl)
		}
	}

	// Copy tables
	copyTable(ctx, pg, sl, "chassis",
		"SELECT id, name, tonnage, tech_base, sarna_url, alternate_name FROM chassis",
		"INSERT INTO chassis (id, name, tonnage, tech_base, sarna_url, alternate_name) VALUES (?,?,?,?,?,?)", 6)

	copyTable(ctx, pg, sl, "variants",
		"SELECT id, chassis_id, model_code, name, battle_value, intro_year, era, role, mul_id, config, source, rules_level FROM variants",
		"INSERT INTO variants (id, chassis_id, model_code, name, battle_value, intro_year, era, role, mul_id, config, source, rules_level) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)", 12)

	copyTable(ctx, pg, sl, "variant_stats",
		`SELECT variant_id, walk_mp, run_mp, jump_mp, armor_total, internal_structure_total,
		        heat_sink_count, heat_sink_type, engine_type, engine_rating,
		        cockpit_type, gyro_type, myomer_type, structure_type, armor_type,
		        tmm, armor_coverage_pct, heat_neutral_damage, heat_neutral_range,
		        max_damage, effective_heat_neutral_damage, tonnage, game_damage,
		        has_targeting_computer, combat_rating, offense_turns, defense_turns
		 FROM variant_stats`,
		`INSERT INTO variant_stats (variant_id, walk_mp, run_mp, jump_mp, armor_total, internal_structure_total,
		        heat_sink_count, heat_sink_type, engine_type, engine_rating,
		        cockpit_type, gyro_type, myomer_type, structure_type, armor_type,
		        tmm, armor_coverage_pct, heat_neutral_damage, heat_neutral_range,
		        max_damage, effective_heat_neutral_damage, tonnage, game_damage,
		        has_targeting_computer, combat_rating, offense_turns, defense_turns)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, 27)

	copyTable(ctx, pg, sl, "equipment",
		`SELECT id, name, type, damage, heat, min_range, short_range, medium_range, long_range,
		        tonnage, slots, internal_name, bv, rack_size, expected_damage, damage_per_ton,
		        damage_per_heat, extreme_range, tech_base, to_hit_modifier,
		        damage_short, damage_medium, damage_long,
		        effective_damage_short, effective_damage_medium, effective_damage_long,
		        effective_dps_ton, effective_dps_heat FROM equipment`,
		`INSERT INTO equipment (id, name, type, damage, heat, min_range, short_range, medium_range, long_range,
		        tonnage, slots, internal_name, bv, rack_size, expected_damage, damage_per_ton,
		        damage_per_heat, extreme_range, tech_base, to_hit_modifier,
		        damage_short, damage_medium, damage_long,
		        effective_damage_short, effective_damage_medium, effective_damage_long,
		        effective_dps_ton, effective_dps_heat) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, 28)

	copyTable(ctx, pg, sl, "variant_equipment",
		"SELECT id, variant_id, equipment_id, location, quantity FROM variant_equipment",
		"INSERT INTO variant_equipment (id, variant_id, equipment_id, location, quantity) VALUES (?,?,?,?,?)", 5)

	copyTable(ctx, pg, sl, "eras",
		"SELECT id, name, start_year, end_year FROM eras",
		"INSERT INTO eras (id, name, start_year, end_year) VALUES (?,?,?,?)", 4)

	copyTable(ctx, pg, sl, "factions",
		"SELECT id, name, abbreviation FROM factions",
		"INSERT INTO factions (id, name, abbreviation) VALUES (?,?,?)", 3)

	copyTable(ctx, pg, sl, "variant_era_factions",
		"SELECT variant_id, era_id, faction_id FROM variant_era_factions",
		"INSERT INTO variant_era_factions (variant_id, era_id, faction_id) VALUES (?,?,?)", 3)

	copyTable(ctx, pg, sl, "equipment_lookup",
		"SELECT equipment_id, lookup_name FROM equipment_lookup",
		"INSERT INTO equipment_lookup (equipment_id, lookup_name) VALUES (?,?)", 2)

	copyTable(ctx, pg, sl, "model_sources",
		"SELECT id, variant_id, source_type, name, url FROM model_sources",
		"INSERT INTO model_sources (id, variant_id, source_type, name, url) VALUES (?,?,?,?,?)", 5)

	log.Println("Export complete!")
}

func copyTable(ctx context.Context, pg *pgxpool.Pool, sl *sql.DB, name, selectQ, insertQ string, cols int) {
	rows, err := pg.Query(ctx, selectQ)
	if err != nil {
		log.Fatalf("select %s: %v", name, err)
	}
	defer rows.Close()

	tx, _ := sl.Begin()
	stmt, _ := tx.Prepare(insertQ)
	count := 0
	for rows.Next() {
		vals := make([]any, cols)
		ptrs := make([]any, cols)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			log.Fatalf("scan %s: %v", name, err)
		}
		if _, err := stmt.Exec(vals...); err != nil {
			log.Fatalf("insert %s: %v", name, err)
		}
		count++
	}
	tx.Commit()
	fmt.Printf("  %s: %d rows\n", name, count)
}
