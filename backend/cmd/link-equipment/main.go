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

// Structural items to skip when linking equipment
var skipItems = map[string]bool{
	"Shoulder": true, "Upper Arm Actuator": true, "Lower Arm Actuator": true,
	"Hand Actuator": true, "Hip": true, "Upper Leg Actuator": true,
	"Lower Leg Actuator": true, "Foot Actuator": true,
	"-Empty-": true, "Engine": true, "Gyro": true,
	"Life Support": true, "Sensors": true, "Cockpit": true,
	"Fusion Engine": true, "Heat Sink": true,
}

// Prefixes/substrings indicating structural items
var skipPrefixes = []string{
	"Endo Steel", "Endo-Steel", "Ferro-Fibrous", "Ferro Fibrous",
	"CASE", "IS Endo Steel", "IS Endo-Steel", "Clan Endo Steel",
	"IS Ferro-Fibrous", "Clan Ferro-Fibrous", "IS Light Ferro-Fibrous",
	"IS Heavy Ferro-Fibrous", "Reactive Armor", "Reflective Armor",
	"IS Stealth", "Clan Stealth",
}

func shouldSkip(item string) bool {
	if skipItems[item] {
		return true
	}
	lower := strings.ToLower(item)
	if strings.Contains(lower, "heat sink") || strings.Contains(lower, "heatsink") {
		return true
	}
	if strings.Contains(lower, "jump jet") {
		return true
	}
	if strings.Contains(lower, "actuator") {
		return true
	}
	for _, p := range skipPrefixes {
		if strings.EqualFold(item, p) || strings.HasPrefix(item, p) {
			return true
		}
	}
	// Ammo
	if strings.Contains(lower, "ammo") {
		return true
	}
	// MASC, TSM, etc
	if item == "ISMASC" || item == "CLMASC" || item == "TSM" || item == "IS Endo-Composite" || item == "Clan Endo-Composite" {
		return true
	}
	return false
}

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

	// Build equipment lookup: internal_name -> id, also lookup_names
	equipMap := buildEquipmentMap(ctx, pool)
	fmt.Printf("Equipment map: %d entries\n", len(equipMap))

	// Clear existing links
	pool.Exec(ctx, "DELETE FROM variant_equipment")

	// Get all variants with their names for matching
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
	variantMap := map[string]int{} // "Chassis Model" -> id
	for rows.Next() {
		var vi variantInfo
		rows.Scan(&vi.ID, &vi.Name, &vi.Model)
		key := vi.Name + " " + vi.Model
		variantMap[key] = vi.ID
	}
	rows.Close()
	fmt.Printf("Variants in DB: %d\n", len(variantMap))

	// Walk MTF files
	linked := 0
	skippedVariants := 0
	totalEquipLinks := 0

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

		// Process location equipment
		for locName, items := range data.LocationEquipment {
			locCode, ok := locationMap[locName]
			if !ok {
				continue
			}

			// Count unique weapons per location (multi-slot items appear multiple times)
			weaponCounts := map[string]int{}
			weaponSeen := map[string]int{} // track consecutive occurrences
			prevItem := ""

			for _, item := range items {
				if shouldSkip(item) {
					prevItem = ""
					continue
				}
				// Strip rear-mount suffix
				cleanItem := strings.TrimSuffix(item, " (R)")
				cleanItem = strings.TrimSuffix(cleanItem, "(R)")
				if _, ok := equipMap[cleanItem]; !ok {
					prevItem = ""
					continue
				}
				item = cleanItem
				if item == prevItem {
					// Same item consecutive = multi-slot, don't count again
					continue
				}
				weaponCounts[item]++
				weaponSeen[item]++
				prevItem = item
			}

			for itemName, qty := range weaponCounts {
				equipID := equipMap[itemName]
				_, err := pool.Exec(ctx, `
					INSERT INTO variant_equipment (variant_id, equipment_id, location, quantity)
					VALUES ($1, $2, $3, $4)`, variantID, equipID, locCode, qty)
				if err != nil {
					log.Printf("Link %s %s: %v", data.FullName(), itemName, err)
					continue
				}
				totalEquipLinks++
			}
		}
		linked++
		return nil
	})

	fmt.Printf("Linked equipment for %d variants (%d links total, %d variants not found in DB)\n",
		linked, totalEquipLinks, skippedVariants)
}

func buildEquipmentMap(ctx context.Context, pool *pgxpool.Pool) map[string]int {
	m := map[string]int{}

	// Primary: internal_name and display name from equipment table
	rows, err := pool.Query(ctx, "SELECT id, internal_name, name FROM equipment WHERE internal_name IS NOT NULL")
	if err != nil {
		log.Fatalf("Query equipment: %v", err)
	}
	for rows.Next() {
		var id int
		var internalName, name string
		rows.Scan(&id, &internalName, &name)
		m[internalName] = id
		if name != "" {
			if _, exists := m[name]; !exists {
				m[name] = id
			}
		}
	}
	rows.Close()

	// All lookup names from equipment_lookup table
	rows2, err := pool.Query(ctx, "SELECT equipment_id, lookup_name FROM equipment_lookup")
	if err != nil {
		log.Printf("Warning: equipment_lookup table not found, skipping lookups")
	} else {
		for rows2.Next() {
			var id int
			var ln string
			rows2.Scan(&id, &ln)
			if _, exists := m[ln]; !exists {
				m[ln] = id
			}
		}
		rows2.Close()
	}

	return m
}
