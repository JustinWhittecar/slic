package bvcalc

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/JustinWhittecar/slic/internal/ingestion"
)

// Result holds the calculated BV breakdown
type Result struct {
	FinalBV      int
	DefensiveBR  float64
	OffensiveBR  float64
	ArmorBV      float64
	StructureBV  float64
	GyroBV       float64
	DefEquipBV   float64
	ExplosivePen float64
	DefFactor    float64
	WeaponBV     float64
	AmmoBV       float64
	SpeedFactor  float64
	HeatEff      int
	Errors       []string
}

// EquipmentDB provides weapon BV lookups
type EquipmentDB struct {
	// Map from internal_name -> equipment info
	ByInternalName map[string]*EquipInfo
	// Map from display name -> equipment info (may have duplicates IS/Clan; stores both)
	ByName map[string][]*EquipInfo
}

// EquipInfo holds equipment data from the DB
type EquipInfo struct {
	Name         string
	InternalName string
	Type         string
	BV           int
	Heat         int
	RackSize     int
	Tonnage      float64
}

// Calculate computes BV2 from an MTF file and equipment database
func Calculate(mtf *ingestion.MTFData, edb *EquipmentDB) Result {
	var r Result

	tonnage := mtf.Mass
	isClan := strings.Contains(strings.ToLower(mtf.TechBase), "clan")

	// ========== DEFENSIVE BATTLE RATING ==========

	// Total armor
	totalArmor := mtf.TotalArmor()
	armorMod := getArmorMod(mtf.ArmorType)
	r.ArmorBV = float64(totalArmor) * 2.5 * armorMod

	// Total IS
	totalIS := 0
	if is, ok := ISPointsByTonnage[tonnage]; ok {
		totalIS = is
	}
	structMod := getStructureMod(mtf.Structure)
	engMod := getEngineMod(mtf.EngineType, isClan)
	r.StructureBV = float64(totalIS) * 1.5 * structMod * engMod

	// Gyro
	gyroMod := GyroModifier(mtf.Gyro)
	r.GyroBV = float64(tonnage) * gyroMod

	// Parse location equipment for defensive gear, ammo, CASE, etc.
	type locEquip struct {
		name     string
		location string
	}

	var allEquip []locEquip
	caseLocations := map[string]bool{}
	ammoByLoc := map[string][]string{} // location -> list of ammo names
	gaussCritsByLoc := map[string]int{}

	for locName, items := range mtf.LocationEquipment {
		loc := normalizeLocation(locName)
		for _, item := range items {
			if item == "-Empty-" || item == "" {
				continue
			}
			allEquip = append(allEquip, locEquip{item, loc})

			if strings.Contains(item, "CASE") && !strings.Contains(item, "Ammo") {
				caseLocations[loc] = true
			}
			if isAmmoItem(item) {
				ammoByLoc[loc] = append(ammoByLoc[loc], item)
			}
			if isGaussWeapon(item) {
				gaussCritsByLoc[loc]++
			}
		}
	}

	// Defensive equipment BV — count per equipment item, not per crit slot
	// Track first occurrence of each defensive item per location
	defEquipBV := 0.0
	amsBV := 0
	amsAmmoBV := 0
	defEquipSeen := map[string]bool{} // "location:name" → seen
	for _, eq := range allEquip {
		bv := defensiveEquipBV(eq.name)
		if bv > 0 {
			key := eq.location + ":" + eq.name
			if !defEquipSeen[key] {
				defEquipSeen[key] = true
				defEquipBV += float64(bv)
				if isAMS(eq.name) {
					amsBV += bv
				}
			}
		}
		if IsAMSAmmo(eq.name) {
			amsAmmoBV += AmmoBV(eq.name)
		}
	}
	// Cap AMS ammo BV at AMS weapon BV
	if amsAmmoBV > amsBV {
		amsAmmoBV = amsBV
	}
	defEquipBV += float64(amsAmmoBV)
	r.DefEquipBV = defEquipBV

	// Explosive ammo penalties
	isXL := isXLEngine(mtf.EngineType)
	isLight := isLightEngine(mtf.EngineType)
	isXXL := isXXLEngine(mtf.EngineType)

	explosivePenalty := 0.0
	for loc, ammoList := range ammoByLoc {
		for _, ammoName := range ammoList {
			if !IsExplosiveAmmo(ammoName) {
				continue
			}
			if IsAMSAmmo(ammoName) {
				continue // AMS ammo handled separately
			}
			if penaltyApplies(loc, caseLocations, isClan, isXL, isLight, isXXL) {
				explosivePenalty += 15
			}
		}
	}

	// Gauss penalties (1 per crit, same location rules)
	for loc, crits := range gaussCritsByLoc {
		if penaltyApplies(loc, caseLocations, isClan, isXL, isLight, isXXL) {
			explosivePenalty += float64(crits)
		}
	}
	r.ExplosivePen = explosivePenalty

	defSubtotal := r.ArmorBV + r.StructureBV + r.GyroBV + r.DefEquipBV - explosivePenalty
	if defSubtotal < 1 {
		defSubtotal = 1
	}

	// Defensive factor from TMM
	runMP := mtf.WalkMP + int(math.Ceil(float64(mtf.WalkMP)*0.5))
	jumpMP := mtf.JumpMP

	// Check for MASC
	hasMASC := false
	hasTSM := false
	for _, items := range mtf.LocationEquipment {
		for _, item := range items {
			if strings.Contains(item, "MASC") {
				hasMASC = true
			}
			if strings.Contains(item, "TSM") || strings.Contains(item, "Triple Strength Myomer") {
				hasTSM = true
			}
		}
	}
	if strings.Contains(mtf.Myomer, "TSM") || strings.Contains(mtf.Myomer, "Triple Strength") {
		hasTSM = true
	}

	mascRunMP := runMP
	if hasMASC {
		mascRunMP = int(math.Ceil(float64(mtf.WalkMP) * 2.0))
	}
	if hasTSM {
		tsmRunMP := (mtf.WalkMP + 1) + int(math.Ceil(float64(mtf.WalkMP+1)*0.5))
		if tsmRunMP > mascRunMP {
			mascRunMP = tsmRunMP
		}
	}

	bestTMM := TMM(mascRunMP)
	jumpTMM := TMM(jumpMP)
	if jumpTMM > bestTMM {
		bestTMM = jumpTMM
	}

	r.DefFactor = DefensiveFactor(bestTMM)
	r.DefensiveBR = defSubtotal * r.DefFactor

	// ========== OFFENSIVE BATTLE RATING ==========

	// Use the Weapons summary block (not location blocks, which list per-crit-slot)
	type weaponInfo struct {
		name         string
		location     string
		isRear       bool
		bv           int
		heat         int
		internalName string
	}

	var weapons []weaponInfo
	hasTC := false

	// Build rear weapon set from location blocks: which weapons are rear-mounted
	// Count (R) crit slots per location per weapon name
	rearCritCounts := map[string]map[string]int{} // loc -> weaponName -> count of (R) crits
	frontCritCounts := map[string]map[string]int{} // loc -> weaponName -> count of non-(R) crits
	for locName, items := range mtf.LocationEquipment {
		loc := normalizeLocation(locName)
		for _, item := range items {
			if strings.Contains(item, "Targeting Computer") || strings.Contains(item, "ISTargeting") || strings.Contains(item, "CLTargeting") {
				hasTC = true
			}
			if strings.HasSuffix(item, "(R)") {
				clean := strings.TrimSuffix(item, " (R)")
				clean = strings.TrimSpace(clean)
				if rearCritCounts[loc] == nil {
					rearCritCounts[loc] = map[string]int{}
				}
				rearCritCounts[loc][clean]++
			} else if !isStructuralItem(item) && !isAmmoItem(item) && item != "-Empty-" {
				if frontCritCounts[loc] == nil {
					frontCritCounts[loc] = map[string]int{}
				}
				frontCritCounts[loc][item]++
			}
		}
	}

	// For each weapon in summary, determine rear by matching against location block
	// Track front weapons assigned per location
	frontAssigned := map[string]map[string]int{}

	for _, w := range mtf.Weapons {
		cleanName := strings.TrimSpace(w.Name)
		// Strip leading count (e.g., "2 ISERMediumLaser" → count=2, name="ISERMediumLaser")
		weaponCount := 1
		if len(cleanName) > 2 && cleanName[0] >= '0' && cleanName[0] <= '9' && cleanName[1] == ' ' {
			weaponCount = int(cleanName[0] - '0')
			cleanName = cleanName[2:]
		}

		loc := normalizeLocation(w.Location)

		// Check if this is a rear weapon:
		// If there are NO front crits of this weapon in this location, it must be rear
		// If there are both, assign front first
		isRear := false
		frontCount := 0
		if fc, ok := frontCritCounts[loc]; ok {
			frontCount = fc[cleanName]
		}
		rearCount := 0
		if rc, ok := rearCritCounts[loc]; ok {
			rearCount = rc[cleanName]
		}

		if rearCount > 0 && frontCount == 0 {
			// All instances of this weapon in this location are rear
			isRear = true
		} else if rearCount > 0 && frontCount > 0 {
			// Mix of front and rear; assign front first
			assigned := 0
			if frontAssigned[loc] != nil {
				assigned = frontAssigned[loc][cleanName]
			}
			if assigned >= frontCount {
				isRear = true // We've used up all front slots
			}
		}

		if !isRear {
			if frontAssigned[loc] == nil {
				frontAssigned[loc] = map[string]int{}
			}
			frontAssigned[loc][cleanName]++
		}

		if eq := lookupWeapon(cleanName, edb, isClan); eq != nil {
			for i := 0; i < weaponCount; i++ {
				weapons = append(weapons, weaponInfo{
					name:         eq.Name,
					location:     loc,
					isRear:       isRear,
					bv:           eq.BV,
					heat:         eq.Heat,
					internalName: eq.InternalName,
				})
			}
		}
	}

	// Check for Artemis IV/V on weapons
	hasArtemisIV := map[string]bool{}
	hasArtemisV := map[string]bool{}
	for _, items := range mtf.LocationEquipment {
		for _, item := range items {
			if strings.Contains(item, "Artemis IV") {
				// Artemis is paired with the weapon in the same location
				hasArtemisIV["_global"] = true
			}
			if strings.Contains(item, "Artemis V") {
				hasArtemisV["_global"] = true
			}
		}
	}

	// Calculate weapon modified BV
	type modWeapon struct {
		modBV float64
		heat  int
		name  string
	}

	totalFrontBV := 0.0
	totalRearBV := 0.0
	var frontWeapons []modWeapon
	var rearWeapons []modWeapon

	for _, w := range weapons {
		bv := float64(w.bv)
		if bv == 0 {
			continue
		}

		// Apply modifiers
		isLRM := strings.Contains(strings.ToLower(w.name), "lrm") || strings.Contains(strings.ToLower(w.name), "srm") || strings.Contains(strings.ToLower(w.name), "mml")
		isATM := strings.Contains(strings.ToLower(w.name), "atm")

		if (isLRM || isATM) && hasArtemisIV["_global"] {
			bv *= 1.2
		}
		if (isLRM || isATM) && hasArtemisV["_global"] {
			bv *= 1.3
		}

		isDirectFire := !isLRM && !isATM && !strings.Contains(strings.ToLower(w.name), "mrm") && !strings.Contains(strings.ToLower(w.name), "narc")
		if hasTC && isDirectFire {
			bv *= 1.25
		}

		// Adjust heat for BV calculation
		heat := w.heat
		if strings.Contains(strings.ToLower(w.name), "ultra") {
			heat *= 2
		}
		if strings.Contains(strings.ToLower(w.name), "rotary") {
			heat *= 6
		}
		if strings.Contains(strings.ToLower(w.name), "streak") {
			heat = int(math.Ceil(float64(heat) * 0.5))
		}
		if strings.Contains(strings.ToLower(w.name), "(os)") || strings.Contains(strings.ToLower(w.internalName), "os") {
			heat = int(math.Ceil(float64(heat) * 0.25))
		}

		mw := modWeapon{modBV: bv, heat: heat, name: w.name}
		if w.isRear {
			totalRearBV += bv
			rearWeapons = append(rearWeapons, mw)
		} else {
			totalFrontBV += bv
			frontWeapons = append(frontWeapons, mw)
		}
	}

	// If rear BV > front BV, swap: rear counts as full, front at half
	if totalRearBV > totalFrontBV {
		frontWeapons, rearWeapons = rearWeapons, frontWeapons
		totalFrontBV, totalRearBV = totalRearBV, totalFrontBV
	}

	// Apply 0.5 to rear weapons
	for i := range rearWeapons {
		rearWeapons[i].modBV *= 0.5
	}

	// Combine all weapons
	allWeapons := append(frontWeapons, rearWeapons...)

	// Sort by modified BV descending, then heat ascending
	sort.Slice(allWeapons, func(i, j int) bool {
		if allWeapons[i].modBV != allWeapons[j].modBV {
			return allWeapons[i].modBV > allWeapons[j].modBV
		}
		return allWeapons[i].heat < allWeapons[j].heat
	})

	// Heat efficiency
	hsCapacity := mtf.HeatSinkCount
	if isDoubleHS(mtf.HeatSinkType) {
		hsCapacity *= 2
		// Engine-integrated heat sinks are already counted in HeatSinkCount
	}

	hasStealth := strings.Contains(strings.ToLower(mtf.ArmorType), "stealth")
	movHeat := MovementHeat(runMP, jumpMP, hasStealth)
	heatEff := 6 + hsCapacity - movHeat
	r.HeatEff = heatEff

	// Apply heat efficiency to weapon BV
	heatUsed := 0
	weaponBV := 0.0
	exceeded := false
	for _, w := range allWeapons {
		if !exceeded {
			heatUsed += w.heat
			if heatUsed > heatEff {
				exceeded = true
				// This weapon still gets full BV
			}
			weaponBV += w.modBV
		} else {
			weaponBV += w.modBV * 0.5
		}
	}
	r.WeaponBV = weaponBV

	// Ammo BV (capped at weapon BV per type)
	weaponBVByType := map[string]float64{} // normalized weapon name -> total BV
	for _, w := range weapons {
		key := normalizeWeaponForAmmo(w.name)
		weaponBVByType[key] += float64(w.bv)
	}

	ammoBVByType := map[string]float64{}
	for _, ammoList := range ammoByLoc {
		for _, ammoName := range ammoList {
			if IsAMSAmmo(ammoName) {
				continue // handled in defensive
			}
			key := normalizeAmmoForWeapon(ammoName)
			bv := float64(AmmoBV(ammoName))
			ammoBVByType[key] += bv
		}
	}

	totalAmmoBV := 0.0
	for key, abv := range ammoBVByType {
		wbv := weaponBVByType[key]
		if wbv > 0 && abv > wbv {
			abv = wbv
		}
		totalAmmoBV += abv
	}
	r.AmmoBV = totalAmmoBV

	// Offensive equipment BV (non-defensive, non-weapon)
	offEquipBV := 0.0
	for _, eq := range allEquip {
		bv := offensiveEquipBV(eq.name)
		offEquipBV += float64(bv)
	}

	// Tonnage modifier
	tonnageBV := float64(tonnage)
	if hasTSM {
		tonnageBV *= 1.5
	}

	offSubtotal := weaponBV + totalAmmoBV + offEquipBV + tonnageBV

	// Speed factor
	sfRunMP := mascRunMP
	r.SpeedFactor = SpeedFactor(sfRunMP, jumpMP)
	r.OffensiveBR = offSubtotal * r.SpeedFactor

	// ========== FINAL ==========
	baseBV := r.DefensiveBR + r.OffensiveBR

	// Small cockpit
	cockpit := mtf.Cockpit
	if cockpit == "" {
		cockpit = "Standard Cockpit"
	}
	if strings.Contains(strings.ToLower(cockpit), "small") {
		baseBV *= 0.95
	}

	// Industrial mech without AFC
	if strings.Contains(strings.ToLower(mtf.Config), "industrial") {
		// Check for AFC
		hasAFC := false
		for _, items := range mtf.LocationEquipment {
			for _, item := range items {
				if strings.Contains(item, "Advanced Fire Control") {
					hasAFC = true
				}
			}
		}
		if !hasAFC {
			baseBV *= 0.9
		}
	}

	r.FinalBV = int(math.Round(baseBV))
	return r
}

func getArmorMod(armorType string) float64 {
	if m, ok := ArmorTypeModifier[armorType]; ok {
		return m
	}
	// Default to 1.0 for unknown types
	return 1.0
}

func getStructureMod(structType string) float64 {
	if m, ok := StructureTypeModifier[structType]; ok {
		return m
	}
	return 1.0
}

func getEngineMod(engineType string, isClan bool) float64 {
	et := strings.ToLower(engineType)
	switch {
	case strings.Contains(et, "xxl"):
		if isClan {
			return 0.75
		}
		return 0.5
	case strings.Contains(et, "xl"):
		if isClan || strings.Contains(et, "clan") {
			return 0.75
		}
		return 0.5
	case strings.Contains(et, "light"):
		return 0.75
	default:
		return 1.0
	}
}

func normalizeLocation(loc string) string {
	switch loc {
	case "Left Arm":
		return "LA"
	case "Right Arm":
		return "RA"
	case "Left Torso":
		return "LT"
	case "Right Torso":
		return "RT"
	case "Center Torso":
		return "CT"
	case "Head":
		return "HD"
	case "Left Leg":
		return "LL"
	case "Right Leg":
		return "RL"
	case "Front Left Leg":
		return "FLL"
	case "Front Right Leg":
		return "FRL"
	case "Rear Left Leg":
		return "RLL"
	case "Rear Right Leg":
		return "RRL"
	default:
		return loc
	}
}

func isAmmoItem(name string) bool {
	n := strings.ToLower(name)
	// Must start with common ammo prefixes, not just contain "ammo" (avoids lore text)
	return (strings.HasPrefix(n, "is ammo") || strings.HasPrefix(n, "clan ammo") ||
		strings.HasPrefix(n, "cl ammo") || strings.HasPrefix(n, "ammo") ||
		strings.HasPrefix(n, "isams ammo") || strings.HasPrefix(n, "clams ammo") ||
		strings.HasPrefix(n, "is ams ammo") || strings.HasPrefix(n, "cl ams ammo") ||
		strings.Contains(n, " ammo ") || strings.HasSuffix(n, " ammo")) &&
		len(n) < 100 // lore text is long
}

func isGaussWeapon(name string) bool {
	n := strings.ToLower(name)
	return (strings.Contains(n, "gauss") && !strings.Contains(n, "ammo"))
}

func isAMS(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "anti-missile") || strings.Contains(n, "antimissile") ||
		n == "isantimissilesystem" || n == "clantimissilesystem" ||
		strings.Contains(n, "ams") && !strings.Contains(n, "ammo")
}

func isStructuralItem(name string) bool {
	n := strings.ToLower(name)
	structural := []string{
		"shoulder", "upper arm", "lower arm", "hand actuator",
		"hip", "upper leg", "lower leg", "foot actuator",
		"life support", "sensors", "cockpit", "gyro",
		"fusion engine", "engine", "-empty-",
		"endo steel", "endo-steel", "ferro-fibrous",
		"heat sink", "double heat sink", "jump jet",
		"case", "is endo steel", "is endo-steel",
		"cl endo steel", "clan endo steel",
		"is ferro-fibrous", "clan ferro-fibrous",
		"is double heat sink", "clan double heat sink",
	}
	for _, s := range structural {
		if n == s || strings.HasPrefix(n, s) {
			return true
		}
	}
	// Heat sinks and jump jets are structural for weapon listing purposes
	// but we still need to not count them as weapons
	if strings.Contains(n, "heat sink") || strings.Contains(n, "jump jet") {
		return true
	}
	if strings.Contains(n, "endo") || strings.Contains(n, "ferro") {
		return true
	}
	if strings.Contains(n, "engine") || strings.Contains(n, "gyro") {
		return true
	}
	return false
}

func defensiveEquipBV(name string) int {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "anti-missile") || n == "isantimissilesystem" || n == "clantimissilesystem":
		return 32
	case strings.Contains(n, "laser anti-missile") || n == "cllaserantimissilesystem" || n == "islaserantimissilesystem":
		return 45
	case strings.Contains(n, "a-pod") || n == "isapod" || n == "clapod":
		return 1
	case strings.Contains(n, "b-pod") || n == "isbpod" || n == "clbpod":
		return 2
	case strings.Contains(n, "guardian ecm") || n == "isguardianecmsuite" || n == "isguardianecm":
		return 61
	case n == "clecmsuite" || n == "clan ecm suite" || strings.Contains(n, "ecm suite"):
		return 61
	case strings.Contains(n, "angel ecm"):
		return 100
	case strings.Contains(n, "beagle") || n == "isbeagleactiveprobe" || strings.Contains(n, "beagle active probe"):
		return 10
	case n == "clactiveprobe" || n == "clan active probe" || strings.Contains(n, "active probe"):
		return 10
	case n == "cllightactiveprobe" || strings.Contains(n, "light active probe"):
		return 7
	case strings.Contains(n, "bloodhound"):
		return 25
	}
	return 0
}

func offensiveEquipBV(name string) int {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "c3 master") || n == "isc3m":
		return 0 // C3 has no BV by itself
	case strings.Contains(n, "c3 slave") || n == "isc3s":
		return 0
	case strings.Contains(n, "tag") || n == "istag" || n == "cltag":
		return 0
	}
	return 0
}

func isXLEngine(engineType string) bool {
	et := strings.ToLower(engineType)
	return strings.Contains(et, "xl") && !strings.Contains(et, "xxl")
}

func isLightEngine(engineType string) bool {
	return strings.Contains(strings.ToLower(engineType), "light")
}

func isXXLEngine(engineType string) bool {
	return strings.Contains(strings.ToLower(engineType), "xxl")
}

func penaltyApplies(loc string, caseLocations map[string]bool, isClan, isXL, isLight, isXXL bool) bool {
	// CT, Head, Legs always take penalty
	critical := loc == "CT" || loc == "HD" || loc == "LL" || loc == "RL" ||
		loc == "FLL" || loc == "FRL" || loc == "RLL" || loc == "RRL"

	if isClan {
		// Clan mechs have CASE built-in everywhere except CT/HD/legs
		return critical
	}

	if isXL || isXXL {
		// XL/XXL: penalty in ANY location without CASE
		if critical {
			return true
		}
		return !caseLocations[loc]
	}

	// Standard/Light engine: penalty in CT/HD/legs, or non-CASE side torsos/arms
	if critical {
		return true
	}
	return !caseLocations[loc]
}

// weaponNameAliases maps display/internal names to equipment DB names
var weaponNameAliases = map[string]string{
	// iATM
	"iATM 3": "Improved ATM 3", "iATM 6": "Improved ATM 6",
	"iATM 9": "Improved ATM 9", "iATM 12": "Improved ATM 12",
	// Non-weapon equipment (BV=0 or defensive)
	"Light TAG": "", "TAG": "", "Clan TAG": "", "Light Active Probe": "",
	"Beagle Active Probe": "", "Guardian ECM Suite": "",
	"AMS": "", "B-Pod": "", "M-Pod": "", "A-Pod": "",
	// IS MegaMek internal names → DB display names
	"ISMediumLaser": "Medium Laser", "ISSmallLaser": "Small Laser", "ISLargeLaser": "Large Laser",
	"ISERMediumLaser": "ER Medium Laser", "ISERSmallLaser": "ER Small Laser", "ISERLargeLaser": "ER Large Laser",
	"ISMediumPulseLaser": "Medium Pulse Laser", "ISSmallPulseLaser": "Small Pulse Laser", "ISLargePulseLaser": "Large Pulse Laser",
	"ISPPC": "PPC", "ISERPPC": "ER PPC", "ISLightPPC": "Light PPC", "ISHeavyPPC": "Heavy PPC", "ISSNPPC": "Snub-Nose PPC",
	"ISFlamer": "Flamer",
	"ISLRM5": "LRM 5", "ISLRM10": "LRM 10", "ISLRM15": "LRM 15", "ISLRM20": "LRM 20",
	"ISSRM2": "SRM 2", "ISSRM4": "SRM 4", "ISSRM6": "SRM 6",
	"ISStreakSRM2": "Streak SRM 2", "ISStreakSRM4": "Streak SRM 4", "ISStreakSRM6": "Streak SRM 6",
	"ISGaussRifle": "Gauss Rifle", "ISLightGaussRifle": "Light Gauss Rifle", "ISHeavyGaussRifle": "Heavy Gauss Rifle",
	"ISLBXAC2": "LB 2-X AC", "ISLBXAC5": "LB 5-X AC", "ISLBXAC10": "LB 10-X AC", "ISLBXAC20": "LB 20-X AC",
	"ISUltraAC2": "Ultra AC/2", "ISUltraAC5": "Ultra AC/5", "ISUltraAC10": "Ultra AC/10", "ISUltraAC20": "Ultra AC/20",
	"ISRotaryAC2": "Rotary AC/2", "ISRotaryAC5": "Rotary AC/5",
	"ISAntiMissileSystem": "", "ISGuardianECM": "", "ISBeagleActiveProbe": "", "ISTAG": "",
	"ISC3SlaveUnit": "", "ISImprovedC3CPU": "",
	"ISMediumXPulseLaser": "Medium X-Pulse Laser", "ISSmallXPulseLaser": "Small X-Pulse Laser", "ISLargeXPulseLaser": "Large X-Pulse Laser",
	"ISRocketLauncher10": "Rocket Launcher 10", "ISRocketLauncher15": "Rocket Launcher 15", "ISRocketLauncher20": "Rocket Launcher 20",
	"ISMachine Gun": "Machine Gun", "ISHeavyMachineGun": "Heavy Machine Gun", "ISLightMachineGun": "Light Machine Gun",
	"ISMML3": "MML 3", "ISMML5": "MML 5", "ISMML7": "MML 7", "ISMML9": "MML 9",
	"ISAC2": "AC/2", "ISAC5": "AC/5", "ISAC10": "AC/10", "ISAC20": "AC/20",
	"ISPlasmaRifle": "Plasma Rifle",
	"ISERFlamer": "ER Flamer",
	// Clan MegaMek internal names → DB display names
	"CLERMediumLaser": "ER Medium Laser", "CLERSmallLaser": "ER Small Laser", "CLERLargeLaser": "ER Large Laser",
	"CLMediumPulseLaser": "Medium Pulse Laser", "CLSmallPulseLaser": "Small Pulse Laser", "CLLargePulseLaser": "Large Pulse Laser",
	"CLERMicroLaser": "ER Micro Laser", "CLMicroPulseLaser": "Micro Pulse Laser",
	"CLERPPC": "ER PPC",
	"CLFlamer": "Flamer", "CLERFlamer": "ER Flamer",
	"CLLRM5": "LRM 5", "CLLRM10": "LRM 10", "CLLRM15": "LRM 15", "CLLRM20": "LRM 20",
	"CLSRM2": "SRM 2", "CLSRM4": "SRM 4", "CLSRM6": "SRM 6",
	"CLStreakSRM2": "Streak SRM 2", "CLStreakSRM4": "Streak SRM 4", "CLStreakSRM6": "Streak SRM 6",
	"CLGaussRifle": "Gauss Rifle", "CLHeavyGaussRifle": "Heavy Gauss Rifle", "CLLightGaussRifle": "Light Gauss Rifle",
	"CLLBXAC2": "LB 2-X AC", "CLLBXAC5": "LB 5-X AC", "CLLBXAC10": "LB 10-X AC", "CLLBXAC20": "LB 20-X AC",
	"CLUltraAC2": "Ultra AC/2", "CLUltraAC5": "Ultra AC/5", "CLUltraAC10": "Ultra AC/10", "CLUltraAC20": "Ultra AC/20",
	"CLRotaryAC2": "Rotary AC/2", "CLRotaryAC5": "Rotary AC/5",
	"CLAntiMissileSystem": "", "CLAMS": "", "CLECMSuite": "", "CLActiveProbe": "", "CLLightActiveProbe": "",
	"CLTAG": "", "CLLightTAG": "",
	"CLHeavyMediumLaser": "Heavy Medium Laser", "CLHeavySmallLaser": "Heavy Small Laser", "CLHeavyLargeLaser": "Heavy Large Laser",
	"CLMachine Gun": "Machine Gun", "CLHeavyMachineGun": "Heavy Machine Gun", "CLLightMachineGun": "Light Machine Gun",
	"CLATM3": "ATM 3", "CLATM6": "ATM 6", "CLATM9": "ATM 9", "CLATM12": "ATM 12",
	"CLiATM3": "Improved ATM 3", "CLiATM6": "Improved ATM 6", "CLiATM9": "Improved ATM 9", "CLiATM12": "Improved ATM 12",
	"CLStreakLRM5": "Streak LRM 5", "CLStreakLRM10": "Streak LRM 10", "CLStreakLRM15": "Streak LRM 15", "CLStreakLRM20": "Streak LRM 20",
	"CLAPGaussRifle": "AP Gauss Rifle",
	"CLHAGRifle20": "HAG/20", "CLHAGRifle30": "HAG/30", "CLHAGRifle40": "HAG/40",
	"CLHAG20": "HAG/20", "CLHAG30": "HAG/30", "CLHAG40": "HAG/40",
	"CLPlasmaRifle": "Plasma Rifle", "CLPlasmaCannon": "Plasma Cannon",
	// Some common display name variants
	"Particle Cannon": "PPC",
	"LAC/5": "Light AC/5", "LAC/2": "Light AC/2",
	"C3 Master with TAG": "", // C3 + TAG combo, no offensive BV
	// Rocket launchers
	"Rocket Launcher 10": "Rocket Launcher 10", "Rocket Launcher 15": "Rocket Launcher 15",
	"Rocket Launcher 20": "Rocket Launcher 20",
}

func lookupWeapon(name string, edb *EquipmentDB, isClan ...bool) *EquipInfo {
	if edb == nil {
		return nil
	}

	clan := false
	if len(isClan) > 0 {
		clan = isClan[0]
	}

	// Try direct internal name lookup
	if eq, ok := edb.ByInternalName[name]; ok {
		return eq
	}

	// Try display name lookup
	if edb.ByName != nil {
		if eq := lookupByName(name, edb, clan); eq != nil {
			return eq
		}
	}

	// Try alias
	if alias, ok := weaponNameAliases[name]; ok {
		if alias == "" {
			return nil // known non-weapon
		}
		if eq := lookupByName(alias, edb, clan); eq != nil {
			return eq
		}
	}

	// Try stripping "(omnipod)" suffix
	clean := strings.TrimSuffix(name, " (omnipod)")
	if clean != name {
		return lookupWeapon(clean, edb, isClan...)
	}

	return nil
}

func lookupByName(name string, edb *EquipmentDB, clan bool) *EquipInfo {
	if eqs, ok := edb.ByName[name]; ok && len(eqs) > 0 {
		for _, eq := range eqs {
			isClanEquip := strings.HasPrefix(strings.ToLower(eq.InternalName), "cl")
			if clan == isClanEquip {
				return eq
			}
		}
		return eqs[0]
	}
	return nil
}

func isDoubleHS(hsType string) bool {
	n := strings.ToLower(hsType)
	return strings.Contains(n, "double")
}

func normalizeWeaponForAmmo(weaponName string) string {
	n := strings.ToLower(weaponName)
	n = strings.TrimPrefix(n, "cl ")
	n = strings.TrimPrefix(n, "is ")
	// Map weapon names to ammo type keys
	replacements := map[string]string{
		"autocannon/2":  "ac/2",
		"autocannon/5":  "ac/5",
		"autocannon/10": "ac/10",
		"autocannon/20": "ac/20",
	}
	for old, new_ := range replacements {
		n = strings.Replace(n, old, new_, 1)
	}
	return n
}

func normalizeAmmoForWeapon(ammoName string) string {
	n := strings.ToLower(ammoName)
	n = strings.TrimPrefix(n, "is ")
	n = strings.TrimPrefix(n, "clan ")
	n = strings.TrimPrefix(n, "cl ")
	n = strings.TrimPrefix(n, "ammo ")
	n = strings.TrimSuffix(n, " artemis-capable")
	n = strings.TrimSuffix(n, " artemis v-capable")
	n = strings.TrimSuffix(n, " narc-capable")
	// Normalize dashes
	n = strings.Replace(n, "-", " ", -1)
	return n
}

// Err helper
func (r *Result) AddError(format string, args ...interface{}) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}
