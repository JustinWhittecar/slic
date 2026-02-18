package main

import "math/rand/v2"

// ─── PSR / Falling mechanics ────────────────────────────────────────────────

var consciousnessThresholds = [6]int{3, 5, 7, 10, 11, 99}

func (m *MechState) psrPreexistingMod() int {
	mod := m.GyroHits * 3
	for _, loc := range []int{LocLL, LocRL} {
		if m.IS[loc] <= 0 {
			mod += 5
		} else if m.HipHit[loc] {
			mod += 2
		} else {
			mod += m.LegFootHits[loc]
		}
	}
	return mod
}

func (m *MechState) rollPSR(extraMod int, rng *rand.Rand) bool {
	if m.Prone {
		return true
	}
	if m.PilotUnconscious || m.PilotDamage >= 6 {
		return false
	}
	if m.GyroHits >= 2 {
		return false
	}
	target := pilotingSkill + m.psrPreexistingMod() + extraMod
	return roll2d6(rng) >= target
}

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

func (m *MechState) applyFall(rng *rand.Rand) {
	m.Prone = true

	fallDmg := (m.Tonnage + 9) / 10
	facing := roll1d6(rng)

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

	target := pilotingSkill + m.psrPreexistingMod()
	if m.PilotUnconscious || roll2d6(rng) < target {
		m.PilotDamage++
		if m.PilotDamage >= 6 {
			return
		}
		threshold := consciousnessThresholds[m.PilotDamage-1]
		if roll2d6(rng) < threshold {
			m.PilotUnconscious = true
		}
	}
}

func rollFallingHitLocation(facing int, rng *rand.Rand) int {
	switch facing {
	case 1:
		return rollHitLocation(false, rng)
	case 4:
		return rollHitLocation(true, rng)
	default:
		return rollHitLocation(false, rng)
	}
}
