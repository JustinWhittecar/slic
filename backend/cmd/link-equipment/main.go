package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/JustinWhittecar/slic/internal/db"
	"github.com/JustinWhittecar/slic/internal/ingestion"
	"github.com/jackc/pgx/v5/pgxpool"
)

var locationMap = map[string]string{
	"Left Arm":        "LA",
	"Right Arm":       "RA",
	"Left Torso":      "LT",
	"Right Torso":     "RT",
	"Center Torso":    "CT",
	"Head":            "HD",
	"Left Leg":        "LL",
	"Right Leg":       "RL",
	"Front Left Leg":  "FLL",
	"Front Right Leg": "FRL",
	"Rear Left Leg":   "RLL",
	"Rear Right Leg":  "RRL",
}

func main() {
	mtfDir := os.Args[1] // path to mtf files root

	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("DB: %v", err)
	}
	defer pool.Close()

	// Build equipment lookup: weapon display name -> equipment id
	equipMap := buildEquipmentMap(ctx, pool)
	fmt.Printf("Equipment map: %d entries\n", len(equipMap))

	// Clear existing links
	pool.Exec(ctx, "DELETE FROM variant_equipment")

	// Get all variants
	type variantInfo struct {
		ID    int
		Name  string
		Model string
	}
	rows, err := pool.Query(ctx, `
		SELECT v.id, c.name, v.model_code 
		FROM variants v JOIN chassis c ON c.id = v.chassis_id`)
	if err != nil {
		log.Fatalf("Query variants: %v", err)
	}
	variantMap := map[string]int{}
	for rows.Next() {
		var vi variantInfo
		rows.Scan(&vi.ID, &vi.Name, &vi.Model)
		key := vi.Name
		if vi.Model != "" {
			key += " " + vi.Model
		}
		variantMap[key] = vi.ID
	}
	rows.Close()
	fmt.Printf("Variants in DB: %d\n", len(variantMap))

	// Walk MTF files and use the Weapons: section as source of truth
	linked := 0
	skippedVariants := 0
	totalEquipLinks := 0
	unmatchedWeapons := map[string]int{}

	filepath.Walk(mtfDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".mtf") {
			return nil
		}

		data, err := ingestion.ParseMTF(path)
		if err != nil {
			return nil
		}

		variantID, ok := variantMap[data.FullName()]
		if !ok {
			skippedVariants++
			return nil
		}

		// Count weapons by (name, location) from the Weapons: section
		type weaponKey struct {
			name string
			loc  string
		}
		counts := map[weaponKey]int{}
		for _, w := range data.Weapons {
			locCode, ok := locationMap[w.Location]
			if !ok {
				continue
			}
			name := w.Name
			// Strip leading quantity number: "1 ISMediumLaser" -> "ISMediumLaser"
			// Some entries have "N WeaponName" format where N is a digit
			if len(name) > 2 && name[0] >= '0' && name[0] <= '9' && name[1] == ' ' {
				name = name[2:]
			} else if len(name) > 3 && name[0] >= '0' && name[0] <= '9' && name[1] >= '0' && name[1] <= '9' && name[2] == ' ' {
				name = name[3:]
			}
			counts[weaponKey{name, locCode}]++
		}

		// Detect targeting computer from crit slots
		hasTC := false
		for _, items := range data.LocationEquipment {
			for _, item := range items {
				lower := strings.ToLower(item)
				if strings.Contains(lower, "targeting computer") || strings.Contains(lower, "targetingcomputer") {
					hasTC = true
					break
				}
			}
			if hasTC {
				break
			}
		}
		if hasTC {
			pool.Exec(ctx, `UPDATE variant_stats SET has_targeting_computer = TRUE WHERE variant_id = $1`, variantID)
		}

		for wk, qty := range counts {
			equipID, ok := equipMap[wk.name]
			if !ok {
				unmatchedWeapons[wk.name]++
				continue
			}
			_, err := pool.Exec(ctx, `
				INSERT INTO variant_equipment (variant_id, equipment_id, location, quantity)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (variant_id, equipment_id, location) DO UPDATE SET quantity = GREATEST(variant_equipment.quantity, $4)`,
				variantID, equipID, wk.loc, qty)
			if err != nil {
				log.Printf("Link %s %s@%s: %v", data.FullName(), wk.name, wk.loc, err)
				continue
			}
			totalEquipLinks++
		}
		linked++
		return nil
	})

	fmt.Printf("Linked equipment for %d variants (%d links total, %d variants not found in DB)\n",
		linked, totalEquipLinks, skippedVariants)

	if len(unmatchedWeapons) > 0 {
		fmt.Printf("\nUnmatched weapons (%d unique):\n", len(unmatchedWeapons))
		// Sort by count descending
		type kv struct {
			name  string
			count int
		}
		var sorted []kv
		for k, v := range unmatchedWeapons {
			sorted = append(sorted, kv{k, v})
		}
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].count > sorted[i].count {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		for _, s := range sorted {
			if s.count >= 3 {
				fmt.Printf("  %4d  %s\n", s.count, s.name)
			}
		}
	}
}

func buildEquipmentMap(ctx context.Context, pool *pgxpool.Pool) map[string]int {
	m := map[string]int{}

	// Primary: display name from equipment table
	rows, err := pool.Query(ctx, "SELECT id, name FROM equipment")
	if err != nil {
		log.Fatalf("Query equipment: %v", err)
	}
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		m[name] = id
	}
	rows.Close()

	// All lookup names from equipment_lookup table
	rows2, err := pool.Query(ctx, "SELECT equipment_id, lookup_name FROM equipment_lookup")
	if err != nil {
		log.Printf("Warning: equipment_lookup table not found")
		return m
	}
	for rows2.Next() {
		var id int
		var ln string
		rows2.Scan(&id, &ln)
		if _, exists := m[ln]; !exists {
			m[ln] = id
		}
	}
	rows2.Close()

	return m
}
