package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JustinWhittecar/slic/internal/bvcalc"
	"github.com/JustinWhittecar/slic/internal/ingestion"
	_ "modernc.org/sqlite"
)

type variantRow struct {
	ID          int
	ChassisName string
	ModelCode   string
	Name        string
	BattleValue int
	TechBase    string
	Tonnage     int
}

type variantStatsRow struct {
	WalkMP       int
	RunMP        int
	JumpMP       int
	HeatSinkCount int
	HeatSinkType string
	EngineType   string
	EngineRating int
	ArmorTotal   int
	ISTotal      int
	CockpitType  sql.NullString
	GyroType     sql.NullString
	MyomerType   sql.NullString
	StructureType sql.NullString
	ArmorType    sql.NullString
}

func main() {
	dbPath := "slic.db"
	mtfRoot := "data/megamek-data/data/mekfiles/meks"

	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		mtfRoot = os.Args[2]
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Load equipment DB
	edb := loadEquipmentDB(db)
	fmt.Printf("Loaded %d equipment entries\n", len(edb.ByInternalName))

	// Build MTF file index
	mtfIndex := buildMTFIndex(mtfRoot)
	fmt.Printf("Indexed %d MTF files\n", len(mtfIndex))

	// Load variants
	variants := loadVariants(db)
	fmt.Printf("Loaded %d variants\n", len(variants))

	// Process

	var results []result
	matched := 0
	noMTF := 0

	for _, v := range variants {
		if v.BattleValue == 0 {
			continue
		}

		// Find MTF file
		mtfPath := findMTF(v.ChassisName, v.ModelCode, mtfIndex)
		if mtfPath == "" {
			noMTF++
			continue
		}

		mtf, err := ingestion.ParseMTF(mtfPath)
		if err != nil {
			continue
		}

		calc := bvcalc.Calculate(mtf, edb)
		diff := calc.FinalBV - v.BattleValue
		absDiff := int(math.Abs(float64(diff)))
		pctDiff := 0.0
		if v.BattleValue > 0 {
			pctDiff = float64(absDiff) / float64(v.BattleValue) * 100
		}

		results = append(results, result{
			variant: v,
			calcBV:  calc.FinalBV,
			pubBV:   v.BattleValue,
			diff:    diff,
			absDiff: absDiff,
			pctDiff: pctDiff,
			defBR:   calc.DefensiveBR,
			offBR:   calc.OffensiveBR,
			mtfPath: mtfPath,
			errors:  calc.Errors,
		})
		matched++
	}

	fmt.Printf("\n=== BV2 Verification Results ===\n")
	fmt.Printf("Total variants: %d\n", len(variants))
	fmt.Printf("MTF matched: %d\n", matched)
	fmt.Printf("No MTF found: %d\n", noMTF)
	fmt.Println()

	// Bucket results
	exact := 0
	within1 := 0
	within5 := 0
	within10 := 0
	within50 := 0
	over50 := 0
	within1pct := 0
	within5pct := 0
	within10pct := 0

	for _, r := range results {
		switch {
		case r.absDiff == 0:
			exact++
			within1++
			within5++
			within10++
			within50++
		case r.absDiff <= 1:
			within1++
			within5++
			within10++
			within50++
		case r.absDiff <= 5:
			within5++
			within10++
			within50++
		case r.absDiff <= 10:
			within10++
			within50++
		case r.absDiff <= 50:
			within50++
		default:
			over50++
		}
		if r.pctDiff <= 1 {
			within1pct++
		}
		if r.pctDiff <= 5 {
			within5pct++
		}
		if r.pctDiff <= 10 {
			within10pct++
		}
	}

	total := len(results)
	fmt.Printf("Exact match:  %d (%.1f%%)\n", exact, pct(exact, total))
	fmt.Printf("Within ±1:    %d (%.1f%%)\n", within1, pct(within1, total))
	fmt.Printf("Within ±5:    %d (%.1f%%)\n", within5, pct(within5, total))
	fmt.Printf("Within ±10:   %d (%.1f%%)\n", within10, pct(within10, total))
	fmt.Printf("Within ±50:   %d (%.1f%%)\n", within50, pct(within50, total))
	fmt.Printf("Over ±50:     %d (%.1f%%)\n", over50, pct(over50, total))
	fmt.Println()
	fmt.Printf("Within 1%%:    %d (%.1f%%)\n", within1pct, pct(within1pct, total))
	fmt.Printf("Within 5%%:    %d (%.1f%%)\n", within5pct, pct(within5pct, total))
	fmt.Printf("Within 10%%:   %d (%.1f%%)\n", within10pct, pct(within10pct, total))

	// Output CSV
	csvFile, err := os.Create("bv-verification.csv")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create csv: %v\n", err)
		os.Exit(1)
	}
	defer csvFile.Close()

	w := csv.NewWriter(csvFile)
	w.Write([]string{"Name", "Model", "Published BV", "Calculated BV", "Diff", "Abs Diff", "Pct Diff", "Defensive BR", "Offensive BR", "MTF Path", "Errors"})

	sort.Slice(results, func(i, j int) bool {
		return results[i].absDiff > results[j].absDiff
	})

	for _, r := range results {
		w.Write([]string{
			r.variant.ChassisName,
			r.variant.ModelCode,
			fmt.Sprintf("%d", r.pubBV),
			fmt.Sprintf("%d", r.calcBV),
			fmt.Sprintf("%d", r.diff),
			fmt.Sprintf("%d", r.absDiff),
			fmt.Sprintf("%.1f", r.pctDiff),
			fmt.Sprintf("%.1f", r.defBR),
			fmt.Sprintf("%.1f", r.offBR),
			r.mtfPath,
			strings.Join(r.errors, "; "),
		})
	}
	w.Flush()
	fmt.Printf("\nCSV written to bv-verification.csv\n")

	// Show top 20 outliers
	fmt.Printf("\n=== Top 20 Outliers (by absolute diff) ===\n")
	for i, r := range results {
		if i >= 20 {
			break
		}
		fmt.Printf("%-30s %-10s pub=%4d calc=%4d diff=%+5d (%.1f%%) def=%.0f off=%.0f\n",
			r.variant.ChassisName, r.variant.ModelCode, r.pubBV, r.calcBV, r.diff, r.pctDiff, r.defBR, r.offBR)
	}

	// Update DB with calculated values
	fmt.Printf("\nUpdating database...\n")
	updateDB(db, results)
	fmt.Printf("Done.\n")
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}

func loadEquipmentDB(db *sql.DB) *bvcalc.EquipmentDB {
	edb := &bvcalc.EquipmentDB{
		ByInternalName: make(map[string]*bvcalc.EquipInfo),
		ByName:         make(map[string][]*bvcalc.EquipInfo),
	}

	rows, err := db.Query(`SELECT id, name, type, bv, heat, rack_size, tonnage, COALESCE(internal_name,'') FROM equipment`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query equipment: %v\n", err)
		return edb
	}
	defer rows.Close()

	for rows.Next() {
		var e bvcalc.EquipInfo
		var id int
		if err := rows.Scan(&id, &e.Name, &e.Type, &e.BV, &e.Heat, &e.RackSize, &e.Tonnage, &e.InternalName); err != nil {
			continue
		}
		eCopy := e // copy for pointer safety
		if e.InternalName != "" {
			edb.ByInternalName[e.InternalName] = &eCopy
		}
		edb.ByName[e.Name] = append(edb.ByName[e.Name], &eCopy)
	}
	return edb
}

func buildMTFIndex(root string) map[string]string {
	index := make(map[string]string)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".mtf") {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(path), ".mtf")
		key := strings.ToLower(base)
		// Don't overwrite - first found wins (could be improved)
		if _, exists := index[key]; !exists {
			index[key] = path
		}
		return nil
	})
	return index
}

func findMTF(chassis, model string, index map[string]string) string {
	// Try exact match: "Chassis Model"
	key := strings.ToLower(chassis + " " + model)
	if path, ok := index[key]; ok {
		return path
	}
	// Try without special characters
	key = strings.ToLower(strings.ReplaceAll(chassis+" "+model, "'", ""))
	if path, ok := index[key]; ok {
		return path
	}
	return ""
}

func loadVariants(db *sql.DB) []variantRow {
	rows, err := db.Query(`
		SELECT v.id, c.name, v.model_code, v.name, COALESCE(v.battle_value,0), c.tech_base, c.tonnage
		FROM variants v
		JOIN chassis c ON v.chassis_id = c.id
		WHERE v.battle_value > 0
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query variants: %v\n", err)
		return nil
	}
	defer rows.Close()

	var variants []variantRow
	for rows.Next() {
		var v variantRow
		if err := rows.Scan(&v.ID, &v.ChassisName, &v.ModelCode, &v.Name, &v.BattleValue, &v.TechBase, &v.Tonnage); err != nil {
			continue
		}
		variants = append(variants, v)
	}
	return variants
}

func updateDB(db *sql.DB, results []result) {
	// Add columns if they don't exist
	db.Exec(`ALTER TABLE variant_stats ADD COLUMN calculated_bv INTEGER`)
	db.Exec(`ALTER TABLE variant_stats ADD COLUMN defensive_br REAL`)
	db.Exec(`ALTER TABLE variant_stats ADD COLUMN offensive_br REAL`)

	tx, err := db.Begin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "begin tx: %v\n", err)
		return
	}

	stmt, err := tx.Prepare(`UPDATE variant_stats SET calculated_bv=?, defensive_br=?, offensive_br=? WHERE variant_id=?`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "prepare: %v\n", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	updated := 0
	for _, r := range results {
		res, err := stmt.Exec(r.calcBV, r.defBR, r.offBR, r.variant.ID)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			updated++
		}
	}
	tx.Commit()
	fmt.Printf("Updated %d variant_stats rows\n", updated)
}

type result struct {
	variant  variantRow
	calcBV   int
	pubBV    int
	diff     int
	absDiff  int
	pctDiff  float64
	defBR    float64
	offBR    float64
	mtfPath  string
	errors   []string
}
