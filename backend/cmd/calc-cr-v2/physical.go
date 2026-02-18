package main

import "math/rand/v2"

// ─── Physical attacks ───────────────────────────────────────────────────────

func resolvePhysical(attacker, defender *MechState, rng *rand.Rand) {
	// Can only do physical attacks at range 1
	dist := HexDistance(attacker.Pos, defender.Pos)
	if dist != 1 {
		return
	}

	// Kick: damage = tonnage/5, target = piloting skill + move mods
	kickDmg := attacker.Tonnage / 5
	kickTarget := pilotingSkill - 2
	// Add attacker movement modifier
	switch attacker.LastMoveMode {
	case ModeWalk:
		kickTarget += 1
	case ModeRun:
		kickTarget += 2
	case ModeJump:
		kickTarget += 3
	}
	// Add target TMM (BMM: physical attacks include TMM)
	kickTarget += tmmFromHexesMoved(defender.LastHexMoved, defender.LastMoveMode)
	if kickTarget < 2 {
		kickTarget = 2
	}

	if roll2d6(rng) >= kickTarget {
		loc := LocLL
		if rng.IntN(2) == 1 {
			loc = LocRL
		}
		defender.applyDamage(loc, kickDmg, false, rng)

		// Kicked mech needs PSR
		if !defender.Prone {
			if !defender.rollPSR(0, rng) {
				defender.applyFall(rng)
			}
		}
	}
}
