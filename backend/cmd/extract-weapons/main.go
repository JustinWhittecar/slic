package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

	// Calculated
	ExpectedDamage       float64 `json:"expected_damage"`
	DamagePerTon         float64 `json:"damage_per_ton"`
	DamagePerHeat        float64 `json:"damage_per_heat"`
	EffDamageShort       float64 `json:"effective_damage_short"`
	EffDamageMedium      float64 `json:"effective_damage_medium"`
	EffDamageLong        float64 `json:"effective_damage_long"`
	EffDPSTon            float64 `json:"effective_dps_ton"`
	EffDPSHeat           float64 `json:"effective_dps_heat"`
}

var skipDirs = map[string]bool{
	"infantry": true, "battleArmor": true, "bayWeapons": true,
	"capitalWeapons": true, "subCapitalWeapons": true, "bombs": true,
	"attacks": true, "handlers": true, "unofficial": true, "c3": true,
	"defensivePods": true, "tag": true,
}

var (
	reInternalName = regexp.MustCompile(`setInternalName\("([^"]+)"\)`)
	reLookupName   = regexp.MustCompile(`addLookupName\("([^"]+)"\)`)
	reNameField    = regexp.MustCompile(`(?m)^\s*(?:this\.)?name\s*=\s*"([^"]+)"`)
	reHeat         = regexp.MustCompile(`(?m)^\s*(?:this\.)?heat\s*=\s*(\d+)`)
	reDamage       = regexp.MustCompile(`(?m)^\s*(?:this\.)?damage\s*=\s*(\d+)`)
	reRackSize     = regexp.MustCompile(`(?m)^\s*(?:this\.)?rackSize\s*=\s*(\d+)`)
	reMinRange     = regexp.MustCompile(`(?m)^\s*(?:this\.)?minimumRange\s*=\s*(\d+)`)
	reShortRange   = regexp.MustCompile(`(?m)^\s*(?:this\.)?shortRange\s*=\s*(\d+)`)
	reMediumRange  = regexp.MustCompile(`(?m)^\s*(?:this\.)?mediumRange\s*=\s*(\d+)`)
	reLongRange    = regexp.MustCompile(`(?m)^\s*(?:this\.)?longRange\s*=\s*(\d+)`)
	reExtremeRange = regexp.MustCompile(`(?m)^\s*(?:this\.)?extremeRange\s*=\s*(\d+)`)
	reTonnage      = regexp.MustCompile(`(?m)^\s*(?:this\.)?tonnage\s*=\s*([\d.]+)`)
	reCritSlots    = regexp.MustCompile(`(?m)^\s*(?:this\.)?criticalSlots\s*=\s*(\d+)`)
	reBV           = regexp.MustCompile(`(?m)^\s*(?:this\.)?bv\s*=\s*([\d.]+)`)
	reToHitMod     = regexp.MustCompile(`(?m)^\s*(?:this\.)?toHitModifier\s*=\s*(-?\d+)`)
	reDamageShort  = regexp.MustCompile(`(?m)^\s*(?:this\.)?damageShort\s*=\s*(\d+)`)
	reDamageMedium = regexp.MustCompile(`(?m)^\s*(?:this\.)?damageMedium\s*=\s*(\d+)`)
	reDamageLong   = regexp.MustCompile(`(?m)^\s*(?:this\.)?damageLong\s*=\s*(\d+)`)
	reDamageCluster = regexp.MustCompile(`(?m)^\s*(?:this\.)?damage\s*=\s*(DAMAGE_BY_RACK|DAMAGE_MISSILE|DAMAGE_BY_CLUSTER_TABLE|DAMAGE_VARIABLE)`)
)

// 2d6 probability of rolling >= target
var pHit = map[int]float64{
	2: 1.0, 3: 0.972, 4: 0.917, 5: 0.833, 6: 0.722,
	7: 0.583, 8: 0.417, 9: 0.278, 10: 0.167, 11: 0.083, 12: 0.028,
}

func hitProb(target int) float64 {
	if target <= 2 {
		return 1.0
	}
	if target >= 13 {
		return 0.0
	}
	return pHit[target]
}

func extractInt(re *regexp.Regexp, content string) int {
	m := re.FindStringSubmatch(content)
	if m == nil {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

func extractFloat(re *regexp.Regexp, content string) float64 {
	m := re.FindStringSubmatch(content)
	if m == nil {
		return 0
	}
	v, _ := strconv.ParseFloat(m[1], 64)
	return v
}

func classifyType(relPath string) string {
	lower := strings.ToLower(relPath)
	switch {
	case strings.Contains(lower, "laser") || strings.Contains(lower, "ppc") || strings.Contains(lower, "flamer"):
		return "energy"
	case strings.Contains(lower, "autocannon") || strings.Contains(lower, "gauss") || strings.Contains(lower, "mg"):
		return "ballistic"
	case strings.Contains(lower, "lrm") || strings.Contains(lower, "srm") || strings.Contains(lower, "missile") ||
		strings.Contains(lower, "rocketlauncher") || strings.Contains(lower, "thuunderbolt"):
		return "missile"
	case strings.Contains(lower, "artillery"):
		return "other"
	case strings.Contains(lower, "mortar"):
		return "other"
	default:
		return "other"
	}
}

func isClusterWeapon(content string) bool {
	return reDamageCluster.MatchString(content)
}

func isVariableDamage(content string) bool {
	return strings.Contains(content, "DAMAGE_VARIABLE")
}

func isStreakSRM(name, path string) bool {
	lower := strings.ToLower(name + " " + path)
	return strings.Contains(lower, "streak") && strings.Contains(lower, "srm")
}

func isStreakLRM(name, path string) bool {
	lower := strings.ToLower(name + " " + path)
	return strings.Contains(lower, "streak") && strings.Contains(lower, "lrm")
}

func isUltraAC(name, path string) bool {
	lower := strings.ToLower(name + " " + path)
	return strings.Contains(lower, "ultra")
}

func isRotaryAC(name, path string) bool {
	lower := strings.ToLower(name + " " + path)
	return strings.Contains(lower, "rotary") || strings.Contains(lower, "rac")
}

func isSRM(name, path string) bool {
	lower := strings.ToLower(name + " " + path)
	return strings.Contains(lower, "srm") || strings.Contains(lower, "srt") // SRT = torpedo variant
}

func calcExpectedDamage(w *Weapon, content, path string) float64 {
	if isVariableDamage(content) {
		if w.DamageMedium > 0 {
			return float64(w.DamageMedium)
		}
		return float64(w.Damage)
	}

	nameAndPath := w.Name + " " + w.InternalName + " " + path

	if isStreakSRM(nameAndPath, path) {
		return float64(w.RackSize) * 2.0 // all hit, 2 dmg each
	}
	if isStreakLRM(nameAndPath, path) {
		return float64(w.RackSize) // all hit, 1 dmg each
	}
	if isUltraAC(nameAndPath, path) && w.Damage > 0 {
		return float64(w.Damage) * 1.5
	}
	if isRotaryAC(nameAndPath, path) && w.Damage > 0 {
		return float64(w.Damage) * 3.5
	}

	// Cluster weapons: rackSize > 0, damage == 0 (inherited from parent as DAMAGE_MISSILE etc)
	// Also catch explicit cluster constants in file
	isCluster := isClusterWeapon(content) || (w.RackSize > 0 && w.Damage == 0)
	if isCluster && w.RackSize > 0 {
		if isSRM(nameAndPath, path) {
			// SRM: each missile does 2 damage, use cluster table for hits
			return float64(w.RackSize) * 0.58 * 2.0
		}
		// LRM, HAG, MRM, etc: 1 damage per missile
		return float64(w.RackSize) * 0.58
	}

	if w.Damage > 0 {
		return float64(w.Damage)
	}
	return 0
}

func calcEffectiveDamage(w *Weapon, content, path string) {
	// Base target: Gunnery 4 + walked(+1) + target TMM(+2) = 7
	// Range mods: short +0, medium +2, long +4
	baseTN := 7
	thm := w.ToHitModifier

	shortTN := baseTN + 0 + thm
	medTN := baseTN + 2 + thm
	longTN := baseTN + 4 + thm

	// Determine expected damage at each range bracket
	var expShort, expMed, expLong float64

	if isVariableDamage(content) {
		expShort = float64(w.DamageShort)
		expMed = float64(w.DamageMedium)
		expLong = float64(w.DamageLong)
	} else {
		expShort = w.ExpectedDamage
		expMed = w.ExpectedDamage
		expLong = w.ExpectedDamage
	}

	w.EffDamageShort = math.Round(expShort*hitProb(shortTN)*100) / 100
	w.EffDamageMedium = math.Round(expMed*hitProb(medTN)*100) / 100
	w.EffDamageLong = math.Round(expLong*hitProb(longTN)*100) / 100

	if w.Tonnage > 0 {
		w.EffDPSTon = math.Round(w.EffDamageMedium/w.Tonnage*100) / 100
	}
	if w.Heat > 0 {
		w.EffDPSHeat = math.Round(w.EffDamageMedium/float64(w.Heat)*100) / 100
	}
}

func parseWeaponFile(path, baseDir string) (*Weapon, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(content)

	// Extract name field first
	var nameVal string
	if m := reNameField.FindStringSubmatch(src); m != nil {
		nameVal = m[1]
	}

	// Get internal name: try literal string first, then variable reference
	var internalName string
	if inMatch := reInternalName.FindStringSubmatch(src); inMatch != nil {
		internalName = inMatch[1]
	} else if strings.Contains(src, "setInternalName(name)") || strings.Contains(src, "setInternalName(this.name)") {
		// Variable reference to name field
		internalName = nameVal
	}

	if internalName == "" {
		return nil, nil // abstract/base class
	}

	w := &Weapon{
		InternalName: internalName,
		Name:         nameVal,
		LookupNames:  []string{},
	}

	// Lookup names
	for _, m := range reLookupName.FindAllStringSubmatch(src, -1) {
		w.LookupNames = append(w.LookupNames, m[1])
	}

	w.Heat = extractInt(reHeat, src)
	w.Damage = extractInt(reDamage, src)
	w.RackSize = extractInt(reRackSize, src)
	w.MinRange = extractInt(reMinRange, src)
	w.ShortRange = extractInt(reShortRange, src)
	w.MediumRange = extractInt(reMediumRange, src)
	w.LongRange = extractInt(reLongRange, src)
	w.ExtremeRange = extractInt(reExtremeRange, src)
	w.Tonnage = extractFloat(reTonnage, src)
	w.CriticalSlots = extractInt(reCritSlots, src)
	w.BV = int(extractFloat(reBV, src))
	w.ToHitModifier = extractInt(reToHitMod, src)
	w.DamageShort = extractInt(reDamageShort, src)
	w.DamageMedium = extractInt(reDamageMedium, src)
	w.DamageLong = extractInt(reDamageLong, src)

	// Handle negative toHitModifier (regex won't catch negative with \d+)
	if m := reToHitMod.FindStringSubmatch(src); m != nil {
		w.ToHitModifier, _ = strconv.Atoi(m[1])
	}

	// Classify
	relPath, _ := filepath.Rel(baseDir, path)
	w.Type = classifyType(relPath)

	// If cluster weapon, damage field is symbolic; set to 0
	if isClusterWeapon(src) {
		w.Damage = 0
	}

	// Calculated fields
	w.ExpectedDamage = math.Round(calcExpectedDamage(w, src, relPath)*100) / 100
	if w.Tonnage > 0 {
		w.DamagePerTon = math.Round(w.ExpectedDamage/w.Tonnage*100) / 100
	}
	if w.Heat > 0 {
		w.DamagePerHeat = math.Round(w.ExpectedDamage/float64(w.Heat)*100) / 100
	}

	calcEffectiveDamage(w, src, relPath)

	return w, nil
}

func main() {
	dirFlag := flag.String("dir", "", "root weapons directory")
	outFlag := flag.String("output", "", "output JSON file")
	flag.Parse()

	if *dirFlag == "" || *outFlag == "" {
		log.Fatal("Usage: --dir <path> --output <path>")
	}

	var weapons []Weapon
	err := filepath.Walk(*dirFlag, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".java") {
			return nil
		}

		w, err := parseWeaponFile(path, *dirFlag)
		if err != nil {
			return nil // skip parse errors
		}
		if w != nil && w.InternalName != "" && w.Name != "" {
			weapons = append(weapons, *w)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Walk error: %v", err)
	}

	data, err := json.MarshalIndent(weapons, "", "  ")
	if err != nil {
		log.Fatalf("JSON marshal error: %v", err)
	}

	if err := os.WriteFile(*outFlag, data, 0644); err != nil {
		log.Fatalf("Write error: %v", err)
	}

	fmt.Printf("Extracted %d weapons to %s\n", len(weapons), *outFlag)
}
