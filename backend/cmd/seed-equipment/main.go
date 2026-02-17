package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/JustinWhittecar/slic/internal/db"
)

type Weapon struct {
	InternalName  string   `json:"internal_name"`
	LookupNames   []string `json:"lookup_names"`
	Name          string   `json:"name"`
	Heat          int      `json:"heat"`
	Damage        int      `json:"damage"`
	RackSize      int      `json:"rack_size"`
	MinRange      int      `json:"min_range"`
	ShortRange    int      `json:"short_range"`
	MediumRange   int      `json:"medium_range"`
	LongRange     int      `json:"long_range"`
	ExtremeRange  int      `json:"extreme_range"`
	Tonnage       float64  `json:"tonnage"`
	CriticalSlots int      `json:"critical_slots"`
	BV            int      `json:"bv"`
	Type          string   `json:"type"`
	ToHitModifier int      `json:"to_hit_modifier"`
	DamageShort   int      `json:"damage_short"`
	DamageMedium  int      `json:"damage_medium"`
	DamageLong    int      `json:"damage_long"`
	ExpectedDamage float64 `json:"expected_damage"`
	DamagePerTon   float64 `json:"damage_per_ton"`
	DamagePerHeat  float64 `json:"damage_per_heat"`
	EffDamageShort  float64 `json:"effective_damage_short"`
	EffDamageMedium float64 `json:"effective_damage_medium"`
	EffDamageLong   float64 `json:"effective_damage_long"`
	EffDPSTon       float64 `json:"effective_dps_ton"`
	EffDPSHeat      float64 `json:"effective_dps_heat"`
}

func main() {
	input := flag.String("input", "", "weapons JSON file")
	flag.Parse()
	if *input == "" {
		log.Fatal("Usage: --input <weapons.json>")
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalf("Read: %v", err)
	}
	var weapons []Weapon
	if err := json.Unmarshal(data, &weapons); err != nil {
		log.Fatalf("JSON: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("DB: %v", err)
	}
	defer pool.Close()

	// Clear existing equipment
	pool.Exec(ctx, "DELETE FROM variant_equipment")
	pool.Exec(ctx, "DELETE FROM equipment")

	// Create lookup_names table if not exists
	pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS equipment_lookup (
		equipment_id INTEGER NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
		lookup_name TEXT NOT NULL,
		PRIMARY KEY (lookup_name)
	)`)
	pool.Exec(ctx, "DELETE FROM equipment_lookup")

	count := 0
	for _, w := range weapons {
		var equipID int
		err := pool.QueryRow(ctx, `
			INSERT INTO equipment (name, type, damage, heat, min_range, short_range, medium_range, long_range,
				tonnage, slots, internal_name, bv, rack_size, expected_damage, damage_per_ton, damage_per_heat,
				extreme_range, to_hit_modifier, damage_short, damage_medium, damage_long,
				effective_damage_short, effective_damage_medium, effective_damage_long,
				effective_dps_ton, effective_dps_heat)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)
			RETURNING id`,
			w.Name, w.Type, w.Damage, w.Heat, w.MinRange, w.ShortRange, w.MediumRange, w.LongRange,
			w.Tonnage, w.CriticalSlots, w.InternalName, w.BV, w.RackSize, w.ExpectedDamage,
			w.DamagePerTon, w.DamagePerHeat, w.ExtremeRange, w.ToHitModifier,
			w.DamageShort, w.DamageMedium, w.DamageLong,
			w.EffDamageShort, w.EffDamageMedium, w.EffDamageLong, w.EffDPSTon, w.EffDPSHeat).Scan(&equipID)
		if err != nil {
			log.Printf("Insert %s: %v", w.InternalName, err)
			continue
		}

		// Insert all name variants as lookups
		allNames := map[string]bool{w.InternalName: true, w.Name: true}
		for _, ln := range w.LookupNames {
			allNames[ln] = true
		}
		for n := range allNames {
			if n == "" {
				continue
			}
			pool.Exec(ctx, `INSERT INTO equipment_lookup (equipment_id, lookup_name) VALUES ($1, $2) ON CONFLICT DO NOTHING`, equipID, n)
		}

		count++
	}
	fmt.Printf("Seeded %d equipment items\n", count)
}
