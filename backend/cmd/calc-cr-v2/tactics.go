package main

import (
	"math"
	"math/rand/v2"
	"sort"
)

// ─── Tactical AI ────────────────────────────────────────────────────────────
//
// Two distinct decision modes:
//
// SECOND MOVER (initiative winner): Sees opponent position. Simple optimization
// — pick hex that maximizes score given known opponent position.
//
// FIRST MOVER (initiative loser): Doesn't know where opponent will end up.
// Uses 1-ply minimax: "For each hex I could go to, what's my opponent's best
// response? Pick the hex where opponent's best response still leaves me best off."
// This naturally produces counter-positioning — it avoids hexes where the opponent
// has strong responses and prefers hexes that constrain opponent's good options.

// ─── Scoring Components ─────────────────────────────────────────────────────

// PositionScore evaluates a specific (myHex, opHex) pairing.
// This is the leaf evaluation — given both positions are known.
func PositionScore(board *Board, me *MechState2, myHex ReachableHex,
	op *MechState2, opPos HexCoord, opFacing int, losCache ...*LOSCache) float64 {

	dist := HexDistance(myHex.Coord, opPos)

	// ─── LOS & Damage ───
	var lc *LOSCache
	if len(losCache) > 0 {
		lc = losCache[0]
	}
	los := CheckLOSCached(board, myHex.Coord, opPos, lc)
	var myDmg float64
	if los.CanSee {
		myDmg = expectedDamage(me, dist, los, myHex, opPos, opFacing)
	}

	opLOS := CheckLOSCached(board, opPos, myHex.Coord, lc)
	var opDmg float64
	if opLOS.CanSee {
		// Estimate opponent's damage — use their last move mode as approximation
		opRH := ReachableHex{Coord: opPos, Facing: opFacing, Mode: ModeWalk}
		opDmg = expectedDamage(op, dist, opLOS, opRH, myHex.Coord, myHex.Facing)
	}

	// ─── Arc Advantage ───
	// What arc am I hitting on them?
	arcOnThem := DetermineArc(opPos, opFacing, myHex.Coord)
	rearShotBonus := 0.0
	if arcOnThem == ArcRear {
		rearShotBonus = myDmg * 0.6 // rear armor ~40% of front, so damage is worth ~60% more
	}

	// What arc are they hitting on me?
	arcOnMe := DetermineArc(myHex.Coord, myHex.Facing, opPos)
	rearExposure := 0.0
	if arcOnMe == ArcRear {
		rearExposure = opDmg * 0.6 // they're hitting my thin rear
	}

	// ─── Cover ───
	myHexData := board.Get(myHex.Coord)
	coverValue := 0.0
	if myHexData != nil {
		if hasWoods, level := myHexData.HasTerrain(TerrainWoods); hasWoods {
			// Woods add to-hit modifier for anyone shooting at me
			// Each +1 to-hit ≈ reduces hit prob by ~15-20% ≈ 3-4 damage reduction
			coverValue = float64(level) * 3.5
		}
	}

	// ─── Elevation ───
	elevValue := 0.0
	if los.ElevationMod < 0 {
		elevValue = 2.5 // I'm higher: -1 to-hit for me, harder for them to close
	} else if los.ElevationMod > 0 {
		elevValue = -1.5
	}

	// ─── TMM ───
	tmm := tmmFromHexesMoved(myHex.HexesMoved, myHex.Mode)
	tmmValue := float64(tmm) * 2.5 // each TMM ≈ 2.5 damage avoided

	// ─── Heat Awareness ───
	// If we're running hot, prefer positions where we can stand next turn in cover
	heatPenalty := 0.0
	if me.Heat > 8 {
		// High heat — value cover more (we may need to cool next turn)
		if coverValue > 0 {
			coverValue *= 1.5
		}
		// Penalize positions that require running (more heat next turn)
		if myHex.Mode == ModeRun {
			heatPenalty = 3.0
		}
	}

	// ─── Weapon Arc Awareness ───
	// Can I fire but they can't? (partial LOS exploitation)
	losAsymmetry := 0.0
	if los.CanSee && !opLOS.CanSee {
		losAsymmetry = myDmg * 0.5 // huge advantage: I shoot, they can't
	} else if !los.CanSee && opLOS.CanSee {
		losAsymmetry = -opDmg * 0.5 // bad: they shoot, I can't
	}

	// ─── No LOS, No Damage: Approach ───
	if !los.CanSee && myDmg == 0 {
		// Can't see them — score by closing distance preferring cover
		approachScore := -float64(dist) * 2.0
		approachScore += coverValue // prefer covered approach routes
		return approachScore
	}

	// ─── Composite Score ───
	score := myDmg - opDmg*0.7 +
		rearShotBonus - rearExposure +
		coverValue + elevValue + tmmValue +
		losAsymmetry - heatPenalty

	// Out of range: prefer closing
	if myDmg == 0 && los.CanSee {
		score = -float64(dist) + coverValue + tmmValue
	}

	return score
}

// ─── Second Mover (knows opponent position) ─────────────────────────────────

// ChooseHexSecondMover picks the best hex knowing exactly where opponent is.
func ChooseHexSecondMover(board *Board, me *MechState2, options []ReachableHex,
	op *MechState2, opPos HexCoord, opFacing int) ReachableHex {

	bestScore := math.Inf(-1)
	bestIdx := 0
	lc := newLOSCache()

	for i, opt := range options {
		// Face toward opponent for best weapon arcs
		opt.Facing = bearingToFacing(opt.Coord, opPos)
		options[i] = opt

		score := PositionScore(board, me, opt, op, opPos, opFacing, lc)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	return options[bestIdx]
}

// ─── First Mover (minimax — opponent responds optimally) ────────────────────

// ChooseHexFirstMover uses 1-ply minimax:
// For each of my candidate hexes, assume opponent picks THEIR best response.
// Pick the hex where opponent's best response still leaves me best off.
//
// Complexity: O(myOptions × opOptions) ≈ 50×50 = 2,500 per decision. Fast.
func ChooseHexFirstMover(board *Board, me *MechState2, myOptions []ReachableHex,
	op *MechState2, opOptions []ReachableHex) ReachableHex {

	bestMyScore := math.Inf(-1)
	bestIdx := 0
	lc := newLOSCache()

	// Optimization: if too many options, sample top candidates
	myOpts := myOptions
	if len(myOpts) > 40 {
		myOpts = preselectCandidates(board, me, myOptions, op, 40, lc)
	}
	opOpts := opOptions
	if len(opOpts) > 40 {
		opOpts = preselectCandidates(board, op, opOptions, me, 40, lc)
	}

	for i, myHex := range myOpts {
		// Face toward opponent's current position (best guess)
		myHex.Facing = bearingToFacing(myHex.Coord, op.Pos)
		myOpts[i] = myHex

		// Opponent sees my position, picks their best response
		worstCaseForMe := math.Inf(1) // opponent minimizes my score

		for j, opHex := range opOpts {
			opHex.Facing = bearingToFacing(opHex.Coord, myHex.Coord)
			opOpts[j] = opHex

			// My score given both positions
			myScore := PositionScore(board, me, myHex, op, opHex.Coord, opHex.Facing, lc)
			// Opponent's score (they try to maximize this)
			opScore := PositionScore(board, op, opHex, me, myHex.Coord, myHex.Facing, lc)
			// Net score from my perspective: my advantage minus their advantage
			netScore := myScore - opScore*0.3

			if netScore < worstCaseForMe {
				worstCaseForMe = netScore
			}
		}

		// My score for this hex = worst case (opponent plays optimally)
		if worstCaseForMe > bestMyScore {
			bestMyScore = worstCaseForMe
			bestIdx = i
		}
	}

	return myOpts[bestIdx]
}

// preselectCandidates quickly scores options against opponent's current position
// and returns the top N candidates for deeper minimax evaluation.
func preselectCandidates(board *Board, me *MechState2, options []ReachableHex,
	op *MechState2, n int, lc *LOSCache) []ReachableHex {

	if len(options) <= n {
		return options
	}

	type scored struct {
		idx   int
		score float64
	}
	var scores []scored

	for i, opt := range options {
		opt.Facing = bearingToFacing(opt.Coord, op.Pos)
		s := PositionScore(board, me, opt, op, op.Pos, op.Facing, lc)
		scores = append(scores, scored{i, s})
	}

	// Sort by score descending
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	result := make([]ReachableHex, n)
	for i := 0; i < n; i++ {
		result[i] = options[scores[i].idx]
	}
	return result
}

// ─── Multi-Turn Approach Value ──────────────────────────────────────────────
// A mech closing through woods might take 2-3 turns to reach engagement range.
// Give credit for hexes that are on a "good approach" even if damage is 0 now.

// approachValue estimates the value of being at a hex for a mech that needs
// to close distance. Considers: distance to opponent's optimal range,
// cover along the way, future damage potential.
func approachValue(board *Board, me *MechState2, myHex ReachableHex,
	opPos HexCoord) float64 {

	dist := HexDistance(myHex.Coord, opPos)

	// Find my optimal engagement range (highest expected damage range)
	optRange := optimalRange(me)

	// How far am I from my optimal range?
	rangeDelta := absInt(dist - optRange)

	// Closer to optimal = better
	approachScore := -float64(rangeDelta) * 2.0

	// Am I in cover? (woods between me and opponent along approach)
	myHexData := board.Get(myHex.Coord)
	if myHexData != nil {
		if hasWoods, level := myHexData.HasTerrain(TerrainWoods); hasWoods {
			approachScore += float64(level) * 2.0 // covered approach
		}
	}

	// Can I get to optimal range next turn?
	turnsToOptimal := float64(rangeDelta) / float64(maxInt(me.WalkMP, 1))
	if turnsToOptimal <= 1 {
		approachScore += 5.0 // one more turn and I'm in the sweet spot
	} else if turnsToOptimal <= 2 {
		approachScore += 2.0
	}

	return approachScore
}

// optimalRange returns the precomputed optimal range stored on the mech.
func optimalRange(mech *MechState2) int {
	if mech.OptimalRange > 0 {
		return mech.OptimalRange
	}
	return 1
}

// calcOptimalRange computes the heat-neutral, damage-greedy optimal engagement range.
// At each hex distance, weapons are filtered to those with TN ≤ 8 (base TN 8 = gunnery 4 +
// attacker running +2 + TMM +2 for 4/6 defender running), then greedily selected by damage
// until heat-neutral. The range with highest total expected damage wins.
func calcOptimalRange(m *MechState) int {
	const baseTN = 8 // gunnery 4 + running +2 + TMM +2

	bestDmg := 0.0
	bestRange := 1

	for r := 1; r <= 30; r++ {
		// Collect eligible weapons at this range
		type candidate struct {
			damage int
			heat   int
			tn     int
		}
		var candidates []candidate

		for i := range m.Weapons {
			w := &m.Weapons[i]
			if w.Destroyed || r > w.LongRange {
				continue
			}
			if w.MinRange > 0 && r < w.MinRange {
				continue
			}

			rangeMod := 0
			switch {
			case r <= w.ShortRange:
				rangeMod = 0
			case r <= w.MedRange:
				rangeMod = 2
			default:
				rangeMod = 4
			}

			// Eligibility: range_mod + to_hit_mod ≤ 0
			if rangeMod+w.ToHitMod > 0 {
				continue
			}

			tn := baseTN + rangeMod + w.ToHitMod
			if tn > 12 {
				continue
			}
			candidates = append(candidates, candidate{w.Damage, w.Heat, tn})
		}

		// Greedy: sort by damage descending, pick until heat-neutral
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].damage > candidates[j].damage
		})

		totalDmg := 0.0
		totalHeat := 0
		dissipation := m.Dissipation
		if dissipation < 0 {
			dissipation = 0
		}

		for _, c := range candidates {
			if totalHeat+c.heat > dissipation {
				continue // skip — would exceed heat neutrality
			}
			totalHeat += c.heat
			totalDmg += float64(c.damage) * hitProbability(c.tn)
		}

		if totalDmg > bestDmg {
			bestDmg = totalDmg
			bestRange = r
		}
	}
	return bestRange
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ─── Expected Damage (detailed) ─────────────────────────────────────────────

// expectedDamage calculates expected damage accounting for arcs, LOS, and
// weapon locations (arm vs torso) relative to target position.
func expectedDamage(mech *MechState2, dist int, los LOSResult,
	myMove ReachableHex, targetPos HexCoord, targetFacing int) float64 {

	totalDmg := 0.0

	// Determine effective facing after optimal torso twist
	twist := BestTorsoTwist(myMove.Coord, myMove.Facing, targetPos)
	effFacing := ((myMove.Facing + twist) % 6 + 6) % 6

	// Which arc is the target in relative to my effective facing?
	arcToTarget := DetermineArc(myMove.Coord, effFacing, targetPos)

	// Target's defensive modifier from their arc to me
	arcOnTarget := DetermineArc(targetPos, targetFacing, myMove.Coord)
	_ = arcOnTarget // used for hit table selection in actual fire resolution

	for i := range mech.Weapons {
		w := &mech.Weapons[i]
		if w.Destroyed {
			continue
		}

		// Can this weapon fire at the target given arcs?
		if !canWeaponFireTactics(w, arcToTarget) {
			continue
		}

		if dist > w.LongRange || dist == 0 {
			continue
		}

		// Range modifier
		rangeMod := 0
		switch {
		case dist <= w.ShortRange:
			rangeMod = 0
		case dist <= w.MedRange:
			rangeMod = 2
		default:
			rangeMod = 4
		}

		// Min range
		minRangeMod := 0
		if w.MinRange > 0 && dist <= w.MinRange {
			minRangeMod = w.MinRange - dist + 1
		}

		// Attacker movement modifier
		atkMoveMod := 0
		switch myMove.Mode {
		case ModeWalk:
			atkMoveMod = 1
		case ModeRun:
			atkMoveMod = 2
		case ModeJump:
			atkMoveMod = 3
		}

		// Total target number
		target := mech.GunnerySkill + rangeMod + minRangeMod + atkMoveMod +
			los.WoodsMod + los.TargetCover + los.ElevationMod +
			w.ToHitMod

		if target > 12 {
			continue
		}
		if target < 2 {
			target = 2
		}

		pHit := hitProbability(target)
		totalDmg += pHit * float64(w.Damage)
	}
	return totalDmg
}

// canWeaponFireTactics checks if a weapon can fire at a target in the specified arc.
// Used by tactical AI for damage estimation. The actual fire resolution
// uses canWeaponFire in combat.go.
func canWeaponFireTactics(w *SimWeapon2, arcToTarget ArcType) bool {
	switch arcToTarget {
	case ArcFront:
		return true
	case ArcLeft, ArcRight:
		return w.Location == LocCT || w.Location == LocLT || w.Location == LocRT || w.Location == LocHD
	case ArcRear:
		return false
	}
	return true
}

// ─── Full Decision Function ─────────────────────────────────────────────────

// ChooseMovement is the top-level movement decision.
// Routes to first-mover or second-mover logic based on initiative.
func ChooseMovement(board *Board, me *MechState2, op *MechState2,
	isSecondMover bool, opPos HexCoord, opFacing int,
	opOptions []ReachableHex, rng *rand.Rand) ReachableHex {

	myOptions := collectAllMoveOptions(board, me)
	if len(myOptions) == 0 {
		return ReachableHex{Coord: me.Pos, Facing: me.Facing, Mode: ModeStand}
	}

	if isSecondMover {
		return ChooseHexSecondMover(board, me, myOptions, op, opPos, opFacing)
	}

	return ChooseHexFirstMover(board, me, myOptions, op, opOptions)
}

func collectAllMoveOptions(board *Board, mech *MechState2) []ReachableHex {
	var all []ReachableHex

	// Standing still
	all = append(all, ReachableHex{
		Coord: mech.Pos, Facing: mech.Facing, Mode: ModeStand,
		HexesMoved: 0, MoveHeat: 0,
	})

	// Walk
	if mech.WalkMP > 0 {
		all = append(all, ReachableHexes(board, mech.Pos, mech.Facing,
			mech.WalkMP, mech.RunMP, mech.JumpMP, ModeWalk)...)
	}

	// Run
	if mech.RunMP > 0 {
		all = append(all, ReachableHexes(board, mech.Pos, mech.Facing,
			mech.WalkMP, mech.RunMP, mech.JumpMP, ModeRun)...)
	}

	// Jump
	if mech.JumpMP > 0 {
		all = append(all, ReachableHexes(board, mech.Pos, mech.Facing,
			mech.WalkMP, mech.RunMP, mech.JumpMP, ModeJump)...)
	}

	return all
}

// ─── Torso Twist ────────────────────────────────────────────────────────────

// BestTorsoTwist returns the optimal torso twist (-1, 0, or +1 hexside)
// given the mech's position/facing and the target position.
func BestTorsoTwist(mechPos HexCoord, facing int, targetPos HexCoord) int {
	arc := DetermineArc(mechPos, facing, targetPos)
	if arc == ArcFront {
		return 0
	}

	// Try left twist
	leftFacing := ((facing - 1) % 6 + 6) % 6
	leftArc := DetermineArc(mechPos, leftFacing, targetPos)
	if leftArc == ArcFront {
		return -1
	}

	// Try right twist
	rightFacing := (facing + 1) % 6
	rightArc := DetermineArc(mechPos, rightFacing, targetPos)
	if rightArc == ArcFront {
		return 1
	}

	// Neither gets front arc — prefer one that avoids rear
	if leftArc != ArcRear {
		return -1
	}
	if rightArc != ArcRear {
		return 1
	}
	return 0
}

// ─── Hit Probability ────────────────────────────────────────────────────────

func hitProbability(target int) float64 {
	if target <= 2 {
		return 1.0
	}
	if target >= 13 {
		return 0.0
	}
	table := [13]float64{
		0, 0, 1.0, 35.0 / 36, 33.0 / 36, 30.0 / 36, 26.0 / 36,
		21.0 / 36, 15.0 / 36, 10.0 / 36, 6.0 / 36, 3.0 / 36, 1.0 / 36,
	}
	return table[target]
}

func tmmFromHexesMoved(hexes int, mode MoveMode) int {
	if mode == ModeJump {
		mp := hexes
		switch {
		case mp <= 2:
			return 1
		case mp <= 4:
			return 2
		case mp <= 6:
			return 3
		case mp <= 9:
			return 4
		default:
			return 5
		}
	}
	switch {
	case hexes <= 2:
		return 0
	case hexes <= 4:
		return 1
	case hexes <= 6:
		return 2
	case hexes <= 9:
		return 3
	case hexes <= 17:
		return 4
	case hexes <= 24:
		return 5
	default:
		return 6
	}
}

// Location constants defined in damage.go
