package main

import (
	"math"
	"math/rand/v2"
	"sort"
	"strings"
)

// ─── Weapon classification ──────────────────────────────────────────────────

type weaponCategory int

const (
	catNormal    weaponCategory = iota
	catLRM
	catSRM
	catMRM
	catStreakSRM
	catStreakLRM
	catUltraAC
	catRotaryAC
	catLBX
	catHAG
	catATM
	catMML
	catArrowIV
	catRocketLauncher
	catPlasmaCannon
	catPlasmaRifle
	catVSP
)

func categorizeWeapon(name string) weaponCategory {
	upper := strings.ToUpper(name)
	switch {
	case strings.Contains(upper, "STREAK LRM"):
		return catStreakLRM
	case strings.Contains(upper, "STREAK SRM"):
		return catStreakSRM
	case strings.Contains(upper, "ROCKET LAUNCHER"):
		return catRocketLauncher
	case upper == "PLASMA CANNON" || upper == "CLPLASMA CANNON":
		return catPlasmaCannon
	case upper == "PLASMA RIFLE":
		return catPlasmaRifle
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
	case strings.Contains(upper, "VSP") || strings.Contains(upper, "VARIABLE SPEED PULSE"):
		return catVSP
	default:
		return catNormal
	}
}

func isDirectFire(cat weaponCategory) bool {
	switch cat {
	case catNormal, catUltraAC, catRotaryAC, catLBX, catHAG:
		return true
	default:
		return false
	}
}

// ─── SimWeapon ──────────────────────────────────────────────────────────────

type SimWeapon struct {
	Name       string
	Category   weaponCategory
	Location   int
	Damage     int
	Heat       int
	MinRange   int
	ShortRange int
	MedRange   int
	LongRange  int
	ToHitMod   int
	RackSize   int
	Type       string // energy, ballistic, missile, artillery
	AmmoKey    string
	Destroyed  bool
	Jammed     bool
}

// isArmWeapon returns true if the weapon is mounted in an arm
func (w *SimWeapon) isArmWeapon() bool {
	return w.Location == LocLA || w.Location == LocRA
}

// isTorsoWeapon returns true if the weapon is in a torso location
func (w *SimWeapon) isTorsoWeapon() bool {
	return w.Location == LocCT || w.Location == LocLT || w.Location == LocRT
}

// ─── Cluster hits table ─────────────────────────────────────────────────────

var clusterRackSizes = []int{2, 3, 4, 5, 6, 8, 9, 10, 12, 15, 20, 30, 40}
var clusterTable = [11][13]int{
	{1, 1, 1, 1, 2, 3, 3, 3, 4, 5, 6, 10, 12},    // roll 2
	{1, 1, 2, 2, 2, 3, 3, 3, 4, 5, 6, 10, 12},    // roll 3
	{1, 1, 2, 2, 3, 4, 4, 4, 5, 6, 9, 12, 18},    // roll 4
	{1, 2, 2, 3, 3, 4, 5, 6, 8, 9, 12, 18, 24},   // roll 5
	{1, 2, 2, 3, 4, 5, 5, 6, 8, 9, 12, 18, 24},   // roll 6
	{1, 2, 3, 3, 4, 5, 5, 6, 8, 9, 12, 18, 24},   // roll 7
	{2, 2, 3, 3, 4, 5, 5, 6, 8, 9, 12, 18, 24},   // roll 8
	{2, 2, 3, 4, 5, 6, 7, 8, 10, 12, 16, 24, 32}, // roll 9
	{2, 3, 3, 4, 5, 6, 7, 8, 10, 12, 16, 24, 32}, // roll 10
	{2, 3, 4, 5, 6, 8, 9, 10, 12, 15, 20, 30, 40}, // roll 11
	{2, 3, 4, 5, 6, 8, 9, 10, 12, 15, 20, 30, 40}, // roll 12
}

func clusterHits(rackSize int, rng *rand.Rand) int {
	return clusterHitsWithBonus(rackSize, 0, rng)
}

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

// ─── AMS ────────────────────────────────────────────────────────────────────

// amsClusterMod returns the cluster roll modifier from AMS (-4 per BMM p.118).
// Marks AMS as used and consumes ammo. Returns 0 if AMS unavailable.
func amsClusterMod(defender *MechState) int {
	if !defender.HasAMS || defender.AMSUsedThisTurn {
		return 0
	}
	if !defender.IsLaserAMS {
		if defender.AMSAmmo <= 0 {
			return 0
		}
		defender.AMSAmmo--
	}
	defender.AMSUsedThisTurn = true
	return -4
}

// amsInterceptStreak handles AMS vs Streak weapons (BMM p.118):
// treat as cluster roll of 7 on the appropriate rack size column.
func amsInterceptStreak(rackSize int, defender *MechState) int {
	if !defender.HasAMS || defender.AMSUsedThisTurn {
		return rackSize
	}
	if !defender.IsLaserAMS {
		if defender.AMSAmmo <= 0 {
			return rackSize
		}
		defender.AMSAmmo--
	}
	defender.AMSUsedThisTurn = true
	colIdx := 0
	for i, rs := range clusterRackSizes {
		if rs <= rackSize {
			colIdx = i
		}
	}
	return clusterTable[5][colIdx] // roll 7 = row index 5
}

func artemisBonus(m *MechState) int {
	if m.HasArtemisV {
		return 3
	}
	if m.HasArtemisIV {
		return 2
	}
	return 0
}

// ─── Range modifier (2D: uses hex distance) ────────────────────────────────

func rangeModifier(w *SimWeapon, dist int) int {
	if w.Category == catArrowIV {
		if dist >= w.MinRange && dist <= w.LongRange {
			return 0
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

// ─── canWeaponFire checks arc constraints for weapon firing ─────────────────

// canWeaponFire returns true if the weapon can fire at the target given the
// mech's facing and torso twist.
func canWeaponFire(w *SimWeapon, mechPos HexCoord, facing int, torsoTwist int, targetPos HexCoord) bool {
	// Determine the effective facing with torso twist
	effFacing := ((facing + torsoTwist) % 6 + 6) % 6

	// Arm weapons fire in the arm's own forward arc (based on torso twist for torso-mounted arms)
	// In BT, arm weapons can fire into the forward arc + the arm's side arc
	// Torso weapons fire in the torso's arc (affected by twist)
	// Head weapons fire forward

	arc := DetermineArc(mechPos, effFacing, targetPos)

	switch {
	case w.Location == LocHD:
		// Head weapons fire forward only (affected by twist)
		return arc == ArcFront
	case w.isArmWeapon():
		// Arm weapons can fire into forward arc + their side arc
		// With torso twist, effective arc shifts
		return arc == ArcFront || arc == ArcLeft || arc == ArcRight
	case w.isTorsoWeapon():
		// Torso weapons fire in forward arc (with twist)
		return arc == ArcFront
	}
	return arc == ArcFront
}

// ─── Weapon fire resolution (2D-aware) ──────────────────────────────────────

// resolveWeaponFire2D resolves weapon fire with 2D hex grid awareness.
// Returns total damage dealt (for PSR tracking).
func resolveWeaponFire2D(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	switch w.Category {
	case catArrowIV:
		return resolveArrowIV(w, target, defender, rng)
	case catStreakSRM:
		return resolveStreakSRM(w, target, isRear, attacker, defender, rng)
	case catStreakLRM:
		return resolveStreakLRM(w, target, isRear, attacker, defender, rng)
	case catRocketLauncher:
		return resolveRocketLauncher(w, target, isRear, defender, rng)
	case catPlasmaCannon:
		return resolvePlasmaCannon(w, target, defender, rng)
	case catPlasmaRifle:
		return resolvePlasmaRifle(w, target, isRear, defender, rng)
	case catUltraAC:
		return resolveUltraAC(w, target, isRear, defender, rng)
	case catRotaryAC:
		return resolveRotaryAC(w, target, isRear, defender, rng)
	case catLBX:
		return resolveLBX(w, target, isRear, defender, rng)
	case catLRM:
		return resolveLRM(w, target, isRear, attacker, defender, rng)
	case catSRM:
		return resolveSRM(w, target, isRear, attacker, defender, rng)
	case catMRM:
		return resolveMRM(w, target, isRear, attacker, defender, rng)
	case catHAG:
		return resolveHAG(w, target, isRear, attacker, defender, rng)
	case catATM:
		return resolveATM(w, target, isRear, attacker, defender, rng)
	case catMML:
		return resolveMML(w, target, isRear, attacker, defender, rng)
	case catVSP:
		return resolveVSP(w, target, isRear, attacker, defender, rng)
	default:
		if roll2d6(rng) >= target {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, w.Damage, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			return w.Damage
		}
		return 0
	}
}

func resolveArrowIV(w *SimWeapon, target int, defender *MechState, rng *rand.Rand) int {
	dmgDealt := 0
	if roll2d6(rng) >= target {
		for dmg := w.RackSize; dmg > 0; dmg -= 5 {
			d := 5
			if dmg < 5 {
				d = dmg
			}
			loc := rollHitLocation(false, rng)
			defender.applyDamage(loc, d, false, rng)
			dmgDealt += d
		}
	} else {
		if roll1d6(rng) == 1 {
			for dmg := 10; dmg > 0; dmg -= 5 {
				loc := rollHitLocation(false, rng)
				defender.applyDamage(loc, 5, false, rng)
				dmgDealt += 5
			}
		}
	}
	return dmgDealt
}

func resolveStreakSRM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	dmgDealt := 0
	if roll2d6(rng) >= target {
		hits := amsInterceptStreak(w.RackSize, defender)
		for m := 0; m < hits; m++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, 2, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += 2
		}
	}
	return dmgDealt
}

func resolveUltraAC(w *SimWeapon, target int, isRear bool, defender *MechState, rng *rand.Rand) int {
	// Ultra AC rapid-fire R2: ONE to-hit roll, jam on natural 2, cluster table column 2 for hits.
	// Even if jammed, the attack still resolves normally.
	r := roll2d6(rng)
	if r == 2 {
		w.Jammed = true // permanently jammed for rest of game
	}
	dmgDealt := 0
	if r >= target {
		hits := clusterHits(2, rng) // cluster table column "2" → 1 or 2 hits
		for h := 0; h < hits; h++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, w.Damage, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += w.Damage
		}
	}
	return dmgDealt
}

// racOptimalShots returns the optimal number of shots (2-6) for a Rotary AC
// based on single-turn EV considering jam probability.
// Jam thresholds: 2-3 shots → jam on 2, 4-5 → jam on ≤3, 6 → jam on ≤4.
// RAC unjams next turn (1-turn cooldown), so jam cost = 1 turn of lost damage.
func racOptimalShots(w *SimWeapon, hitProb float64) int {
	bestShots := 2
	bestEV := 0.0
	// Approximate future damage per turn (used to cost jam penalty)
	avgFutureDmg := float64(w.Damage) * 4 * hitProb * 0.58 // rough mid estimate
	for shots := 2; shots <= 6; shots++ {
		var jamProb float64
		switch {
		case shots <= 3:
			jamProb = 1.0 / 36.0 // natural 2
		case shots <= 5:
			jamProb = 3.0 / 36.0 // ≤3
		default:
			jamProb = 6.0 / 36.0 // ≤4
		}
		// EV = hits * damage * hitProb * clusterFraction - jamProb * futureDmgLost
		clusterAvg := clusterAverage(shots)
		ev := clusterAvg*float64(w.Damage)*hitProb - jamProb*avgFutureDmg
		if ev > bestEV {
			bestEV = ev
			bestShots = shots
		}
	}
	return bestShots
}

// clusterAverage returns the average number of hits from the cluster table for a given rack size.
func clusterAverage(rackSize int) float64 {
	colIdx := 0
	for i, rs := range clusterRackSizes {
		if rs <= rackSize {
			colIdx = i
		}
	}
	total := 0
	for row := 0; row < 11; row++ {
		total += clusterTable[row][colIdx]
	}
	// Each row is equally likely on 2d6? No — 2d6 distribution.
	// Weights for 2d6 results 2-12: 1,2,3,4,5,6,5,4,3,2,1 (out of 36)
	weights := [11]int{1, 2, 3, 4, 5, 6, 5, 4, 3, 2, 1}
	weighted := 0.0
	for row := 0; row < 11; row++ {
		weighted += float64(clusterTable[row][colIdx]) * float64(weights[row])
	}
	return weighted / 36.0
}

func resolveRotaryAC(w *SimWeapon, target int, isRear bool, defender *MechState, rng *rand.Rand) int {
	// RAC rapid-fire R6: choose optimal shot count, ONE to-hit roll.
	// Jam thresholds vary by shots fired. RAC unjams next turn (not permanent).
	p := hitProb(target)
	shots := racOptimalShots(w, p)

	r := roll2d6(rng)

	// Jam check based on shots fired
	var jamThreshold int
	switch {
	case shots <= 3:
		jamThreshold = 2
	case shots <= 5:
		jamThreshold = 3
	default:
		jamThreshold = 4
	}
	if r <= jamThreshold {
		w.Jammed = true // unjams next turn (handled in turn loop)
	}

	// Attack still resolves even on jam
	dmgDealt := 0
	if r >= target {
		hits := clusterHits(shots, rng)
		for h := 0; h < hits; h++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, w.Damage, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += w.Damage
		}
	}
	return dmgDealt
}

func resolveLBX(w *SimWeapon, target int, isRear bool, defender *MechState, rng *rand.Rand) int {
	dmgDealt := 0
	if roll2d6(rng) >= target {
		hits := clusterHits(w.RackSize, rng)
		for h := 0; h < hits; h++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, 1, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt++
		}
	}
	return dmgDealt
}

func resolveLRM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	dmgDealt := 0
	if roll2d6(rng) >= target {
		// AMS applies -4 to cluster roll (BMM p.118)
		hits := clusterHitsWithBonus(w.RackSize, artemisBonus(attacker)+amsClusterMod(defender), rng)
		if hits <= 0 {
			return 0
		}
		for hits > 0 {
			grp := 5
			if hits < 5 {
				grp = hits
			}
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += grp
			hits -= grp
		}
	}
	return dmgDealt
}

func resolveSRM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	dmgDealt := 0
	if roll2d6(rng) >= target {
		hits := clusterHitsWithBonus(w.RackSize, artemisBonus(attacker)+amsClusterMod(defender), rng)
		for h := 0; h < hits; h++ {
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, 2, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += 2
		}
	}
	return dmgDealt
}

func resolveMRM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	dmgDealt := 0
	if roll2d6(rng) >= target {
		bonus := 0
		if attacker.HasApollo {
			bonus = -1 // Apollo FCS: subtract 1 from cluster roll (BMM p.113)
		}
		hits := clusterHitsWithBonus(w.RackSize, bonus+amsClusterMod(defender), rng)
		// MRM is C5: 1 dmg/missile, apply in 5-point groups
		totalDmg := hits // hits * 1 dmg per missile
		for totalDmg > 0 {
			grp := 5
			if totalDmg < 5 {
				grp = totalDmg
			}
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += grp
			totalDmg -= grp
		}
	}
	return dmgDealt
}

func resolveHAG(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	dmgDealt := 0
	if roll2d6(rng) >= target {
		// HAG cluster modifier: +2 short range, -2 long range (BMM p.100)
		dist := HexDistance(attacker.Pos, defender.Pos)
		hagBonus := 0
		if dist <= w.ShortRange {
			hagBonus = 2
		} else if dist > w.MedRange {
			hagBonus = -2
		}
		hits := clusterHitsWithBonus(w.RackSize, hagBonus, rng)
		for hits > 0 {
			grp := 5
			if hits < 5 {
				grp = hits
			}
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += grp
			hits -= grp
		}
	}
	return dmgDealt
}

func resolveATM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	dist := HexDistance(attacker.Pos, defender.Pos)
	// ATM damage per missile depends on range mode:
	// HE (short): 3 dmg, range 0-9
	// Standard (medium): 2 dmg, range 4-15
	// ER (long): 1 dmg, range 4-27
	// The weapon selection already picked the best mode; determine from actual distance
	dmgPerMissile := 2 // default standard
	if dist <= 3 {
		dmgPerMissile = 3 // HE mode (short range)
	} else if dist <= 9 {
		dmgPerMissile = 3 // HE mode still best at this range
	} else if dist <= 15 {
		dmgPerMissile = 2 // standard mode
	} else {
		dmgPerMissile = 1 // ER mode
	}

	dmgDealt := 0
	if roll2d6(rng) >= target {
		hits := clusterHitsWithBonus(w.RackSize, artemisBonus(attacker)+amsClusterMod(defender), rng)
		// ATM is C5: total damage = hits * dmgPerMissile, apply in 5-point groups
		totalDmg := hits * dmgPerMissile
		for totalDmg > 0 {
			grp := 5
			if totalDmg < 5 {
				grp = totalDmg
			}
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += grp
			totalDmg -= grp
		}
	}
	return dmgDealt
}

func resolveMML(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	dist := HexDistance(attacker.Pos, defender.Pos)
	// MML switches between SRM mode (short range, 2 dmg/missile, individual locs)
	// and LRM mode (long range, 5-point groups)
	// SRM mode: range 0 to ShortRange (which is the SRM short range)
	// LRM mode: range MinRange to LongRange
	useSRMMode := dist <= w.ShortRange && dist < w.MinRange+1 // if within SRM range and not forced to LRM

	// Simpler: MML SRM range is typically 0-3/6/9, LRM range is 6-7/14/21
	// If distance <= short range of the SRM component (roughly rackSize-dependent, ~3 hexes for short),
	// use SRM mode. The weapon's ShortRange in the DB reflects the LRM short range.
	// Use SRM mode if dist <= 9 (SRM max range for large racks)
	useSRMMode = dist <= 9 // SRM mode within 9 hexes

	dmgDealt := 0
	if roll2d6(rng) >= target {
		hits := clusterHitsWithBonus(w.RackSize, artemisBonus(attacker)+amsClusterMod(defender), rng)
		if useSRMMode {
			// SRM mode: 2 damage per missile, individual hit locations
			for h := 0; h < hits; h++ {
				loc := rollHitLocation(isRear, rng)
				defender.applyDamage(loc, 2, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
				dmgDealt += 2
			}
		} else {
			// LRM mode: 1 damage per missile, 5-point groups
			for hits > 0 {
				grp := 5
				if hits < 5 {
					grp = hits
				}
				loc := rollHitLocation(isRear, rng)
				defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
				dmgDealt += grp
				hits -= grp
			}
		}
	}
	return dmgDealt
}

func resolveStreakLRM(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	// Streak LRM: all missiles hit on successful to-hit (no cluster roll). 1 dmg/missile, 5-point groupings.
	dmgDealt := 0
	if roll2d6(rng) >= target {
		hits := amsInterceptStreak(w.RackSize, defender) // Streak: AMS uses cluster roll 7 (BMM p.118)
		for hits > 0 {
			grp := 5
			if hits < 5 {
				grp = hits
			}
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += grp
			hits -= grp
		}
	}
	return dmgDealt
}

func resolveRocketLauncher(w *SimWeapon, target int, isRear bool, defender *MechState, rng *rand.Rand) int {
	// Rocket Launchers: one-shot cluster weapon. 1 dmg/missile, 5-point groupings.
	// After firing, weapon is spent (destroyed).
	w.Destroyed = true // one-shot weapon
	dmgDealt := 0
	if roll2d6(rng) >= target {
		hits := clusterHits(w.RackSize, rng)
		for hits > 0 {
			grp := 5
			if hits < 5 {
				grp = hits
			}
			loc := rollHitLocation(isRear, rng)
			defender.applyDamage(loc, grp, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
			dmgDealt += grp
			hits -= grp
		}
	}
	return dmgDealt
}

func resolvePlasmaCannon(w *SimWeapon, target int, defender *MechState, rng *rand.Rand) int {
	// Plasma Cannon: 0 damage, applies 2d6 heat to target on hit.
	if roll2d6(rng) >= target {
		heatApplied := roll2d6(rng)
		defender.HeatPenalty += heatApplied
	}
	return 0
}

func resolvePlasmaRifle(w *SimWeapon, target int, isRear bool, defender *MechState, rng *rand.Rand) int {
	// Plasma Rifle: 10 damage + heat to target on hit.
	if roll2d6(rng) >= target {
		loc := rollHitLocation(isRear, rng)
		defender.applyDamage(loc, w.Damage, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
		heatApplied := roll2d6(rng)
		defender.HeatPenalty += heatApplied
		return w.Damage
	}
	return 0
}

// vspDamageByRange returns the variable damage for VSP lasers based on range bracket.
// Small VSP: 5/4/3, Medium VSP: 9/7/5, Large VSP: 11/9/7
func vspDamageByRange(w *SimWeapon, dist int) int {
	// w.Damage is set to max (short range) value in DB
	shortDmg := w.Damage
	var medDmg, longDmg int
	switch shortDmg {
	case 5: // Small VSP
		medDmg, longDmg = 4, 3
	case 9: // Medium VSP
		medDmg, longDmg = 7, 5
	case 11: // Large VSP
		medDmg, longDmg = 9, 7
	default:
		medDmg, longDmg = shortDmg, shortDmg
	}
	if dist <= w.ShortRange {
		return shortDmg
	}
	if dist <= w.MedRange {
		return medDmg
	}
	return longDmg
}

// vspToHitMod returns the VSP to-hit modifier by range: -3 short, -2 medium, -1 long
func vspToHitMod(w *SimWeapon, dist int) int {
	if dist <= w.ShortRange {
		return -3
	}
	if dist <= w.MedRange {
		return -2
	}
	return -1
}

func resolveVSP(w *SimWeapon, target int, isRear bool, attacker *MechState, defender *MechState, rng *rand.Rand) int {
	// VSP target already includes range modifier from rangeModifier() but NOT the VSP-specific to-hit mod.
	// We need to adjust: remove the DB toHitMod (0) and apply vspToHitMod instead.
	// Actually, the target number is computed externally with w.ToHitMod (0 from DB) + rangeModifier.
	// We need the distance to compute VSP mod. Use attacker/defender positions.
	dist := HexDistance(attacker.Pos, defender.Pos)
	vspMod := vspToHitMod(w, dist)
	adjustedTarget := target + vspMod // add the VSP pulse bonus
	dmg := vspDamageByRange(w, dist)

	if roll2d6(rng) >= adjustedTarget {
		loc := rollHitLocation(isRear, rng)
		defender.applyDamage(loc, dmg, isRear && (loc == LocCT || loc == LocLT || loc == LocRT), rng)
		return dmg
	}
	return 0
}

// effectiveWeaponHeat returns the actual heat generated when firing a weapon,
// accounting for rapid-fire modes (UAC 2×, RAC shots×).
func effectiveWeaponHeat(w *SimWeapon) int {
	switch w.Category {
	case catUltraAC:
		return w.Heat * 2 // always rapid-fire 2 shots
	case catRotaryAC:
		return w.Heat * 4 // default ~4 shots (heuristic, actual chosen at resolve time)
	default:
		return w.Heat
	}
}

// ─── EV-based weapon selection ──────────────────────────────────────────────

// selectWeaponsEV selects which weapons to fire based on EV-based heat management.
// Returns indices into mech.Weapons and total weapon heat.
func selectWeaponsEV(mech *MechState, board *Board, defender *MechState, dist int, baseTarget int) ([]int, int) {
	type weaponFire struct {
		idx    int
		expDmg float64
		heat   int
	}
	var candidates []weaponFire

	// Check LOS and arc for each weapon
	for i := range mech.Weapons {
		w := &mech.Weapons[i]
		if w.Destroyed || w.Jammed {
			continue
		}
		if w.AmmoKey != "" && mech.Ammo[w.AmmoKey] <= 0 {
			continue
		}

		// Check if weapon can fire at target (arc constraint)
		if !canWeaponFire(w, mech.Pos, mech.Facing, mech.TorsoTwist, defender.Pos) {
			continue
		}

		ed := weaponExpectedDamage(w, dist, baseTarget)
		if ed <= 0 {
			continue
		}
		candidates = append(candidates, weaponFire{i, ed, effectiveWeaponHeat(w)})
	}

	// Sort by damage/heat ratio
	sort.Slice(candidates, func(a, b int) bool {
		ra := candidates[a].expDmg / math.Max(float64(candidates[a].heat), 0.1)
		rb := candidates[b].expDmg / math.Max(float64(candidates[b].heat), 0.1)
		return ra > rb
	})

	avgTurnDmg := 0.0
	for _, c := range candidates {
		avgTurnDmg += c.expDmg
	}
	ammoExpDmg := mechAmmoExplosionDamage(mech)

	var firingWeapons []int
	weaponHeatTotal := 0
	heatThisMod := heatToHitMod(mech.Heat)

	for _, c := range candidates {
		if c.heat == 0 {
			firingWeapons = append(firingWeapons, c.idx)
			continue
		}

		newWeaponHeat := weaponHeatTotal + c.heat
		projectedHeat := mech.Heat + newWeaponHeat - mech.Dissipation
		if projectedHeat < 0 {
			projectedHeat = 0
		}

		oldProjectedHeat := mech.Heat + weaponHeatTotal - mech.Dissipation
		if oldProjectedHeat < 0 {
			oldProjectedHeat = 0
		}

		oldCost := heatCostEV(oldProjectedHeat, avgTurnDmg, ammoExpDmg, mech.WalkMP, avgTurnDmg)
		newCost := heatCostEV(projectedHeat, avgTurnDmg, ammoExpDmg, mech.WalkMP, avgTurnDmg)
		marginalCost := newCost - oldCost

		oldToHitMod := heatToHitMod(oldProjectedHeat)
		newToHitMod := heatToHitMod(projectedHeat)
		toHitPenaltyCost := 0.0
		if newToHitMod > oldToHitMod {
			for _, fi := range firingWeapons {
				fw := &mech.Weapons[fi]
				oldDmg := weaponExpectedDamage(fw, dist, baseTarget+oldToHitMod-heatThisMod)
				newDmg := weaponExpectedDamage(fw, dist, baseTarget+newToHitMod-heatThisMod)
				toHitPenaltyCost += oldDmg - newDmg
			}
		}

		actualDmg := weaponExpectedDamage(&mech.Weapons[c.idx], dist, baseTarget+newToHitMod-heatThisMod)
		marginalEV := actualDmg - marginalCost - toHitPenaltyCost

		if marginalEV > 0 {
			firingWeapons = append(firingWeapons, c.idx)
			weaponHeatTotal += c.heat
		}
	}

	return firingWeapons, weaponHeatTotal
}

// ─── Expected damage calculation ────────────────────────────────────────────

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
	case catStreakLRM:
		return float64(w.RackSize) * 1 * p
	case catUltraAC:
		// Average hits from cluster table column 2, factoring jam (permanent loss)
		avgHits := clusterAverage(2) // ~1.39
		jamProb := 1.0 / 36.0
		return float64(w.Damage) * avgHits * p * (1 - jamProb)
	case catRotaryAC:
		// Use optimal shot count EV
		bestEV := 0.0
		for shots := 2; shots <= 6; shots++ {
			var jamProb float64
			switch {
			case shots <= 3:
				jamProb = 1.0 / 36.0
			case shots <= 5:
				jamProb = 3.0 / 36.0
			default:
				jamProb = 6.0 / 36.0
			}
			avgHits := clusterAverage(shots)
			// RAC jam = 1 turn lost, approximate cost as losing this turn's EV
			ev := float64(w.Damage) * avgHits * p * (1 - jamProb)
			if ev > bestEV {
				bestEV = ev
			}
		}
		return bestEV
	case catRocketLauncher:
		// One-shot cluster weapon, 1 dmg/missile, C5 grouping
		return float64(w.RackSize) * p * 0.58
	case catPlasmaCannon:
		// 0 damage but applies ~7 avg heat; model heat disruption as ~3.5 equivalent damage
		return 3.5 * p
	case catPlasmaRifle:
		// 10 damage + ~7 avg heat to target
		return (float64(w.Damage) + 3.5) * p
	case catLBX:
		return float64(w.RackSize) * p * 0.7
	case catLRM:
		return float64(w.RackSize) * p * 0.58
	case catSRM:
		return float64(w.RackSize) * 2 * p * 0.58
	case catMRM:
		return float64(w.RackSize) * p * 0.58
	case catHAG:
		return float64(w.RackSize) * p * 0.58
	case catATM:
		bestDmg := 0.0
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
	case catVSP:
		// VSP has range-dependent damage and to-hit mod; approximate with average
		// Short: -3 mod, max dmg; Med: -2 mod, mid dmg; Long: -1 mod, min dmg
		// Use the range bracket for this distance
		vspMod := -2 // average approximation
		dmg := float64(w.Damage) * 0.8 // rough average across ranges
		if dist <= w.ShortRange {
			vspMod = -3
			dmg = float64(w.Damage)
		} else if dist <= w.MedRange {
			vspMod = -2
			dmg = float64(vspDamageByRange(w, dist))
		} else {
			vspMod = -1
			dmg = float64(vspDamageByRange(w, dist))
		}
		adjustedT := t + vspMod
		return dmg * hitProb(adjustedT)
	default:
		return float64(w.Damage) * p
	}
}

func calcExpectedDamage(m *MechState, dist int, baseTarget int, defTMM int) float64 {
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
