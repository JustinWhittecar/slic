package main

import "math"

// ─── Dice helpers ───────────────────────────────────────────────────────────

func roll1d6(rng randSource) int  { return rng.IntN(6) + 1 }
func roll2d6(rng randSource) int  { return roll1d6(rng) + roll1d6(rng) }

type randSource interface {
	IntN(n int) int
	Float64() float64
}

// ─── 2d6 probability table ─────────────────────────────────────────────────

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

func prob2d6Fail(threshold int) float64 {
	if threshold <= 2 {
		return 0
	}
	if threshold > 12 {
		return 1.0
	}
	return 1.0 - pHitTable[threshold]
}

// ─── TMM ────────────────────────────────────────────────────────────────────

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

// ─── Heat scale ─────────────────────────────────────────────────────────────

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

func heatCostEV(heat int, avgTurnDmg float64, ammoExpDmg float64, walkMP int, currentDmgCapability float64) float64 {
	cost := 0.0

	shutdownP := heatShutdownProb(heat)
	if shutdownP > 0 {
		cost += shutdownP * avgTurnDmg * 1.5
	}

	ammoExpP := heatAmmoExpProb(heat)
	if ammoExpP > 0 {
		cost += ammoExpP * ammoExpDmg
	}

	mpLoss := heatMPReduction(heat)
	if mpLoss > 0 && walkMP > 0 {
		reducedWalk := walkMP - mpLoss
		if reducedWalk < 0 {
			reducedWalk = 0
		}
		origTMM := tmmFromMP(int(math.Ceil(float64(walkMP) * 1.5)))
		newTMM := tmmFromMP(int(math.Ceil(float64(reducedWalk) * 1.5)))
		tmmLoss := origTMM - newTMM
		if tmmLoss > 0 {
			cost += float64(tmmLoss) * 0.15 * currentDmgCapability
		}
	}

	return cost
}
