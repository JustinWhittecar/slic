package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/JustinWhittecar/slic/internal/db"
	"github.com/JustinWhittecar/slic/internal/ingestion"
)

// ─── Constants ───────────────────────────────────────────────────────────────

const (
	numSims       = 1000
	maxTurns      = 200
	startDistance  = 34
	gunnerySkill  = 4
	pilotingSkill = 5
	kFactor       = 3.5 // log-scale spread factor
)

// ─── Location constants ─────────────────────────────────────────────────────

const (
	LocHD  = 0
	LocCT  = 1
	LocLT  = 2
	LocRT  = 3
	LocLA  = 4
	LocRA  = 5
	LocLL  = 6
	LocRL  = 7
	NumLoc = 8
)

var locNames = [NumLoc]string{"HD", "CT", "LT", "RT", "LA", "RA", "LL", "RL"}

// Front hit location table (2d6, index 0-10 for rolls 2-12)
var frontHitTable = [11]int{
	LocCT, LocRA, LocRA, LocRL, LocRT, LocCT, LocLT, LocLL, LocLA, LocLA, LocHD,
}

// Rear hit table — same locations but CT/LT/RT use rear armor
var rearHitTable = [11]int{
	LocCT, LocRA, LocRA, LocRL, LocRT, LocCT, LocLT, LocLL, LocLA, LocLA, LocHD,
}

// ─── IS table by tonnage ────────────────────────────────────────────────────

// isTable[tons] = [HD, CT, LT, RT, LA, RA, LL, RL]
var isTable = map[int][NumLoc]int{
	20:  {3, 6, 5, 5, 3, 3, 4, 4},
	25:  {3, 8, 6, 6, 4, 4, 6, 6},
	30:  {3, 10, 7, 7, 5, 5, 7, 7},
	35:  {3, 11, 8, 8, 6, 6, 8, 8},
	40:  {3, 12, 10, 10, 6, 6, 10, 10},
	45:  {3, 14, 11, 11, 7, 7, 11, 11},
	50:  {3, 16, 12, 12, 8, 8, 12, 12},
	55:  {3, 18, 13, 13, 9, 9, 13, 13},
	60:  {3, 20, 14, 14, 10, 10, 14, 14},
	65:  {3, 21, 15, 15, 10, 10, 15, 15},
	70:  {3, 22, 15, 15, 11, 11, 15, 15},
	75:  {3, 23, 16, 16, 12, 12, 16, 16},
	80:  {3, 25, 17, 17, 13, 13, 17, 17},
	85:  {3, 27, 18, 18, 14, 14, 18, 18},
	90:  {3, 29, 19, 19, 15, 15, 19, 19},
	95:  {3, 30, 20, 20, 16, 16, 20, 20},
	100: {3, 31, 21, 21, 17, 17, 21, 21},
}

func getISForTonnage(tons int) [NumLoc]int {
	if v, ok := isTable[tons]; ok {
		return v
	}
	// Interpolate: find nearest lower
	bestTons := 20
	for t := range isTable {
		if t <= tons && t > bestTons {
			bestTons = t
		}
	}
	return isTable[bestTons]
}

// ─── Cluster hits table ─────────────────────────────────────────────────────

// clusterTable[roll-2][rackSizeIndex] = missiles hitting
// Rack sizes: 2,3,4,5,6,8,9,10,12,15,20,30,40
var clusterRackSizes = []int{2, 3, 4, 5, 6, 8, 9, 10, 12, 15, 20, 30, 40}
var clusterTable = [11][13]int{
	{1, 1, 1, 1, 2, 2, 2, 3, 4, 5, 6, 10, 12},   // roll 2
	{1, 1, 2, 2, 2, 2, 3, 3, 4, 5, 6, 10, 12},   // roll 3
	{1, 1, 2, 2, 3, 3, 4, 4, 5, 6, 9, 12, 18},   // roll 4
	{1, 2, 2, 3, 3, 4, 4, 6, 8, 9, 12, 18, 24},  // roll 5
	{1, 2, 3, 3, 4, 4, 5, 6, 8, 9, 12, 18, 24},  // roll 6
	{1, 2, 3, 3, 4, 4, 5, 6, 8, 9, 12, 18, 24},  // roll 7
	{2, 2, 3, 4, 4, 5, 5, 8, 10, 12, 16, 24, 32}, // roll 8
	{2, 3, 3, 4, 5, 5, 5, 8, 10, 12, 16, 24, 32}, // roll 9
	{2, 3, 4, 4, 5, 6, 7, 8, 10, 12, 16, 24, 32}, // roll 10
	{2, 3, 4, 5, 5, 8, 8, 10, 12, 15, 20, 30, 40}, // roll 11
	{2, 4, 4, 5, 6, 8, 9, 10, 12, 15, 20, 30, 40}, // roll 12
}

func clusterHits(rackSize int, rng *rand.Rand) int {
	return clusterHitsWithBonus(rackSize, 0, rng)
}

// clusterHitsWithBonus rolls cluster hits with an Artemis bonus (+2 for IV, +3 for V)
func clusterHitsWithBonus(rackSize int, bonus int, rng *rand.Rand) int {
	roll := roll2d6(rng) + bonus
	if roll > 12 {
		roll = 12
	}
	if roll < 2 {
		roll = 2
	}
	colIdx := 0
	for i, rs := range clusterRackSizes {
		if rs <= rackSize {
			colIdx = i
		}
	}
	return clusterTable[roll-2][colIdx]
}

// isDirectFire returns true for weapon categories eligible for Targeting Computer bonus
func isDirectFire(cat weaponCategory) bool {
	switch cat {
	case catNormal, catUltraAC, catRotaryAC, catLBX, catHAG:
		return true
	default:
		return false // missiles (LRM, SRM, MRM, Streak, ATM, MML) and artillery
	}
}

// amsIntercept reduces missile hits when defender has AMS. Call once per missile attack.
// Returns the reduced hit count.
func amsIntercept(hits int, defender *MechState, rng *rand.Rand) int {
	if !defender.HasAMS || defender.AMSUsedThisTurn {
		return hits
	}
	if !defender.IsLaserAMS {
		if defender.AMSAmmo <= 0 {
			return hits
		}
		defender.AMSAmmo--
	}
	defender.AMSUsedThisTurn = true
	reduction := roll1d6(rng)
	hits -= reduction
	if hits < 0 {
		hits = 0
	}
	return hits
}

// artemisBonus returns the cluster table bonus for a mech's fire control
func artemisBonus(m *MechState) int {
	if m.HasArtemisV {
		return 3
	}
	if m.HasArtemisIV {
		return 2
	}
	return 0
}

// ─── Ammo shots per ton ─────────────────────────────────────────────────────

var ammoPerTon = map[string]int{
	"AC/2": 45, "AC/5": 20, "AC/10": 10, "AC/20": 5,
	"LRM-5": 24, "LRM-10": 12, "LRM-15": 8, "LRM-20": 6,
	"SRM-2": 50, "SRM-4": 25, "SRM-6": 15,
	"Streak SRM-2": 50, "Streak SRM-4": 25, "Streak SRM-6": 15,
	"MRM-10": 24, "MRM-20": 12, "MRM-30": 8, "MRM-40": 6,
	"Ultra AC/2": 45, "Ultra AC/5": 20, "Ultra AC/10": 10, "Ultra AC/20": 5,
	"LB 2-X AC": 45, "LB 5-X AC": 20, "LB 10-X AC": 10, "LB 20-X AC": 5,
	"Rotary AC/2": 45, "Rotary AC/5": 20,
	"Gauss Rifle": 8, "Light Gauss": 16, "Heavy Gauss": 4,
	"Machine Gun": 200, "Light Machine Gun": 200, "Heavy Machine Gun": 100,
	"ATM 3": 20, "ATM 6": 10, "ATM 9": 7, "ATM 12": 5,
	"MML-3": 40, "MML-5": 24, "MML-7": 17, "MML-9": 13,
	"HAG/20": 6, "HAG/30": 4, "HAG/40": 3,
	"Arrow IV": 5,
	"Plasma Rifle": 10, "Plasma Cannon": 10,
	"Sniper": 10, "Thumper": 20, "Long Tom": 5,
	"Light AC/2": 45, "Light AC/5": 20,
	"Thunderbolt 5": 12, "Thunderbolt 10": 6, "Thunderbolt 15": 4, "Thunderbolt 20": 3,
}

// ─── Dice helpers ───────────────────────────────────────────────────────────

func roll1d6(rng *rand.Rand) int  { return rng.IntN(6) + 1 }
func roll2d6(rng *rand.Rand) int  { return roll1d6(rng) + roll1d6(rng) }

func tmmFromMP(mp int) int {
	switch {
	case mp <= 2:
		return 0
	case mp <= 4:
		return 1
	case mp <= 6:
		return 2
	case mp <= 9:
		return 3
	case mp <= 17:
		return 4
	case mp <= 24:
		return 5
	default:
		return 6
	}
}

// ─── 2d6 hit probability ───────────────────────────────────────────────────

var pHitTable = [13]float64{
	0, 0, 1.0, 35.0 / 36, 33.0 / 36, 30.0 / 36, 26.0 / 36,
	21.0 / 36, 15.0 / 36, 10.0 / 36, 6.0 / 36, 3.0 / 36, 1.0 / 36,
}

func hitProb(target int) float64 {
	if target <= 2 {
		return 1.0
	}
	if target >= 13 {
		return 0.0
	}
	return pHitTable[target]
}

// ─── Heat scale helpers ─────────────────────────────────────────────────────

// heatMPReduction returns walking MP penalty for a given heat level.
func heatMPReduction(heat int) int {
	switch {
	case heat >= 25:
		return 5
	case heat >= 20:
		return 4
	case heat >= 15:
		return 3
	case heat >= 10:
		return 2
	case heat >= 5:
		return 1
	default:
		return 0
	}
}

// heatToHitMod returns the to-hit penalty from heat.
func heatToHitMod(heat int) int {
	switch {
	case heat >= 24:
		return 4
	case heat >= 17:
		return 3
	case heat >= 13:
		return 2
	case heat >= 8:
		return 1
	default:
		return 0
	}
}

// prob2d6Fail returns P(roll < threshold) on 2d6.
// "avoid on X+" means you need X+ to avoid, so P(fail) = P(roll < X).
func prob2d6Fail(threshold int) float64 {
	if threshold <= 2 {
		return 0
	}
	if threshold > 12 {
		return 1.0
	}
	// P(roll >= threshold) = pHitTable[threshold], so P(fail) = 1 - pHitTable[threshold]
	return 1.0 - pHitTable[threshold]
}

// heatShutdownProb returns probability of shutdown at given heat level.
func heatShutdownProb(heat int) float64 {
	switch {
	case heat >= 30:
		return 1.0
	case heat >= 26:
		return prob2d6Fail(10)
	case heat >= 22:
		return prob2d6Fail(8)
	case heat >= 18:
		return prob2d6Fail(6)
	case heat >= 14:
		return prob2d6Fail(4)
	default:
		return 0
	}
}

// heatAmmoExpProb returns probability of ammo explosion at given heat level.
func heatAmmoExpProb(heat int) float64 {
	switch {
	case heat >= 28:
		return prob2d6Fail(8)
	case heat >= 23:
		return prob2d6Fail(6)
	case heat >= 19:
		return prob2d6Fail(4)
	default:
		return 0
	}
}

// heatCostEV estimates the expected value cost of being at a given heat level.
// avgTurnDmg = mech's average damage output per turn (cost of losing a turn to shutdown).
// ammoExpDmg = expected self-damage from an ammo explosion.
// walkMP = base walk MP before heat reduction.
// currentDmgCapability = current turn's damage output (for MP reduction cost).
func heatCostEV(heat int, avgTurnDmg float64, ammoExpDmg float64, walkMP int, currentDmgCapability float64) float64 {
	cost := 0.0

	// Shutdown cost: P(shutdown) × lost turn value
	// Shutdown means: can't fire next turn, auto-fail PSR (immobile), fall damage
	shutdownP := heatShutdownProb(heat)
	if shutdownP > 0 {
		cost += shutdownP * avgTurnDmg * 1.5 // 1.5x because you also take fall damage and lose positioning
	}

	// Ammo explosion cost
	ammoExpP := heatAmmoExpProb(heat)
	if ammoExpP > 0 {
		cost += ammoExpP * ammoExpDmg
	}

	// MP reduction cost: reduced TMM means taking more damage next turn
	// Estimate as fraction of damage capability lost
	mpLoss := heatMPReduction(heat)
	if mpLoss > 0 && walkMP > 0 {
		reducedWalk := walkMP - mpLoss
		if reducedWalk < 0 {
			reducedWalk = 0
		}
		origTMM := tmmFromMP(int(math.Ceil(float64(walkMP) * 1.5))) // run TMM
		newTMM := tmmFromMP(int(math.Ceil(float64(reducedWalk) * 1.5)))
		tmmLoss := origTMM - newTMM
		if tmmLoss > 0 {
			// Each TMM point is roughly 15% damage reduction
			cost += float64(tmmLoss) * 0.15 * currentDmgCapability
		}
	}

	return cost
}

// mechAmmoExplosionDamage estimates expected self-damage from a heat-triggered ammo explosion.
// Gauss rifles and one-shot weapons don't explode from heat.
func mechAmmoExplosionDamage(m *MechState) float64 {
	totalDmg := 0.0
	for key, shots := range m.Ammo {
		if shots <= 0 {
			continue
		}
		k := strings.ToLower(key)
		// Gauss rifles don't explode from heat
		if strings.Contains(k, "gauss") {
			continue
		}
		dmgPerShot := estimateAmmoDamage(key)
		locDmg := float64(shots * dmgPerShot)

		// Find which location this ammo is in for CASE check
		hasCASE := false
		for loc := 0; loc < NumLoc; loc++ {
			for _, slot := range m.Slots[loc] {
				if strings.Contains(strings.ToLower(slot), "ammo") && canonicalAmmoType(slot) == key {
					if m.HasCASE[loc] || m.HasCASEII[loc] {
						hasCASE = true
					}
					break
				}
			}
		}

		if hasCASE {
			// CASE limits damage significantly
			locDmg *= 0.1 // rough estimate: only lose that location's IS
		}
		totalDmg += locDmg
	}
	return totalDmg
}

// ─── Weapon classification ──────────────────────────────────────────────────

type weaponCategory int

const (
	catNormal    weaponCategory = iota
	catLRM                      // damage in 5-point groups, each rolls location
	catSRM                      // 2 damage per missile, each rolls location
	catMRM                      // 1 damage per missile, each rolls location
	catStreakSRM                // all-or-nothing, 2 dmg per missile
	catUltraAC                  // can fire twice
	catRotaryAC                 // can fire 1-6 times
	catLBX                      // cluster mode
	catHAG                      // cluster in 5-point groups
	catATM                      // 3 modes
	catMML                      // LRM or SRM mode
	catArrowIV                  // artillery
)

func categorizeWeapon(name string) weaponCategory {
	upper := strings.ToUpper(name)
	switch {
	case strings.Contains(upper, "STREAK SRM") || strings.Contains(upper, "STREAK LRM"):
		return catStreakSRM
	case strings.Contains(upper, "MML"):
		return catMML
	case strings.Contains(upper, "ATM") && !strings.Contains(upper, "ANTI"):
		return catATM
	case strings.Contains(upper, "ARROW IV"):
		return catArrowIV
	case strings.Contains(upper, "HAG"):
		return catHAG
	case strings.Contains(upper, "ROTARY AC"):
		return catRotaryAC
	case strings.Contains(upper, "ULTRA AC"):
		return catUltraAC
	case strings.Contains(upper, "LB") && strings.Contains(upper, "AC"):
		return catLBX
	case strings.Contains(upper, "MRM"):
		return catMRM
	case strings.Contains(upper, "SRM"):
		return catSRM
	case strings.Contains(upper, "LRM"):
		return catLRM
	default:
		return catNormal
	}
}

// ─── Mech state for simulation ──────────────────────────────────────────────

type SimWeapon struct {
	Name          string
	Category      weaponCategory
	Location      int // which body location
	Damage        int
	Heat          int
	MinRange      int
	ShortRange    int
	MedRange      int
	LongRange     int
	ToHitMod      int
	RackSize      int
	Type          string // energy, ballistic, missile, artillery
	AmmoKey       string // key into ammo pool, "" for energy
	Destroyed     bool
	Jammed        bool
}

type MechState struct {
	// Identity
	DebugName     string
	// Static
	Tonnage       int
	WalkMP        int
	RunMP         int
	JumpMP        int
	HeatSinkCount int
	Dissipation   int // per turn
	IsXL          bool
	IsClanXL      bool
	IsReinforced  bool
	IsComposite   bool
	TechBase      string // "IS" or "Clan"

	// Per-location
	Armor     [NumLoc]int
	RearArmor [3]int // CT, LT, RT rear (indices 0,1,2)
	IS        [NumLoc]int
	MaxIS     [NumLoc]int

	// Equipment per location (for crit)
	Slots [NumLoc][]string

	// CASE per torso location
	HasCASE   [NumLoc]bool
	HasCASEII [NumLoc]bool

	// Weapons
	Weapons []SimWeapon

	// Ammo pools: key -> shots remaining
	Ammo map[string]int

	// Dynamic state
	Heat           int
	EngineHits     int
	GyroHits       int
	SensorHits     int
	CockpitHit     bool
	ArmActuatorHit [NumLoc]int // extra to-hit for arm weapons
	LegActuatorHit int         // reduces walk MP

	// Tracking IS exposure
	ISExposed [NumLoc]bool // armor breached and IS took damage

	// Equipment: Targeting Computer, AMS, Artemis
	HasTargetingComputer bool
	HasAMS               bool
	AMSAmmo              int
	IsLaserAMS           bool
	HasArtemisIV         bool
	HasArtemisV          bool
	AMSUsedThisTurn      bool // reset each turn, only one AMS activation per turn

	// Heat state
	IsShutdown        bool // currently shut down (skip next turn)
	ProneFromShutdown bool // fell from shutdown, needs to stand

	// PSR / falling state
	Prone            bool
	PilotDamage      int  // 6 = dead
	PilotUnconscious bool
	HipHit           [NumLoc]bool // per leg location (LocLL, LocRL)
	LegFootHits      [NumLoc]int  // per leg: upper leg, lower leg, foot actuator hits
	NeedsPSRFromCrit bool         // set during crit resolution, checked after damage
}

func (m *MechState) effectiveWalkMP() int {
	mp := m.WalkMP - m.LegActuatorHit - heatMPReduction(m.Heat)
	if mp < 0 {
		mp = 0
	}
	return mp
}

func (m *MechState) effectiveRunMP() int {
	w := m.effectiveWalkMP()
	return int(math.Ceil(float64(w) * 1.5))
}

// ─── PSR / Falling mechanics ────────────────────────────────────────────────

// psrPreexistingMod returns cumulative modifier from preexisting damage
func (m *MechState) psrPreexistingMod() int {
	mod := m.GyroHits * 3
	for _, loc := range []int{LocLL, LocRL} {
		if m.IS[loc] <= 0 {
			mod += 5 // destroyed leg (ignore other crit mods)
		} else if m.HipHit[loc] {
			mod += 2 // hip replaces other leg actuator mods
		} else {
			mod += m.LegFootHits[loc] // +1 per destroyed actuator
		}
	}
	return mod
}

// rollPSR rolls a piloting skill roll. Returns true if passed.
// extraMod is the situational modifier (e.g. +1 for 20+ damage).
// Prone mechs ignore PSRs (except when standing - use rollPSRForStanding).
func (m *MechState) rollPSR(extraMod int, rng *rand.Rand) bool {
	if m.Prone {
		return true // prone mechs ignore PSRs
	}
	if m.PilotUnconscious || m.PilotDamage >= 6 {
		return false // auto-fail
	}
	if m.GyroHits >= 2 {
		return false // auto-fall
	}
	target := pilotingSkill + m.psrPreexistingMod() + extraMod
	return roll2d6(rng) >= target
}

// rollPSRForStanding rolls PSR to stand from prone. Returns true if passed.
func (m *MechState) rollPSRForStanding(rng *rand.Rand) bool {
	if m.PilotUnconscious || m.PilotDamage >= 6 {
		return false
	}
	if m.GyroHits >= 2 {
		return false
	}
	target := pilotingSkill + m.psrPreexistingMod()
	return roll2d6(rng) >= target
}

// consciousnessThreshold returns the 2d6 target to stay conscious after pilot damage
var consciousnessThresholds = [6]int{3, 5, 7, 10, 11, 99} // index = hits-1, 6th = dead

// applyFall handles a mech falling: prone, falling damage, pilot damage check
func (m *MechState) applyFall(rng *rand.Rand) {
	m.Prone = true

	// Falling damage: tonnage/10 (round up) for 0-level fall
	fallDmg := (m.Tonnage + 9) / 10

	// Facing after fall: 1d6: 1=Front, 2=Right, 3=Right, 4=Rear, 5=Left, 6=Left
	facing := roll1d6(rng)

	// Apply damage in 5-point groups to random locations
	for remaining := fallDmg; remaining > 0; {
		grp := 5
		if remaining < 5 {
			grp = remaining
		}
		loc := rollFallingHitLocation(facing, rng)
		isRear := facing == 4 && (loc == LocCT || loc == LocLT || loc == LocRT)
		m.applyDamage(loc, grp, isRear, rng)
		remaining -= grp
	}

	// Pilot damage check: PSR with preexisting mods, fail = 1 pilot damage
	target := pilotingSkill + m.psrPreexistingMod()
	if m.PilotUnconscious || roll2d6(rng) < target {
		m.PilotDamage++
		if m.PilotDamage >= 6 {
			return // dead
		}
		// Consciousness check
		threshold := consciousnessThresholds[m.PilotDamage-1]
		if roll2d6(rng) < threshold {
			m.PilotUnconscious = true
		}
	}
}

// rollFallingHitLocation rolls a hit location based on fall facing
func rollFallingHitLocation(facing int, rng *rand.Rand) int {
	// Use the standard hit table but with facing determining front/rear/side
	switch facing {
	case 1: // Front
		return rollHitLocation(false, rng)
	case 4: // Rear
		return rollHitLocation(true, rng)
	default: // 2,3=Right, 5,6=Left - use front table (simplified)
		return rollHitLocation(false, rng)
	}
}

func (m *MechState) isDestroyed() bool {
	if m.PilotDamage >= 6 {
		return true
	}
	if m.CockpitHit {
		return true
	}
	if m.EngineHits >= 3 {
		return true
	}
	if m.GyroHits >= 2 {
		return true
	}
	// CT destroyed
	if m.IS[LocCT] <= 0 {
		return true
	}
	// Head destroyed
	if m.IS[LocHD] <= 0 {
		return true
	}
	// XL engine checks
	if m.IsXL && !m.IsClanXL {
		// IS XL: either side torso destroyed = dead
		if m.IS[LocLT] <= 0 || m.IS[LocRT] <= 0 {
			return true
		}
	}
	if m.IsClanXL {
		// Clan XL: both side torsos destroyed = dead
		if m.IS[LocLT] <= 0 && m.IS[LocRT] <= 0 {
			return true
		}
	}
	// Leg destroyed
	if m.IS[LocLL] <= 0 || m.IS[LocRL] <= 0 {
		return true
	}
	// IS exposed in 3+ locations
	exposed := 0
	for i := 0; i < NumLoc; i++ {
		if m.ISExposed[i] {
			exposed++
		}
	}
	if exposed >= 3 {
		return true
	}
	return false
}

// applyDamage to a location, handling armor → IS → crits → transfer
func (m *MechState) applyDamage(loc int, dmg int, isRear bool, rng *rand.Rand) {
	if dmg <= 0 || loc < 0 || loc >= NumLoc {
		return
	}
	if m.IS[loc] <= 0 {
		// Location already destroyed, transfer
		m.transferDamage(loc, dmg, rng)
		return
	}

	remaining := dmg

	// Apply to armor first
	if isRear && (loc == LocCT || loc == LocLT || loc == LocRT) {
		rearIdx := loc - 1 // CT=1→0, LT=2→1, RT=3→2
		if m.RearArmor[rearIdx] > 0 {
			if m.RearArmor[rearIdx] >= remaining {
				m.RearArmor[rearIdx] -= remaining
				return
			}
			remaining -= m.RearArmor[rearIdx]
			m.RearArmor[rearIdx] = 0
		}
	} else {
		if m.Armor[loc] > 0 {
			if m.Armor[loc] >= remaining {
				m.Armor[loc] -= remaining
				return
			}
			remaining -= m.Armor[loc]
			m.Armor[loc] = 0
		}
	}

	// Damage reaches IS
	wasExposed := m.ISExposed[loc]
	isMult := 1.0
	if m.IsReinforced {
		isMult = 0.5
	} else if m.IsComposite {
		isMult = 2.0
	}

	effectiveDmg := int(math.Ceil(float64(remaining) * isMult))
	if m.IS[loc] > effectiveDmg {
		m.IS[loc] -= effectiveDmg
		m.ISExposed[loc] = true
		if !wasExposed {
			m.rollCrits(loc, rng)
		}
		return
	}

	// Location destroyed
	overflow := effectiveDmg - m.IS[loc]
	m.IS[loc] = 0
	m.ISExposed[loc] = true

	if !wasExposed {
		m.rollCrits(loc, rng)
	}

	// Destroy weapons in this location
	for i := range m.Weapons {
		if m.Weapons[i].Location == loc {
			m.Weapons[i].Destroyed = true
		}
	}

	// Transfer overflow
	if overflow > 0 && isMult != 1.0 {
		overflow = int(math.Ceil(float64(overflow) / isMult))
	}
	if overflow > 0 {
		m.transferDamage(loc, overflow, rng)
	}
}

func (m *MechState) transferDamage(fromLoc int, dmg int, rng *rand.Rand) {
	// Transfer: arms→torso, legs→torso, side torso→CT
	var toLoc int
	switch fromLoc {
	case LocLA:
		toLoc = LocLT
	case LocRA:
		toLoc = LocRT
	case LocLL:
		toLoc = LocLT
	case LocRL:
		toLoc = LocRT
	case LocLT, LocRT:
		toLoc = LocCT
	default:
		return // HD and CT don't transfer
	}
	m.applyDamage(toLoc, dmg, false, rng)
}

func (m *MechState) rollCrits(loc int, rng *rand.Rand) {
	critRoll := roll2d6(rng)
	if m.IsReinforced {
		critRoll--
	}

	numCrits := 0
	if loc == LocHD {
		if critRoll >= 12 {
			m.CockpitHit = true
			return
		}
		if critRoll >= 8 {
			numCrits = 1
		}
	} else {
		switch {
		case critRoll >= 12:
			numCrits = 3
		case critRoll >= 10:
			numCrits = 2
		case critRoll >= 8:
			numCrits = 1
		}
	}

	for i := 0; i < numCrits; i++ {
		m.applyCrit(loc, rng)
	}
}

func (m *MechState) applyCrit(loc int, rng *rand.Rand) {
	slots := m.Slots[loc]
	if len(slots) == 0 {
		return
	}

	// Filter non-empty slots
	var valid []int
	for i, s := range slots {
		if s != "-Empty-" && s != "" {
			valid = append(valid, i)
		}
	}
	if len(valid) == 0 {
		return
	}

	idx := valid[rng.IntN(len(valid))]
	slot := strings.ToLower(slots[idx])

	switch {
	case strings.Contains(slot, "engine"):
		m.EngineHits++
	case strings.Contains(slot, "gyro"):
		m.GyroHits++
		m.NeedsPSRFromCrit = true
	case strings.Contains(slot, "cockpit"):
		m.CockpitHit = true
	case strings.Contains(slot, "sensors"):
		m.SensorHits++
	case strings.Contains(slot, "heat sink"):
		if m.Dissipation > 0 {
			m.Dissipation--
			// Double heat sinks lose 2
			if strings.Contains(slot, "double") || strings.Contains(slot, "laser") {
				if m.Dissipation > 0 {
					m.Dissipation--
				}
			}
		}
	case strings.Contains(slot, "shoulder") || strings.Contains(slot, "upper arm") ||
		strings.Contains(slot, "lower arm") || strings.Contains(slot, "hand"):
		if loc == LocLA || loc == LocRA {
			m.ArmActuatorHit[loc]++
		}
	case strings.Contains(slot, "hip"):
		if !m.HipHit[loc] {
			m.HipHit[loc] = true
			m.LegActuatorHit += m.effectiveWalkMP() // hip = 0 MP
			m.NeedsPSRFromCrit = true
		}
	case strings.Contains(slot, "upper leg") || strings.Contains(slot, "lower leg") || strings.Contains(slot, "foot"):
		m.LegActuatorHit++
		m.LegFootHits[loc]++
		m.NeedsPSRFromCrit = true
	case strings.Contains(slot, "ammo"):
		m.ammoExplosion(loc, slots[idx], rng)
	default:
		// Weapon crit — find and destroy a matching weapon
		for i := range m.Weapons {
			if m.Weapons[i].Location == loc && !m.Weapons[i].Destroyed {
				if strings.Contains(strings.ToLower(m.Weapons[i].Name), strings.TrimSpace(slot)) ||
					strings.Contains(slot, strings.ToLower(m.Weapons[i].Name)) {
					m.Weapons[i].Destroyed = true
					break
				}
			}
		}
		// If no name match, destroy first undestroyed weapon in location
		for i := range m.Weapons {
			if m.Weapons[i].Location == loc && !m.Weapons[i].Destroyed {
				m.Weapons[i].Destroyed = true
				break
			}
		}
	}
}

func (m *MechState) ammoExplosion(loc int, slotName string, rng *rand.Rand) {
	if m.HasCASEII[loc] {
		return // no damage
	}

	// Find ammo key and remaining shots
	ammoKey := parseAmmoSlotKey(slotName)
	shots := m.Ammo[ammoKey]
	if shots <= 0 {
		return
	}
	m.Ammo[ammoKey] = 0

	// Damage per shot — estimate from ammo key
	dmgPerShot := estimateAmmoDamage(ammoKey)
	totalDmg := shots * dmgPerShot

	if m.HasCASE[loc] {
		// CASE: damage contained to this location only
		m.IS[loc] = 0
		m.ISExposed[loc] = true
		for i := range m.Weapons {
			if m.Weapons[i].Location == loc {
				m.Weapons[i].Destroyed = true
			}
		}
	} else {
		// Full explosion
		remaining := totalDmg - m.IS[loc]
		m.IS[loc] = 0
		m.ISExposed[loc] = true
		for i := range m.Weapons {
			if m.Weapons[i].Location == loc {
				m.Weapons[i].Destroyed = true
			}
		}
		if remaining > 0 {
			m.transferDamage(loc, remaining, rng)
		}
	}
}

// canonicalAmmoType maps any ammo slot name or weapon name to a canonical type string.
// This is the single source of truth for ammo matching.
func canonicalAmmoType(s string) string {
	s = strings.ToLower(s)
	// Strip prefixes, suffixes, and markers
	s = strings.ReplaceAll(s, "(omnipod)", "")
	s = strings.ReplaceAll(s, "(OMNIPOD)", "")
	s = strings.ReplaceAll(s, "- full", "")
	s = strings.ReplaceAll(s, "- half", "")
	s = strings.ReplaceAll(s, "cluster", "")
	s = strings.ReplaceAll(s, "ammo", "")
	s = strings.ReplaceAll(s, "clan ", "")
	s = strings.ReplaceAll(s, "is ", "")
	s = strings.ReplaceAll(s, "inner sphere ", "")
	s = strings.TrimSpace(s)

	// Normalize: remove spaces, dashes, slashes, underscores for matching
	norm := strings.NewReplacer(" ", "", "-", "", "/", "", "_", "", ".", "").Replace(s)

	// Map to canonical types
	switch {
	// Gauss variants
	case strings.Contains(norm, "heavygauss"):
		return "heavy gauss"
	case strings.Contains(norm, "lightgauss"):
		return "light gauss"
	case strings.Contains(norm, "apgauss"):
		return "ap gauss"
	case strings.Contains(norm, "gaussrifle") || norm == "gauss" || strings.Contains(norm, "clgauss") || strings.Contains(norm, "isgauss"):
		return "gauss rifle"
	// LBX
	case strings.Contains(norm, "lbx") || strings.Contains(norm, "lb2x") || strings.Contains(norm, "lb5x") || strings.Contains(norm, "lb10x") || strings.Contains(norm, "lb20x"):
		if strings.Contains(norm, "20") {
			return "lb 20-x ac"
		} else if strings.Contains(norm, "10") {
			return "lb 10-x ac"
		} else if strings.Contains(norm, "5") {
			return "lb 5-x ac"
		} else if strings.Contains(norm, "2") {
			return "lb 2-x ac"
		}
		return "lb 10-x ac"
	// Ultra AC
	case strings.Contains(norm, "ultra"):
		if strings.Contains(norm, "20") {
			return "ultra ac/20"
		} else if strings.Contains(norm, "10") {
			return "ultra ac/10"
		} else if strings.Contains(norm, "5") {
			return "ultra ac/5"
		} else if strings.Contains(norm, "2") {
			return "ultra ac/2"
		}
		return "ultra ac/5"
	// Rotary AC
	case strings.Contains(norm, "rotary"):
		if strings.Contains(norm, "5") {
			return "rotary ac/5"
		}
		return "rotary ac/2"
	// Light AC
	case strings.Contains(norm, "lightac") || strings.Contains(norm, "lac"):
		if strings.Contains(norm, "5") {
			return "light ac/5"
		}
		return "light ac/2"
	// Standard AC
	case strings.Contains(norm, "ac"):
		if strings.Contains(norm, "20") {
			return "ac/20"
		} else if strings.Contains(norm, "10") {
			return "ac/10"
		} else if strings.Contains(norm, "5") {
			return "ac/5"
		} else if strings.Contains(norm, "2") {
			return "ac/2"
		}
	// LRM
	case strings.Contains(norm, "lrm"):
		if strings.Contains(norm, "20") {
			return "lrm-20"
		} else if strings.Contains(norm, "15") {
			return "lrm-15"
		} else if strings.Contains(norm, "10") {
			return "lrm-10"
		} else if strings.Contains(norm, "5") {
			return "lrm-5"
		}
		return "lrm-10"
	// Streak SRM (must check before SRM)
	case strings.Contains(norm, "streak"):
		if strings.Contains(norm, "6") {
			return "streak srm-6"
		} else if strings.Contains(norm, "4") {
			return "streak srm-4"
		} else if strings.Contains(norm, "2") {
			return "streak srm-2"
		}
		return "streak srm-4"
	// SRM
	case strings.Contains(norm, "srm"):
		if strings.Contains(norm, "6") {
			return "srm-6"
		} else if strings.Contains(norm, "4") {
			return "srm-4"
		} else if strings.Contains(norm, "2") {
			return "srm-2"
		}
		return "srm-4"
	// MRM
	case strings.Contains(norm, "mrm"):
		if strings.Contains(norm, "40") {
			return "mrm-40"
		} else if strings.Contains(norm, "30") {
			return "mrm-30"
		} else if strings.Contains(norm, "20") {
			return "mrm-20"
		} else if strings.Contains(norm, "10") {
			return "mrm-10"
		}
		return "mrm-20"
	// MML
	case strings.Contains(norm, "mml"):
		if strings.Contains(norm, "9") {
			return "mml-9"
		} else if strings.Contains(norm, "7") {
			return "mml-7"
		} else if strings.Contains(norm, "5") {
			return "mml-5"
		} else if strings.Contains(norm, "3") {
			return "mml-3"
		}
		return "mml-5"
	// ATM
	case strings.Contains(norm, "atm"):
		if strings.Contains(norm, "12") {
			return "atm-12"
		} else if strings.Contains(norm, "9") {
			return "atm-9"
		} else if strings.Contains(norm, "6") {
			return "atm-6"
		} else if strings.Contains(norm, "3") {
			return "atm-3"
		}
		return "atm-6"
	// HAG
	case strings.Contains(norm, "hag"):
		if strings.Contains(norm, "40") {
			return "hag/40"
		} else if strings.Contains(norm, "30") {
			return "hag/30"
		}
		return "hag/20"
	// Machine Gun
	case strings.Contains(norm, "heavymachinegun") || strings.Contains(norm, "heavymg"):
		return "heavy machine gun"
	case strings.Contains(norm, "lightmachinegun") || strings.Contains(norm, "lightmg"):
		return "light machine gun"
	case strings.Contains(norm, "machinegun") || norm == "mg" || strings.Contains(norm, "clmg") || strings.Contains(norm, "ismg") || strings.Contains(norm, "ismachinegun"):
		return "machine gun"
	// Arrow IV
	case strings.Contains(norm, "arrow"):
		return "arrow iv"
	// Thunderbolt
	case strings.Contains(norm, "thunderbolt"):
		if strings.Contains(norm, "20") {
			return "thunderbolt-20"
		} else if strings.Contains(norm, "15") {
			return "thunderbolt-15"
		} else if strings.Contains(norm, "10") {
			return "thunderbolt-10"
		}
		return "thunderbolt-5"
	// Plasma
	case strings.Contains(norm, "plasma"):
		if strings.Contains(norm, "cannon") {
			return "plasma cannon"
		}
		return "plasma rifle"
	// AMS (not a weapon we track, but prevent false matches)
	case strings.Contains(norm, "ams") || strings.Contains(norm, "antimissile"):
		return "ams"
	// Sniper/Thumper/Long Tom
	case strings.Contains(norm, "longtom"):
		return "long tom"
	case strings.Contains(norm, "sniper"):
		return "sniper"
	case strings.Contains(norm, "thumper"):
		return "thumper"
	}

	// Fallback: return normalized string
	return strings.TrimSpace(s)
}

func parseAmmoSlotKey(slotName string) string {
	return canonicalAmmoType(slotName)
}

func estimateAmmoDamage(ammoKey string) int {
	k := strings.ToLower(ammoKey)
	switch {
	case strings.Contains(k, "ac/20"), strings.Contains(k, "ac/10"):
		if strings.Contains(k, "20") {
			return 20
		}
		return 10
	case strings.Contains(k, "ac/5"):
		return 5
	case strings.Contains(k, "ac/2"):
		return 2
	case strings.Contains(k, "gauss"):
		return 15
	case strings.Contains(k, "lrm"), strings.Contains(k, "mml"):
		return 1
	case strings.Contains(k, "srm"), strings.Contains(k, "streak"):
		return 2
	case strings.Contains(k, "mrm"):
		return 1
	case strings.Contains(k, "atm"):
		return 2
	default:
		return 5
	}
}

// ─── Build mech from MTF + DB data ─────────────────────────────────────────

type DBWeapon struct {
	Name      string
	Type      string
	Damage    int
	Heat      int
	MinRange  int
	Short     int
	Medium    int
	Long      int
	ToHitMod  int
	RackSize  int
	Location  string
	Quantity  int
}

type DBVariant struct {
	ID         int
	Name       string
	ModelCode  string
	TechBase   string
	Tonnage    int
	WalkMP     int
	RunMP      int
	JumpMP     int
	HSCount    int
	HSType     string
	EngineType string
	StructType string
	HasTC      bool
	Weapons    []DBWeapon
}

func locNameToIndex(name string) int {
	switch name {
	case "HD", "Head":
		return LocHD
	case "CT", "Center Torso":
		return LocCT
	case "LT", "Left Torso":
		return LocLT
	case "RT", "Right Torso":
		return LocRT
	case "LA", "Left Arm", "FLL", "Front Left Leg":
		return LocLA
	case "RA", "Right Arm", "FRL", "Front Right Leg":
		return LocRA
	case "LL", "Left Leg", "RLL", "Rear Left Leg":
		return LocLL
	case "RL", "Right Leg", "RRL", "Rear Right Leg":
		return LocRL
	default:
		return -1
	}
}

func buildMechState(v *DBVariant, mtf *ingestion.MTFData) *MechState {
	m := &MechState{
		DebugName:     v.Name + " " + v.ModelCode,
		Tonnage:       v.Tonnage,
		WalkMP:        v.WalkMP,
		RunMP:         v.RunMP,
		JumpMP:        v.JumpMP,
		HeatSinkCount: v.HSCount,
		TechBase:      v.TechBase,
		Ammo:          make(map[string]int),
	}

	// Dissipation
	hsLower := strings.ToLower(v.HSType)
	if strings.Contains(hsLower, "double") || strings.Contains(hsLower, "laser") {
		m.Dissipation = v.HSCount * 2
	} else {
		m.Dissipation = v.HSCount
	}

	// Engine type
	engLower := strings.ToLower(v.EngineType)
	if strings.Contains(engLower, "xl") {
		m.IsXL = true
		if strings.Contains(strings.ToLower(v.TechBase), "clan") {
			m.IsClanXL = true
		}
	}

	// Structure type
	if v.StructType != "" {
		stLower := strings.ToLower(v.StructType)
		if strings.Contains(stLower, "reinforced") {
			m.IsReinforced = true
		} else if strings.Contains(stLower, "composite") {
			m.IsComposite = true
		}
	}

	// IS values
	isVals := getISForTonnage(v.Tonnage)
	m.MaxIS = isVals
	m.IS = isVals

	// Armor from MTF
	if mtf != nil {
		for locStr, val := range mtf.ArmorValues {
			switch locStr {
			case "HD":
				m.Armor[LocHD] = val
			case "CT":
				m.Armor[LocCT] = val
			case "LT":
				m.Armor[LocLT] = val
			case "RT":
				m.Armor[LocRT] = val
			case "LA":
				m.Armor[LocLA] = val
			case "RA":
				m.Armor[LocRA] = val
			case "LL":
				m.Armor[LocLL] = val
			case "RL":
				m.Armor[LocRL] = val
			case "FLL":
				m.Armor[LocLA] = val
			case "FRL":
				m.Armor[LocRA] = val
			case "RLL":
				m.Armor[LocLL] = val
			case "RRL":
				m.Armor[LocRL] = val
			case "RTC":
				m.RearArmor[0] = val
			case "RTL":
				m.RearArmor[1] = val
			case "RTR":
				m.RearArmor[2] = val
			}
		}

		// Equipment slots
		for locStr, slots := range mtf.LocationEquipment {
			li := locNameToIndex(locStr)
			if li < 0 {
				continue
			}
			m.Slots[li] = make([]string, len(slots))
			copy(m.Slots[li], slots)

			// Parse ammo, CASE, and special equipment from slots
			for _, slot := range slots {
				sLower := strings.ToLower(slot)
				if strings.Contains(sLower, "case ii") {
					m.HasCASEII[li] = true
				} else if strings.Contains(sLower, "case") && !strings.Contains(sLower, "ammo") {
					m.HasCASE[li] = true
				}
				if strings.Contains(sLower, "ammo") {
					key := parseAmmoSlotKey(slot)
					if key == "ams" {
						m.AMSAmmo += 12 // 12 shots per ton standard AMS
					} else {
						shots := guessAmmoShots(slot)
						m.Ammo[key] += shots
					}
				}
				// Targeting Computer detection
				if strings.Contains(sLower, "targeting computer") || strings.Contains(sLower, "istargeting computer") || strings.Contains(sLower, "cltargeting computer") {
					m.HasTargetingComputer = true
				}
				// AMS detection
				if strings.Contains(sLower, "anti-missile") || (strings.Contains(sLower, "ams") && !strings.Contains(sLower, "ammo")) {
					m.HasAMS = true
					if strings.Contains(sLower, "laser") {
						m.IsLaserAMS = true
					}
				}
				// Artemis detection
				if strings.Contains(sLower, "artemis v") && !strings.Contains(sLower, "artemis iv") {
					m.HasArtemisV = true
				} else if strings.Contains(sLower, "artemis iv") {
					m.HasArtemisIV = true
				}
			}
		}
	}

	// Build weapons from DB
	for _, w := range v.Weapons {
		cat := categorizeWeapon(w.Name)
		li := locNameToIndex(w.Location)
		if li < 0 {
			li = LocCT // fallback
		}

		ammoKey := ""
		if w.Type == "ballistic" || w.Type == "missile" || w.Type == "artillery" {
			// Determine ammo key
			ammoKey = weaponToAmmoKey(w.Name)
		}
		if w.Type == "energy" {
			ammoKey = "" // energy = unlimited
		}

		for q := 0; q < w.Quantity; q++ {
			thm := w.ToHitMod
			// Targeting Computer: -1 to-hit for direct-fire weapons (not missiles)
			if m.HasTargetingComputer && isDirectFire(cat) {
				thm--
			}
			// LBX cluster mode: -1 to-hit bonus
			if cat == catLBX {
				thm--
			}
			sw := SimWeapon{
				Name:       w.Name,
				Category:   cat,
				Location:   li,
				Damage:     int(w.Damage),
				Heat:       w.Heat,
				MinRange:   w.MinRange,
				ShortRange: w.Short,
				MedRange:   w.Medium,
				LongRange:  w.Long,
				ToHitMod:   thm,
				RackSize:   w.RackSize,
				Type:       w.Type,
				AmmoKey:    ammoKey,
			}
			m.Weapons = append(m.Weapons, sw)
		}
	}

	return m
}

func weaponToAmmoKey(name string) string {
	return canonicalAmmoType(name)
}

// canonicalAmmoPerTon is built at init time with canonical keys
var canonicalAmmoPerTon map[string]int

func init() {
	canonicalAmmoPerTon = make(map[string]int, len(ammoPerTon))
	for k, v := range ammoPerTon {
		canonicalAmmoPerTon[canonicalAmmoType(k)] = v
	}
}

func guessAmmoShots(slotName string) int {
	key := canonicalAmmoType(slotName)
	if v, ok := canonicalAmmoPerTon[key]; ok {
		return v
	}
	// Partial match fallback
	for k, v := range canonicalAmmoPerTon {
		if strings.Contains(key, k) || strings.Contains(k, key) {
			return v
		}
	}
	return 10 // default fallback
}

// ─── HBK-4P hardcoded ──────────────────────────────────────────────────────

func buildHBK4P() *MechState {
	m := &MechState{
		DebugName:     "Hunchback HBK-4P",
		Tonnage:       50,
		WalkMP:        4,
		RunMP:         6,
		JumpMP:        0,
		HeatSinkCount: 23,
		Dissipation:   23,
		TechBase:      "Inner Sphere",
		Ammo:          make(map[string]int),
	}

	m.IS = [NumLoc]int{3, 16, 12, 12, 8, 8, 12, 12}
	m.MaxIS = m.IS
	m.Armor = [NumLoc]int{9, 26, 20, 20, 16, 16, 20, 20}
	m.RearArmor = [3]int{5, 4, 4} // CT, LT, RT rear

	// Crit slots
	m.Slots[LocHD] = []string{"Life Support", "Sensors", "Cockpit", "Small Laser", "Sensors", "Life Support"}
	m.Slots[LocCT] = []string{"Engine", "Engine", "Engine", "Gyro", "Gyro", "Gyro", "Gyro", "Engine", "Engine", "Engine", "Heat Sink", "Heat Sink"}
	m.Slots[LocLT] = []string{"Heat Sink", "Heat Sink", "Heat Sink", "Heat Sink", "Heat Sink", "Heat Sink", "Heat Sink"}
	m.Slots[LocRT] = []string{"Heat Sink", "Heat Sink", "Medium Laser", "Medium Laser", "Medium Laser", "Medium Laser", "Medium Laser", "Medium Laser"}
	m.Slots[LocLA] = []string{"Shoulder", "Upper Arm", "Lower Arm", "Hand", "Medium Laser"}
	m.Slots[LocRA] = []string{"Shoulder", "Upper Arm", "Lower Arm", "Hand", "Medium Laser"}
	m.Slots[LocLL] = []string{"Hip", "Upper Leg", "Lower Leg", "Foot", "Heat Sink", "Heat Sink"}
	m.Slots[LocRL] = []string{"Hip", "Upper Leg", "Lower Leg", "Foot", "Heat Sink", "Heat Sink"}

	// Weapons: 8x Medium Laser, 1x Small Laser
	for i := 0; i < 6; i++ {
		m.Weapons = append(m.Weapons, SimWeapon{
			Name: "Medium Laser", Category: catNormal, Location: LocRT,
			Damage: 5, Heat: 3, ShortRange: 3, MedRange: 6, LongRange: 9,
			Type: "energy",
		})
	}
	m.Weapons = append(m.Weapons, SimWeapon{
		Name: "Medium Laser", Category: catNormal, Location: LocLA,
		Damage: 5, Heat: 3, ShortRange: 3, MedRange: 6, LongRange: 9,
		Type: "energy",
	})
	m.Weapons = append(m.Weapons, SimWeapon{
		Name: "Medium Laser", Category: catNormal, Location: LocRA,
		Damage: 5, Heat: 3, ShortRange: 3, MedRange: 6, LongRange: 9,
		Type: "energy",
	})
	m.Weapons = append(m.Weapons, SimWeapon{
		Name: "Small Laser", Category: catNormal, Location: LocHD,
		Damage: 3, Heat: 1, ShortRange: 1, MedRange: 2, LongRange: 3,
		Type: "energy",
	})

	return m
}

// ─── Clone mech state for fresh sim ─────────────────────────────────────────

func cloneMech(src *MechState) *MechState {
	m := &MechState{}
	*m = *src
	m.Weapons = make([]SimWeapon, len(src.Weapons))
	copy(m.Weapons, src.Weapons)
	m.Ammo = make(map[string]int, len(src.Ammo))
	for k, v := range src.Ammo {
		m.Ammo[k] = v
	}
	for i := range m.Slots {
		if src.Slots[i] != nil {
			m.Slots[i] = make([]string, len(src.Slots[i]))
			copy(m.Slots[i], src.Slots[i])
		}
	}
	return m
}

// ─── Simulation core ────────────────────────────────────────────────────────

// simulateCombat: attacker fires at defender. Defender doesn't fire back.
var debugMech = "" // set to mech name to trace one sim, "" to disable
var debugOnce bool

// Both move smartly. Returns turns until defender is forced withdrawn.
func simulateCombat(attackerTemplate, defenderTemplate *MechState, rng *rand.Rand) int {
	debugSim := false
	if debugMech != "" && !debugOnce && strings.Contains(defenderTemplate.DebugName, debugMech) {
		debugSim = true
		debugOnce = true
		fmt.Printf("DEBUG DEFENSE: attacker=%s vs defender=%s\n", attackerTemplate.DebugName, defenderTemplate.DebugName)
		fmt.Printf("  Attacker: %d weapons, tonnage=%d, walk=%d, HS=%d/%d\n",
			len(attackerTemplate.Weapons), attackerTemplate.Tonnage, attackerTemplate.WalkMP,
			attackerTemplate.HeatSinkCount, attackerTemplate.Dissipation)
		fmt.Printf("  Defender: tonnage=%d, walk=%d, run=%d, jump=%d, armor total=%d\n",
			defenderTemplate.Tonnage, defenderTemplate.WalkMP, defenderTemplate.RunMP, defenderTemplate.JumpMP,
			func() int { t := 0; for i := 0; i < NumLoc; i++ { t += defenderTemplate.Armor[i] }; t += defenderTemplate.RearArmor[0]+defenderTemplate.RearArmor[1]+defenderTemplate.RearArmor[2]; return t }())
	}
	attacker := cloneMech(attackerTemplate)
	defender := cloneMech(defenderTemplate)

	distance := startDistance

	for turn := 1; turn <= maxTurns; turn++ {
		if attacker.isDestroyed() || defender.isDestroyed() {
			return turn - 1
		}

		// Handle shutdown state
		if attacker.IsShutdown {
			attacker.IsShutdown = false
			if !attacker.Prone {
				attacker.applyFall(rng)
			}
			if attacker.isDestroyed() {
				return turn
			}
			// Skip this turn
			attacker.Heat -= attacker.Dissipation
			if attacker.Heat < 0 {
				attacker.Heat = 0
			}
			defender.Heat -= defender.Dissipation
			if defender.Heat < 0 {
				defender.Heat = 0
			}
			continue
		}

		// Attacker: attempt to stand if prone
		if attacker.Prone {
			attacker.Heat += 1 // standing generates 1 heat
			if attacker.rollPSRForStanding(rng) {
				attacker.Prone = false
			} else {
				attacker.applyFall(rng) // failed stand = fall again
				if attacker.isDestroyed() {
					return turn
				}
				// Stays prone, skip movement — can still fire with penalty
			}
		}

		// Defender: attempt to stand if prone
		if defender.Prone {
			defender.Heat += 1
			if defender.rollPSRForStanding(rng) {
				defender.Prone = false
			} else {
				defender.applyFall(rng)
				if defender.isDestroyed() {
					return turn
				}
			}
		}

		// Movement phase — both mechs seek their own optimal engagement range.
		// This creates a tug-of-war: a sniper tries to hold range while a brawler closes.
		// Both choose simultaneously based on current distance; speed differential determines who wins.

		const maxDistance = 40 // map boundary (two standard mapsheets)

		type moveChoice struct {
			closing  int     // hexes toward opponent (positive=closing, negative=retreating)
			hexMoved int     // total hexes moved (for TMM)
			moveHeat int
			atkMod   int     // to-hit modifier if firing
			isJump   bool
			flanking float64 // P(rear shot), attacker only
		}

		// ── Defender movement: seek own optimal engagement range ──
		// Prone mechs cannot move (already attempted to stand above).
		defWalk := 0
		defRun := 0
		if !defender.Prone {
			defWalk = defender.effectiveWalkMP()
			defRun = defender.effectiveRunMP()
		}

		bestDefScore := -math.MaxFloat64
		var defChoice moveChoice

		evalDefOption := func(opt moveChoice) {
			newDist := distance - opt.closing
			if newDist < 1 {
				newDist = 1
			}
			if newDist > maxDistance {
				newDist = maxDistance
			}

			// Defender seeks range where IT does best damage (as if fighting)
			myDmg := calcExpectedDamage(defender, newDist, gunnerySkill+defender.SensorHits, 0, 0)

			// TMM tiebreaker (small weight so range preference dominates)
			myTMM := tmmFromMP(opt.hexMoved)
			if opt.isJump {
				myTMM++
			}

			score := myDmg + float64(myTMM)*0.5

			if score > bestDefScore || (score == bestDefScore && opt.hexMoved < defChoice.hexMoved) {
				bestDefScore = score
				defChoice = opt
			}
		}

		// Stand
		evalDefOption(moveChoice{closing: 0, hexMoved: 0, moveHeat: 0, atkMod: 0})
		// Walk toward/away/lateral
		for d := 1; d <= defWalk; d++ {
			evalDefOption(moveChoice{closing: d, hexMoved: d, moveHeat: 1, atkMod: 1})
			evalDefOption(moveChoice{closing: -d, hexMoved: d, moveHeat: 1, atkMod: 1})
		}
		if defWalk > 0 {
			evalDefOption(moveChoice{closing: 0, hexMoved: defWalk, moveHeat: 1, atkMod: 1}) // lateral
		}
		// Run toward/away/lateral
		for d := defWalk + 1; d <= defRun; d++ {
			evalDefOption(moveChoice{closing: d, hexMoved: d, moveHeat: 2, atkMod: 2})
			evalDefOption(moveChoice{closing: -d, hexMoved: d, moveHeat: 2, atkMod: 2})
		}
		if defRun > defWalk {
			evalDefOption(moveChoice{closing: 0, hexMoved: defRun, moveHeat: 2, atkMod: 2}) // lateral
		}
		// Jump toward/away/lateral
		for j := 1; j <= defender.JumpMP; j++ {
			evalDefOption(moveChoice{closing: j, hexMoved: j, moveHeat: j, atkMod: 3, isJump: true})
			evalDefOption(moveChoice{closing: -j, hexMoved: j, moveHeat: j, atkMod: 3, isJump: true})
			evalDefOption(moveChoice{closing: 0, hexMoved: j, moveHeat: j, atkMod: 3, isJump: true})
		}

		defTMM := tmmFromMP(defChoice.hexMoved)
		if defChoice.isJump {
			defTMM++
		}
		defMoveHeat := defChoice.moveHeat

		// ── Attacker movement: maximize damage output with asymmetric modifier advantage ──
		var bestAtk moveChoice
		bestAtkScore := -math.MaxFloat64

		evalAtkOption := func(opt moveChoice) {
			newDist := distance - opt.closing
			if newDist < 1 {
				newDist = 1
			}
			if newDist > maxDistance {
				newDist = maxDistance
			}

			// Steady-state damage at destination (drives range selection)
			steadyDmg := calcExpectedDamage(attacker, newDist, gunnerySkill+attacker.SensorHits, defTMM, 0)

			// Asymmetric modifier advantage (drives movement type selection)
			myTMM := tmmFromMP(opt.hexMoved)
			if opt.isJump {
				myTMM++
			}

			myDmgWithMod := calcExpectedDamage(attacker, newDist, gunnerySkill+attacker.SensorHits+opt.atkMod, defTMM, 0)
			oppDmgWithTMM := calcExpectedDamage(defender, newDist, gunnerySkill+defender.SensorHits, myTMM, 0)
			oppDmgNoTMM := calcExpectedDamage(defender, newDist, gunnerySkill+defender.SensorHits, 0, 0)

			tmmBenefit := oppDmgNoTMM - oppDmgWithTMM
			atkCost := steadyDmg - myDmgWithMod
			asymBonus := tmmBenefit - atkCost

			var score float64
			if steadyDmg > 0 {
				// In range: balance damage output vs incoming damage
				score = steadyDmg + asymBonus
			} else {
				// Out of range: close to where we can shoot (prefer shorter distance)
				score = -float64(newDist)
			}

			newDistAtk := distance - opt.closing
			bestDistAtk := distance - bestAtk.closing
			better := score > bestAtkScore || (score == bestAtkScore && newDistAtk < bestDistAtk)
			if better {
				bestAtkScore = score
				bestAtk = opt
			}
		}

		atkWalk := 0
		atkRun := 0
		if !attacker.Prone {
			atkWalk = attacker.effectiveWalkMP()
			atkRun = attacker.effectiveRunMP()
		}

		// Stand
		evalAtkOption(moveChoice{closing: 0, hexMoved: 0, moveHeat: 0, atkMod: 0})
		// Walk toward/away
		for d := 1; d <= atkWalk; d++ {
			excess := float64(d)
			newDist := distance - d
			if newDist < 1 {
				excess = float64(d - abs(distance-1))
				newDist = 1
			}
			flank := 0.0
			if newDist > 0 {
				flank = 0.5 * min64(1.0, excess/float64(2*newDist))
			}
			evalAtkOption(moveChoice{closing: d, hexMoved: d, moveHeat: 1, atkMod: 1, flanking: flank})
			evalAtkOption(moveChoice{closing: -d, hexMoved: d, moveHeat: 1, atkMod: 1})
		}
		// Walk lateral
		if atkWalk > 0 {
			flank := 0.5 * min64(1.0, float64(atkWalk)/float64(2*max(distance, 1)))
			evalAtkOption(moveChoice{closing: 0, hexMoved: atkWalk, moveHeat: 1, atkMod: 1, flanking: flank})
		}
		// Run toward/away
		for d := atkWalk + 1; d <= atkRun; d++ {
			excess := float64(d)
			newDist := distance - d
			if newDist < 1 {
				excess = float64(d - abs(distance-1))
				newDist = 1
			}
			flank := 0.0
			if newDist > 0 {
				flank = 0.5 * min64(1.0, excess/float64(2*newDist))
			}
			evalAtkOption(moveChoice{closing: d, hexMoved: d, moveHeat: 2, atkMod: 2, flanking: flank})
			evalAtkOption(moveChoice{closing: -d, hexMoved: d, moveHeat: 2, atkMod: 2})
		}
		// Run lateral
		if atkRun > atkWalk {
			flank := 0.5 * min64(1.0, float64(atkRun)/float64(2*max(distance, 1)))
			evalAtkOption(moveChoice{closing: 0, hexMoved: atkRun, moveHeat: 2, atkMod: 2, flanking: flank})
		}
		// Jump toward/away/lateral
		for j := 1; j <= attacker.JumpMP; j++ {
			excess := float64(j)
			newDist := distance - j
			if newDist < 1 {
				excess = float64(j - abs(distance-1))
				newDist = 1
			}
			flank := 0.0
			if newDist > 0 {
				flank = 0.5 * min64(1.0, excess/float64(2*newDist))
			}
			evalAtkOption(moveChoice{closing: j, hexMoved: j, moveHeat: j, atkMod: 3, isJump: true, flanking: flank})
			evalAtkOption(moveChoice{closing: -j, hexMoved: j, moveHeat: j, atkMod: 3, isJump: true})
			// Lateral
			latFlank := 0.5 * min64(1.0, float64(j)/float64(2*max(distance, 1)))
			evalAtkOption(moveChoice{closing: 0, hexMoved: j, moveHeat: j, atkMod: 3, isJump: true, flanking: latFlank})
		}

		// Apply both movements simultaneously
		distance = distance - bestAtk.closing - defChoice.closing
		if distance < 1 {
			distance = 1
		}
		if distance > maxDistance {
			distance = maxDistance
		}

		// Reset per-turn state
		defender.AMSUsedThisTurn = false

		// Movement heat (dissipation happens AFTER weapons fire, at end of turn)
		attacker.Heat += bestAtk.moveHeat
		defender.Heat += defMoveHeat

		// Weapon fire — EV-based heat selection
		heatThisMod := heatToHitMod(attacker.Heat)
		baseTarget := gunnerySkill + bestAtk.atkMod + attacker.SensorHits + defTMM + heatThisMod

		// Prone modifiers
		if attacker.Prone {
			baseTarget += 1 // prone attacker +1
		}
		if defender.Prone {
			if distance <= 1 {
				baseTarget += 1 // harder to hit prone at close range
			} else {
				baseTarget -= 2 // easier to hit prone at distance (partial cover inverted)
			}
		}

		// Select weapons using EV-based approach: fire weapons where marginal damage > marginal heat cost
		type weaponFire struct {
			idx    int
			expDmg float64
			heat   int
		}
		var candidates []weaponFire

		for i := range attacker.Weapons {
			w := &attacker.Weapons[i]
			if w.Destroyed || w.Jammed {
				continue
			}
			if w.AmmoKey != "" && attacker.Ammo[w.AmmoKey] <= 0 {
				continue
			}

			ed := weaponExpectedDamage(w, distance, baseTarget)
			if ed <= 0 {
				continue
			}
			candidates = append(candidates, weaponFire{i, ed, w.Heat})
		}

		// Sort by damage/heat ratio (zero-heat first)
		sort.Slice(candidates, func(a, b int) bool {
			ra := candidates[a].expDmg / max64(float64(candidates[a].heat), 0.1)
			rb := candidates[b].expDmg / max64(float64(candidates[b].heat), 0.1)
			return ra > rb
		})

		// Estimate average turn damage for heat cost calculation
		avgTurnDmg := 0.0
		for _, c := range candidates {
			avgTurnDmg += c.expDmg
		}
		ammoExpDmg := mechAmmoExplosionDamage(attacker)

		// Greedy EV-based selection: add weapons while marginal EV is positive
		var firingWeapons []int
		weaponHeatTotal := 0

		for _, c := range candidates {
			if c.heat == 0 {
				// Zero-heat weapons always fire
				firingWeapons = append(firingWeapons, c.idx)
				continue
			}

			// Projected heat after adding this weapon
			newWeaponHeat := weaponHeatTotal + c.heat
			projectedHeat := attacker.Heat + newWeaponHeat - attacker.Dissipation
			if projectedHeat < 0 {
				projectedHeat = 0
			}

			// Current heat cost (without this weapon)
			oldProjectedHeat := attacker.Heat + weaponHeatTotal - attacker.Dissipation
			if oldProjectedHeat < 0 {
				oldProjectedHeat = 0
			}

			// Marginal heat cost
			oldCost := heatCostEV(oldProjectedHeat, avgTurnDmg, ammoExpDmg, attacker.WalkMP, avgTurnDmg)
			newCost := heatCostEV(projectedHeat, avgTurnDmg, ammoExpDmg, attacker.WalkMP, avgTurnDmg)
			marginalCost := newCost - oldCost

			// Check if to-hit modifier changes (affects ALL weapons)
			oldToHitMod := heatToHitMod(oldProjectedHeat)
			newToHitMod := heatToHitMod(projectedHeat)
			toHitPenaltyCost := 0.0
			if newToHitMod > oldToHitMod {
				// Recalculate total damage of already-selected weapons with worse modifier
				for _, fi := range firingWeapons {
					fw := &attacker.Weapons[fi]
					oldDmg := weaponExpectedDamage(fw, distance, baseTarget+oldToHitMod-heatThisMod)
					newDmg := weaponExpectedDamage(fw, distance, baseTarget+newToHitMod-heatThisMod)
					toHitPenaltyCost += oldDmg - newDmg
				}
			}

			// Also account for this weapon's own damage at new to-hit level
			actualDmg := weaponExpectedDamage(&attacker.Weapons[c.idx], distance, baseTarget+newToHitMod-heatThisMod)

			marginalEV := actualDmg - marginalCost - toHitPenaltyCost

			if marginalEV > 0 {
				firingWeapons = append(firingWeapons, c.idx)
				weaponHeatTotal += c.heat
			}
		}

		// Apply total weapon heat to attacker
		attacker.Heat += weaponHeatTotal

		if debugSim {
			defHP := 0
			for i := 0; i < NumLoc; i++ {
				defHP += defender.Armor[i] + defender.IS[i]
			}
			defHP += defender.RearArmor[0] + defender.RearArmor[1] + defender.RearArmor[2]
			fmt.Printf("T%d: dist=%d atkMod=%d defTMM=%d base=%d heat=%d diss=%d firing=%d/%d defHP=%d moveHeat=%d defClose=%d\n",
				turn, distance, bestAtk.atkMod, defTMM, baseTarget, attacker.Heat, attacker.Dissipation, len(firingWeapons), len(candidates), defHP, bestAtk.moveHeat, defChoice.closing)
			projH := attacker.Heat - attacker.Dissipation; if projH < 0 { projH = 0 }
			fmt.Printf("  projectedHeat=%d heatToHitMod=%d shutdownP=%.2f\n", projH, heatToHitMod(projH), heatShutdownProb(projH))
			for _, wi := range firingWeapons {
				w := &attacker.Weapons[wi]
				rm := rangeModifier(w, distance)
				fmt.Printf("  Fire: %s dmg=%d heat=%d range=%d/%d/%d rm=%d target=%d\n",
					w.Name, w.Damage, w.Heat, w.ShortRange, w.MedRange, w.LongRange, rm, baseTarget+w.ToHitMod+rm)
			}
			if turn >= 50 {
				debugSim = false
			}
		}

		// Resolve weapon fire
		for _, wi := range firingWeapons {
			w := &attacker.Weapons[wi]

			// Consume ammo
			if w.AmmoKey != "" {
				if attacker.Ammo[w.AmmoKey] <= 0 {
					continue
				}
				attacker.Ammo[w.AmmoKey]--
			}

			target := baseTarget + w.ToHitMod + attacker.ArmActuatorHit[w.Location]

			// Range mod
			rangeMod := rangeModifier(w, distance)
			if rangeMod < 0 {
				continue // out of range
			}
			target += rangeMod

			// Min range penalty
			if w.MinRange > 0 && distance <= w.MinRange {
				target += w.MinRange - distance + 1
			}

			// Determine if rear shot
			isRear := false
			if bestAtk.flanking > 0 && rng.Float64() < bestAtk.flanking {
				isRear = true
			}

			resolveWeaponFire(w, target, isRear, attacker, defender, rng)

			if defender.isDestroyed() {
				return turn
			}
		}

		// PSR checks for defender after weapon fire
		{
			// Check for 20+ damage PSR
			// We approximate: if total expected damage dealt >= 20, trigger PSR with +1
			// For accuracy, we'd track actual damage dealt, but we can use a simpler approach:
			// Count total damage of weapons that were fired and hit
			// Actually, let's track it properly by checking defender state change
			// Simpler: calculate total potential damage from firing weapons
			totalFiredDmg := 0
			for _, wi := range firingWeapons {
				w := &attacker.Weapons[wi]
				totalFiredDmg += w.Damage
				if w.Category == catUltraAC {
					totalFiredDmg += w.Damage
				}
				if w.Category == catRotaryAC {
					totalFiredDmg += w.Damage * 5
				}
			}
			// Use expected hit rate to estimate actual damage dealt
			// For simplicity, if total potential >= 30 (accounting for misses), trigger 20+ PSR
			if totalFiredDmg >= 20 && !defender.Prone {
				if !defender.rollPSR(1, rng) {
					defender.applyFall(rng)
					if defender.isDestroyed() {
						return turn
					}
				}
			}

			// PSR from critical hits (gyro, actuator)
			if defender.NeedsPSRFromCrit && !defender.isDestroyed() {
				defender.NeedsPSRFromCrit = false
				if !defender.rollPSR(0, rng) {
					defender.applyFall(rng)
					if defender.isDestroyed() {
						return turn
					}
				}
			}
		}

		// Physical attacks at range 1
		if distance == 1 {
			resolvePhysical(attacker, defender, bestAtk.atkMod, rng)
			if defender.isDestroyed() {
				return turn
			}
			// PSR for being kicked
			if !defender.Prone {
				if !defender.rollPSR(0, rng) {
					defender.applyFall(rng)
					if defender.isDestroyed() {
						return turn
					}
				}
			}
		}

		// End of turn: heat dissipation
		attacker.Heat -= attacker.Dissipation
		if attacker.Heat < 0 {
			attacker.Heat = 0
		}
		defender.Heat -= defender.Dissipation
		if defender.Heat < 0 {
			defender.Heat = 0
		}

		// Heat phase: shutdown check for attacker
		shutdownP := heatShutdownProb(attacker.Heat)
		if shutdownP >= 1.0 || (shutdownP > 0 && rng.Float64() < shutdownP) {
			attacker.IsShutdown = true
		}

		// Heat phase: ammo explosion check for attacker
		ammoExpP := heatAmmoExpProb(attacker.Heat)
		if ammoExpP > 0 && rng.Float64() < ammoExpP {
			// Find an ammo bin to explode (random non-gauss ammo)
			type ammoBin struct {
				key string
				loc int
			}
			var bins []ammoBin
			for loc := 0; loc < NumLoc; loc++ {
				for _, slot := range attacker.Slots[loc] {
					sLower := strings.ToLower(slot)
					if strings.Contains(sLower, "ammo") && !strings.Contains(sLower, "gauss") {
						key := parseAmmoSlotKey(slot)
						if attacker.Ammo[key] > 0 {
							bins = append(bins, ammoBin{key, loc})
						}
					}
				}
			}
			if len(bins) > 0 {
				bin := bins[rng.IntN(len(bins))]
				attacker.ammoExplosion(bin.loc, bin.key, rng)
				if attacker.isDestroyed() {
					return turn
				}
			}
		}
	}

	return maxTurns
}

func rangeModifier(w *SimWeapon, dist int) int {
	if w.Category == catArrowIV {
		if dist >= w.MinRange && dist <= w.LongRange {
			return 0 // Arrow IV has flat modifier via ToHitMod
		}
		return -1
	}
	if dist > w.LongRange || w.LongRange == 0 {
		return -1
	}
	if dist <= w.ShortRange {
		return 0
	}
	if dist <= w.MedRange {
		return 2
	}
	return 4
}

func resolveWeaponFire(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) {
	switch w.Category {
	case catArrowIV:
		resolveArrowIV(w, target, defender, rng)
	case catStreakSRM:
		resolveStreakSRM(w, target, isRear, attacker, defender, rng)
	case catUltraAC:
		resolveUltraAC(w, target, isRear, defender, rng)
	case catRotaryAC:
		resolveRotaryAC(w, target, isRear, defender, rng)
	case catLBX:
		resolveLBX(w, target, isRear, defender, rng)
	case catLRM:
		resolveLRM(w, target, isRear, attacker, defender, rng)
	case catSRM:
		resolveSRM(w, target, isRear, attacker, defender, rng)
	case catMRM:
		resolveMRM(w, target, isRear, attacker, defender, rng)
	case catHAG:
		resolveHAG(w, target, isRear, defender, rng)
	case catATM:
		resolveATM(w, target, isRear, attacker, defender, rng)
	case catMML:
		resolveMML(w, target, isRear, attacker, defender, rng)
	default:
		// Normal single-hit weapon
		if roll2d6(rng) >= target {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, w.Damage, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
		}
	}
}

func rollHitLocation(isRear bool, rng *rand.Rand) int {
	roll := roll2d6(rng) - 2
	if isRear {
		return rearHitTable[roll]
	}
	return frontHitTable[roll]
}

func resolveArrowIV(w *SimWeapon, target int, defender *MechState, rng *rand.Rand) {
	// Arrow IV ignores target TMM in target number (already encoded)
	if roll2d6(rng) >= target {
		// Hit: 20 damage in 5-point groups
		for dmg := w.RackSize; dmg > 0; dmg -= 5 {
			d := 5
			if dmg < 5 {
				d = dmg
			}
			loc := rollHitLocation(false, rng)
			defender.applyDamage(loc, d, false, rng)
		}
	} else {
		// Miss: 1/6 chance of 1-hex scatter
		if roll1d6(rng) == 1 {
			for dmg := 10; dmg > 0; dmg -= 5 {
				loc := rollHitLocation(false, rng)
				defender.applyDamage(loc, 5, false, rng)
			}
		}
	}
}

func resolveStreakSRM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) {
	if roll2d6(rng) >= target {
		// All missiles hit, but AMS can reduce
		hits := amsIntercept(w.RackSize, defender, rng)
		for m := 0; m < hits; m++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, 2, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
		}
	}
}

func resolveUltraAC(w *SimWeapon, target int, isRear bool, defender *MechState, rng *rand.Rand) {
	// First shot
	if roll2d6(rng) >= target {
		loc := rollHitLocation(isRear, rng)
		defender.applyDamage(loc, w.Damage, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
	}
	// Second shot
	secondRoll := roll2d6(rng)
	if secondRoll == 2 {
		w.Jammed = true
		return
	}
	if secondRoll >= target {
		loc := rollHitLocation(isRear, rng)
		defender.applyDamage(loc, w.Damage, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
	}
}

func resolveRotaryAC(w *SimWeapon, target int, isRear bool, defender *MechState, rng *rand.Rand) {
	// Fire 6 times (maximize damage)
	for shot := 0; shot < 6; shot++ {
		r := roll2d6(rng)
		if r == 2 {
			w.Jammed = true
			return
		}
		if r >= target {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, w.Damage, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
		}
	}
}

func resolveLBX(w *SimWeapon, target int, isRear bool, defender *MechState, rng *rand.Rand) {
	// Cluster mode
	if roll2d6(rng) >= target {
		hits := clusterHits(w.RackSize, rng)
		for h := 0; h < hits; h++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, 1, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
		}
	}
}

func resolveLRM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) {
	if roll2d6(rng) >= target {
		hits := clusterHitsWithBonus(w.RackSize, artemisBonus(attacker), rng)
		hits = amsIntercept(hits, defender, rng)
		// Apply in 5-point groups
		for hits > 0 {
			grp := 5
			if hits < 5 {
				grp = hits
			}
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			hits -= grp
		}
	}
}

func resolveSRM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) {
	if roll2d6(rng) >= target {
		hits := clusterHitsWithBonus(w.RackSize, artemisBonus(attacker), rng)
		hits = amsIntercept(hits, defender, rng)
		for h := 0; h < hits; h++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, 2, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
		}
	}
}

func resolveMRM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) {
	if roll2d6(rng) >= target {
		hits := clusterHits(w.RackSize, rng) // MRM not Artemis-compatible
		hits = amsIntercept(hits, defender, rng)
		for h := 0; h < hits; h++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, 1, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
		}
	}
}

func resolveHAG(w *SimWeapon, target int, isRear bool, defender *MechState, rng *rand.Rand) {
	if roll2d6(rng) >= target {
		hits := clusterHits(w.RackSize, rng)
		// Apply in 5-point groups
		for hits > 0 {
			grp := 5
			if hits < 5 {
				grp = hits
			}
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			hits -= grp
		}
	}
}

func resolveATM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) {
	// ATM not directly supported in DB ranges; use standard mode (damage 2 per missile)
	if roll2d6(rng) >= target {
		hits := clusterHitsWithBonus(w.RackSize, artemisBonus(attacker), rng)
		hits = amsIntercept(hits, defender, rng)
		for h := 0; h < hits; h++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, 2, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
		}
	}
}

func resolveMML(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) {
	// Use LRM mode by default (ranges in DB are LRM)
	if roll2d6(rng) >= target {
		hits := clusterHitsWithBonus(w.RackSize, artemisBonus(attacker), rng)
		hits = amsIntercept(hits, defender, rng)
		for hits > 0 {
			grp := 5
			if hits < 5 {
				grp = hits
			}
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			hits -= grp
		}
	}
}

func resolvePhysical(attacker, defender *MechState, atkMoveMod int, rng *rand.Rand) {
	// Kick: damage = tonnage/5, to-hit = piloting - 2 + moveMod
	kickDmg := attacker.Tonnage / 5
	kickTarget := pilotingSkill - 2 + atkMoveMod
	if kickTarget < 2 {
		kickTarget = 2
	}
	if roll2d6(rng) >= kickTarget {
		// Kick hits legs
		loc := LocLL
		if rng.IntN(2) == 1 {
			loc = LocRL
		}
		defender.applyDamage(loc, kickDmg, false, rng)
	}
}

// calcExpectedDamage estimates the expected damage of the attacker at given distance
func calcExpectedDamage(m *MechState, dist int, baseTarget int, defTMM int, flanking float64) float64 {
	total := 0.0
	for i := range m.Weapons {
		w := &m.Weapons[i]
		if w.Destroyed || w.Jammed {
			continue
		}
		if w.AmmoKey != "" && m.Ammo[w.AmmoKey] <= 0 {
			continue
		}
		ed := weaponExpectedDamage(w, dist, baseTarget+defTMM)
		total += ed
	}
	return total
}

func weaponExpectedDamage(w *SimWeapon, dist int, target int) float64 {
	if w.Category == catArrowIV {
		if dist < w.MinRange || dist > w.LongRange {
			return 0
		}
		t := target + w.ToHitMod
		p := hitProb(t)
		return float64(w.RackSize)*p + float64(w.RackSize-10)/6.0*(1-p)
	}

	rm := rangeModifier(w, dist)
	if rm < 0 {
		return 0
	}

	t := target + w.ToHitMod + rm
	if w.MinRange > 0 && dist <= w.MinRange {
		t += w.MinRange - dist + 1
	}

	p := hitProb(t)

	switch w.Category {
	case catStreakSRM:
		return float64(w.RackSize) * 2 * p
	case catUltraAC:
		return float64(w.Damage) * 2 * p // approximate: 2 shots
	case catRotaryAC:
		return float64(w.Damage) * 6 * p // approximate: 6 shots
	case catLBX:
		return float64(w.RackSize) * p * 0.7 // cluster approximation
	case catLRM:
		return float64(w.RackSize) * p * 0.58
	case catSRM:
		return float64(w.RackSize) * 2 * p * 0.58
	case catMRM:
		return float64(w.RackSize) * p * 0.58
	case catHAG:
		return float64(w.RackSize) * p * 0.58
	case catATM:
		// Use best mode based on distance
		// Standard: 2dmg, 5/10/15; HE: 3dmg, 3/6/9; ER: 1dmg, 9/18/27
		bestDmg := 0.0
		// Standard mode
		if dist >= 4 && dist <= 15 {
			rm := 0
			if dist <= 5 {
				rm = 0
			} else if dist <= 10 {
				rm = 2
			} else {
				rm = 4
			}
			minP := 0
			if dist <= 4 {
				minP = 4 - dist + 1
			}
			tp := target + w.ToHitMod + rm + minP
			bestDmg = float64(w.RackSize) * 2 * hitProb(tp) * 0.58
		}
		// HE mode
		if dist <= 9 {
			rm := 0
			if dist <= 3 {
				rm = 0
			} else if dist <= 6 {
				rm = 2
			} else {
				rm = 4
			}
			tp := target + w.ToHitMod + rm
			d := float64(w.RackSize) * 3 * hitProb(tp) * 0.58
			if d > bestDmg {
				bestDmg = d
			}
		}
		// ER mode
		if dist >= 4 && dist <= 27 {
			rm := 0
			if dist <= 9 {
				rm = 0
			} else if dist <= 18 {
				rm = 2
			} else {
				rm = 4
			}
			minP := 0
			if dist <= 4 {
				minP = 4 - dist + 1
			}
			tp := target + w.ToHitMod + rm + minP
			d := float64(w.RackSize) * 1 * hitProb(tp) * 0.58
			if d > bestDmg {
				bestDmg = d
			}
		}
		return bestDmg
	case catMML:
		// LRM mode
		bestDmg := 0.0
		if dist >= 6 && dist <= 21 {
			rm := 0
			if dist <= 7 {
				rm = 0
			} else if dist <= 14 {
				rm = 2
			} else {
				rm = 4
			}
			minP := 0
			if dist <= 6 {
				minP = 6 - dist + 1
			}
			tp := target + w.ToHitMod + rm + minP
			bestDmg = float64(w.RackSize) * 1 * hitProb(tp) * 0.58
		}
		// SRM mode
		if dist <= 9 {
			rm := 0
			if dist <= 3 {
				rm = 0
			} else if dist <= 6 {
				rm = 2
			} else {
				rm = 4
			}
			tp := target + w.ToHitMod + rm
			d := float64(w.RackSize) * 2 * hitProb(tp) * 0.58
			if d > bestDmg {
				bestDmg = d
			}
		}
		return bestDmg
	default:
		return float64(w.Damage) * p
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ─── Main ───────────────────────────────────────────────────────────────────

func main() {
	mechFilter := flag.String("mech", "", "Comma-separated mech names to test (e.g. 'HBK-4P,AWS-8Q')")
	flag.Parse()

	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("DB: %v", err)
	}
	defer pool.Close()

	// Load MTF files
	mtfDir := filepath.Join("..", "..", "data", "megamek-data", "data", "mekfiles")
	if _, err := os.Stat(mtfDir); err != nil {
		// Try absolute path
		mtfDir = "/Users/puckopenclaw/projects/slic/data/megamek-data/data/mekfiles"
	}

	log.Println("Loading MTF files...")
	mtfMap := make(map[string]*ingestion.MTFData) // "Chassis Model" -> MTFData
	err = filepath.Walk(mtfDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".mtf") {
			return nil
		}
		data, err := ingestion.ParseMTF(path)
		if err != nil {
			return nil
		}
		mtfMap[data.FullName()] = data
		return nil
	})
	if err != nil {
		log.Fatalf("Walk MTF: %v", err)
	}
	log.Printf("Loaded %d MTF files", len(mtfMap))

	// Load variants from DB
	log.Println("Loading variants from DB...")
	rows, err := pool.Query(ctx, `
		SELECT v.id, v.name, v.model_code, COALESCE(c.tech_base, 'Inner Sphere'),
			   c.tonnage, vs.walk_mp, vs.run_mp, vs.jump_mp,
			   vs.heat_sink_count, vs.heat_sink_type, vs.engine_type,
			   COALESCE(vs.structure_type, 'Standard'), COALESCE(vs.has_targeting_computer, false)
		FROM variants v
		JOIN chassis c ON c.id = v.chassis_id
		JOIN variant_stats vs ON vs.variant_id = v.id`)
	if err != nil {
		log.Fatalf("Query variants: %v", err)
	}

	var variants []DBVariant
	for rows.Next() {
		var v DBVariant
		err := rows.Scan(&v.ID, &v.Name, &v.ModelCode, &v.TechBase,
			&v.Tonnage, &v.WalkMP, &v.RunMP, &v.JumpMP,
			&v.HSCount, &v.HSType, &v.EngineType,
			&v.StructType, &v.HasTC)
		if err != nil {
			continue
		}
		variants = append(variants, v)
	}
	rows.Close()
	log.Printf("Loaded %d variants", len(variants))

	// Filter variants if -mech flag provided
	if *mechFilter != "" {
		filters := strings.Split(*mechFilter, ",")
		var filtered []DBVariant
		for _, v := range variants {
			for _, f := range filters {
				f = strings.TrimSpace(f)
				if strings.Contains(v.ModelCode, f) || strings.Contains(v.Name+" "+v.ModelCode, f) {
					filtered = append(filtered, v)
					break
				}
			}
		}
		log.Printf("Filtered to %d variants matching %q", len(filtered), *mechFilter)
		variants = filtered
	}

	// Load weapons for all variants
	log.Println("Loading weapons...")
	for i := range variants {
		v := &variants[i]
		wRows, err := pool.Query(ctx, `
			SELECT e.name, COALESCE(e.type,''), COALESCE(e.damage,0), COALESCE(e.heat,0),
				   COALESCE(e.min_range,0), COALESCE(e.short_range,0), COALESCE(e.medium_range,0),
				   COALESCE(e.long_range,0), COALESCE(e.to_hit_modifier,0), COALESCE(e.rack_size,0),
				   ve.location, ve.quantity
			FROM variant_equipment ve
			JOIN equipment e ON e.id = ve.equipment_id
			WHERE ve.variant_id = $1
			  AND e.type IN ('energy','ballistic','missile','artillery')`, v.ID)
		if err != nil {
			continue
		}
		for wRows.Next() {
			var w DBWeapon
			wRows.Scan(&w.Name, &w.Type, &w.Damage, &w.Heat,
				&w.MinRange, &w.Short, &w.Medium, &w.Long,
				&w.ToHitMod, &w.RackSize, &w.Location, &w.Quantity)
			v.Weapons = append(v.Weapons, w)
		}
		wRows.Close()
	}

	// Run HBK-4P baseline first
	log.Println("Running HBK-4P baseline...")
	hbkTemplate := buildHBK4P()

	baselineOffense := runSimsBatch(hbkTemplate, hbkTemplate, numSims)
	baselineDefense := runSimsBatch(hbkTemplate, hbkTemplate, numSims)
	baselineRatio := float64(baselineDefense) / float64(baselineOffense)
	if baselineRatio == 0 {
		baselineRatio = 1.0
	}
	log.Printf("HBK-4P baseline: offense=%.1f defense=%.1f ratio=%.3f",
		baselineOffense, baselineDefense, baselineRatio)

	// Process all variants with worker pool
	numWorkers := runtime.NumCPU()
	log.Printf("Processing %d variants with %d workers...", len(variants), numWorkers)

	type result struct {
		id       int
		offense  float64
		defense  float64
		score    float64
	}

	results := make(chan result, len(variants))
	jobs := make(chan int, len(variants))

	var processed atomic.Int64

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				v := &variants[idx]
				mtf := mtfMap[v.Name]
				if mtf == nil {
					// Try chassis + model_code
					altName := strings.TrimSuffix(v.Name, " "+v.ModelCode)
					altName = altName + " " + v.ModelCode
					mtf = mtfMap[altName]
				}

				mechTemplate := buildMechState(v, mtf)

				offTurns := runSimsBatch(mechTemplate, hbkTemplate, numSims)
				defTurns := runSimsBatch(hbkTemplate, mechTemplate, numSims)

				ratio := float64(defTurns) / float64(offTurns)
				score := 5.0 + kFactor*math.Log(ratio/baselineRatio)
				if score < 1 {
					score = 1
				}
				if score > 10 {
					score = 10
				}

				if *mechFilter != "" {
					log.Printf("  %s %s: offense=%.1f defense=%.1f score=%.2f",
						v.Name, v.ModelCode, offTurns, defTurns, score)
				}

				results <- result{v.ID, offTurns, defTurns, score}

				n := processed.Add(1)
				if n%100 == 0 {
					log.Printf("Progress: %d/%d variants", n, len(variants))
				}
			}
		}()
	}

	for i := range variants {
		jobs <- i
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and write results
	updated := 0
	for r := range results {
		_, err := pool.Exec(ctx, `
			UPDATE variant_stats SET combat_rating = $2, offense_turns = $3, defense_turns = $4
			WHERE variant_id = $1`, r.id, r.score, r.offense, r.defense)
		if err != nil {
			log.Printf("Update %d: %v", r.id, err)
			continue
		}
		updated++
	}

	log.Printf("Done! Updated %d variants", updated)
}

// runSimsBatch runs N sims and returns median turn count
func runSimsBatch(attackerTemplate, defenderTemplate *MechState, n int) float64 {
	results := make([]int, n)
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

	for i := 0; i < n; i++ {
		results[i] = simulateCombat(attackerTemplate, defenderTemplate, rng)
	}

	sort.Ints(results)
	if n%2 == 0 {
		return float64(results[n/2-1]+results[n/2]) / 2.0
	}
	return float64(results[n/2])
}
