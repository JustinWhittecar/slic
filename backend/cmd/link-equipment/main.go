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
	equipMap, equipSlots := buildEquipmentMap(ctx, pool)
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
		key := vi.Name
		if vi.Model != "" {
			key += " " + vi.Model
		}
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

			// Count weapons per location, accounting for multi-slot items
			// Multi-slot weapons appear N consecutive times in slots (e.g. AC/20 = 10 slots)
			// We need to count: every N consecutive identical entries = 1 weapon
			weaponCounts := map[string]int{}
			prevItem := ""
			consecutiveCount := 0

			for _, item := range items {
				if shouldSkip(item) {
					// Flush any pending weapon
					if prevItem != "" && consecutiveCount > 0 {
						sl := equipSlots[prevItem]
						if sl <= 0 {
							sl = 1
						}
						weaponCounts[prevItem] += (consecutiveCount + sl - 1) / sl
					}
					prevItem = ""
					consecutiveCount = 0
					continue
				}
				// Strip suffixes: rear-mount (R), omnipod, etc.
				cleanItem := item
				cleanItem = strings.TrimSuffix(cleanItem, " (R)")
				cleanItem = strings.TrimSuffix(cleanItem, "(R)")
				cleanItem = strings.TrimSuffix(cleanItem, " (omnipod)")
				cleanItem = strings.TrimSuffix(cleanItem, " (OMNIPOD)")
				cleanItem = strings.TrimSuffix(cleanItem, " (fixed)")
				cleanItem = strings.TrimSuffix(cleanItem, " (Fixed)")
				if _, ok := equipMap[cleanItem]; !ok {
					// Unrecognized item — flush pending
					if prevItem != "" && consecutiveCount > 0 {
						sl := equipSlots[prevItem]
						if sl <= 0 {
							sl = 1
						}
						weaponCounts[prevItem] += (consecutiveCount + sl - 1) / sl
					}
					prevItem = ""
					consecutiveCount = 0
					continue
				}

				if cleanItem == prevItem {
					consecutiveCount++
				} else {
					// New weapon type — flush previous
					if prevItem != "" && consecutiveCount > 0 {
						sl := equipSlots[prevItem]
						if sl <= 0 {
							sl = 1
						}
						weaponCounts[prevItem] += (consecutiveCount + sl - 1) / sl
					}
					prevItem = cleanItem
					consecutiveCount = 1
				}
			}
			// Flush last weapon
			if prevItem != "" && consecutiveCount > 0 {
				sl := equipSlots[prevItem]
				if sl <= 0 {
					sl = 1
				}
				weaponCounts[prevItem] += (consecutiveCount + sl - 1) / sl
			}

			for itemName, qty := range weaponCounts {
				equipID := equipMap[itemName]
				_, err := pool.Exec(ctx, `
					INSERT INTO variant_equipment (variant_id, equipment_id, location, quantity)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (variant_id, equipment_id, location) DO UPDATE SET quantity = GREATEST(variant_equipment.quantity, $4)`,
					variantID, equipID, locCode, qty)
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

func buildEquipmentMap(ctx context.Context, pool *pgxpool.Pool) (map[string]int, map[string]int) {
	m := map[string]int{}
	slots := map[string]int{}

	// Primary: internal_name and display name from equipment table
	rows, err := pool.Query(ctx, "SELECT id, internal_name, name, COALESCE(slots,1) FROM equipment WHERE internal_name IS NOT NULL")
	if err != nil {
		log.Fatalf("Query equipment: %v", err)
	}
	for rows.Next() {
		var id, sl int
		var internalName, name string
		rows.Scan(&id, &internalName, &name, &sl)
		m[internalName] = id
		slots[internalName] = sl
		if name != "" {
			if _, exists := m[name]; !exists {
				m[name] = id
				slots[name] = sl
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
				// Get slots for this equipment
				var sl int
				pool.QueryRow(ctx, "SELECT COALESCE(slots,1) FROM equipment WHERE id=$1", id).Scan(&sl)
				slots[ln] = sl
			}
		}
		rows2.Close()
	}

	return m, slots
}
