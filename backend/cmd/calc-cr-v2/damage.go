package main

import (
	"math"
	"math/rand/v2"
	"strings"
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

// Rear hit table — roll 12 = Head for ALL columns (BMM p.33) [fix #1]
var rearHitTable = [11]int{
	LocCT, LocRA, LocRA, LocRL, LocRT, LocCT, LocLT, LocLL, LocLA, LocLA, LocHD,
}

// ─── IS table by tonnage ────────────────────────────────────────────────────

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
	bestTons := 20
	for t := range isTable {
		if t <= tons && t > bestTons {
			bestTons = t
		}
	}
	return isTable[bestTons]
}

// ─── MechState — full 2D-aware mech state ───────────────────────────────────

type MechState struct {
	// Identity
	DebugName string

	// Static
	Tonnage       int
	WalkMP        int
	RunMP         int
	JumpMP        int
	HeatSinkCount int
	Dissipation   int
	IsXL          bool
	IsClanXL      bool
	IsReinforced  bool
	IsComposite   bool
	TechBase      string

	// 2D position
	Pos    HexCoord
	Facing int // 0-5

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

	// Ammo pools
	Ammo map[string]int

	// Dynamic state
	Heat           int
	EngineHits     int
	GyroHits       int
	SensorHits     int
	CockpitHit     bool
	ArmActuatorHit [NumLoc]int
	LegActuatorHit int

	// IS exposure tracking
	ISExposed [NumLoc]bool

	// Computed
	OptimalRange int

	// Equipment
	HasTargetingComputer bool
	HasAMS               bool
	AMSAmmo              int
	IsLaserAMS           bool
	HasArtemisIV         bool
	HasArtemisV          bool
	HasApollo            bool
	AMSUsedThisTurn      bool

	// Heat state
	HeatPenalty       int // heat applied by enemy plasma weapons
	IsShutdown        bool
	ProneFromShutdown bool

	// PSR / falling state
	Prone            bool
	PilotDamage      int
	PilotUnconscious bool
	HipHit           [NumLoc]bool
	LegFootHits      [NumLoc]int
	NeedsPSRFromCrit bool

	// Movement tracking (set each turn)
	LastMoveMode MoveMode
	LastHexMoved int
	TorsoTwist   int // -1, 0, +1
}

func (m *MechState) effectiveWalkMP() int {
	// One leg destroyed = immobile (BMM)
	if m.IS[LocLL] <= 0 || m.IS[LocRL] <= 0 {
		return 0
	}
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

func (m *MechState) isDestroyed() bool {
	// [fix #6] Gyro destruction ≠ mech destroyed (BMM p.48 + errata)
	// [fix #12] Removed "3+ IS exposed = dead" — not in BMM
	// [fix #13] Both legs destroyed ≠ dead — just immobile + prone
	if m.PilotDamage >= 6 || m.CockpitHit || m.EngineHits >= 3 {
		return true
	}
	if m.IS[LocCT] <= 0 || m.IS[LocHD] <= 0 {
		return true
	}
	if m.IsXL && !m.IsClanXL {
		if m.IS[LocLT] <= 0 || m.IS[LocRT] <= 0 {
			return true
		}
	}
	if m.IsClanXL {
		if m.IS[LocLT] <= 0 && m.IS[LocRT] <= 0 {
			return true
		}
	}
	return false
}

// isForcedWithdrawal returns true if the mech should retreat (BMM p.81)
func (m *MechState) isForcedWithdrawal() bool {
	if m.isDestroyed() {
		return true
	}
	// Pilot damage 4+
	if m.PilotDamage >= 4 {
		return true
	}
	// 2 engine crits
	if m.EngineHits >= 2 {
		return true
	}
	// 1 gyro + 1 engine
	if m.GyroHits >= 1 && m.EngineHits >= 1 {
		return true
	}
	// Side torso destroyed (IS XL already dead, but standard engines survive)
	if m.IS[LocLT] <= 0 || m.IS[LocRT] <= 0 {
		return true
	}
	// IS exposed in 3+ locations (limbs count)
	exposedLimbs := 0
	exposedTorsos := 0
	for i := 0; i < NumLoc; i++ {
		if m.ISExposed[i] {
			switch i {
			case LocCT, LocLT, LocRT:
				exposedTorsos++
			case LocLA, LocRA, LocLL, LocRL:
				exposedLimbs++
			}
		}
	}
	if exposedLimbs >= 3 || exposedTorsos >= 2 {
		return true
	}
	// All weapons destroyed
	allDestroyed := true
	for _, w := range m.Weapons {
		if !w.Destroyed {
			allDestroyed = false
			break
		}
	}
	if allDestroyed && len(m.Weapons) > 0 {
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
		// [fix #7] propagate isRear through transfer
		m.transferDamage(loc, dmg, isRear, rng)
		return
	}

	remaining := dmg

	if isRear && (loc == LocCT || loc == LocLT || loc == LocRT) {
		rearIdx := loc - 1
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
		// [fix #2] Roll crits every time IS takes damage (BMM p.45)
		m.rollCrits(loc, rng)
		return
	}

	overflow := effectiveDmg - m.IS[loc]
	m.IS[loc] = 0
	m.ISExposed[loc] = true

	// Location destroyed — destroy all equipment
	for i := range m.Weapons {
		if m.Weapons[i].Location == loc {
			m.Weapons[i].Destroyed = true
		}
	}

	if overflow > 0 {
		// [fix #8] Composite: do NOT transfer overflow (BMM p.117)
		if m.IsComposite {
			return
		}
		// [fix #9] Reinforced: transfer overflow as-is, don't convert back to raw
		// (the old code did ceil(overflow / 0.5) which doubled transfer damage)
		// Standard & reinforced both just transfer overflow directly
		m.transferDamage(loc, overflow, isRear, rng)
	}
}

// [fix #7] transferDamage now propagates isRear flag
func (m *MechState) transferDamage(fromLoc int, dmg int, isRear bool, rng *rand.Rand) {
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
		return
	}
	m.applyDamage(toLoc, dmg, isRear, rng)
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
		switch {
		case critRoll >= 10:
			numCrits = 2
		case critRoll >= 8:
			numCrits = 1
		}
	} else {
		isLimb := loc == LocLA || loc == LocRA || loc == LocLL || loc == LocRL
		switch {
		case critRoll >= 12:
			// [fix #3] Roll 12 on arms/legs = limb blown off (BMM p.47-48)
			if isLimb {
				m.IS[loc] = 0
				m.ISExposed[loc] = true
				for i := range m.Weapons {
					if m.Weapons[i].Location == loc {
						m.Weapons[i].Destroyed = true
					}
				}
				return
			}
			numCrits = 3 // Torsos get 3 crits on roll 12
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
			m.LegActuatorHit += m.effectiveWalkMP()
			m.NeedsPSRFromCrit = true
		}
	case strings.Contains(slot, "upper leg") || strings.Contains(slot, "lower leg") || strings.Contains(slot, "foot"):
		m.LegActuatorHit++
		m.LegFootHits[loc]++
		m.NeedsPSRFromCrit = true
	case strings.Contains(slot, "ammo"):
		m.ammoExplosion(loc, slots[idx], rng)
	default:
		for i := range m.Weapons {
			if m.Weapons[i].Location == loc && !m.Weapons[i].Destroyed {
				if strings.Contains(strings.ToLower(m.Weapons[i].Name), strings.TrimSpace(slot)) ||
					strings.Contains(slot, strings.ToLower(m.Weapons[i].Name)) {
					m.Weapons[i].Destroyed = true
					break
				}
			}
		}
		for i := range m.Weapons {
			if m.Weapons[i].Location == loc && !m.Weapons[i].Destroyed {
				m.Weapons[i].Destroyed = true
				break
			}
		}
	}
}

func (m *MechState) ammoExplosion(loc int, slotName string, rng *rand.Rand) {
	ammoKey := parseAmmoSlotKey(slotName)

	// [fix #11] Gauss ammo is non-explosive; gauss WEAPON explodes, not ammo
	if strings.Contains(strings.ToLower(ammoKey), "gauss") {
		return
	}

	shots := m.Ammo[ammoKey]
	if shots <= 0 {
		return
	}

	// [fix #5] Per-slot ammo: divide by number of locations containing this ammo type (BMM p.47)
	locCount := 0
	for l := 0; l < NumLoc; l++ {
		for _, slot := range m.Slots[l] {
			if strings.Contains(strings.ToLower(slot), "ammo") && parseAmmoSlotKey(slot) == ammoKey {
				locCount++
				break
			}
		}
	}
	if locCount < 1 {
		locCount = 1
	}
	slotShots := shots / locCount
	if slotShots < 1 {
		slotShots = 1
	}
	m.Ammo[ammoKey] -= slotShots

	dmgPerShot := estimateAmmoDamage(ammoKey)
	totalDmg := slotShots * dmgPerShot

	// [fix #10] CASE II: apply 1 IS damage, roll crits with filtering, discard excess (BMM p.47 + errata)
	if m.HasCASEII[loc] {
		if m.IS[loc] > 1 {
			m.IS[loc]--
			m.ISExposed[loc] = true
			m.rollCritsCASEII(loc, rng)
		} else if m.IS[loc] == 1 {
			m.IS[loc] = 0
			m.ISExposed[loc] = true
			for i := range m.Weapons {
				if m.Weapons[i].Location == loc {
					m.Weapons[i].Destroyed = true
				}
			}
		}
		return
	}

	// [fix #4] CASE: apply explosion damage to IS normally; excess discarded (BMM p.118)
	if m.HasCASE[loc] {
		if m.IS[loc] > totalDmg {
			m.IS[loc] -= totalDmg
			m.ISExposed[loc] = true
			m.rollCrits(loc, rng)
		} else {
			m.IS[loc] = 0
			m.ISExposed[loc] = true
			for i := range m.Weapons {
				if m.Weapons[i].Location == loc {
					m.Weapons[i].Destroyed = true
				}
			}
		}
		// Excess damage discarded (not transferred)
		return
	}

	// No CASE: apply to IS, transfer excess
	if m.IS[loc] > totalDmg {
		m.IS[loc] -= totalDmg
		m.ISExposed[loc] = true
		m.rollCrits(loc, rng)
	} else {
		remaining := totalDmg - m.IS[loc]
		m.IS[loc] = 0
		m.ISExposed[loc] = true
		for i := range m.Weapons {
			if m.Weapons[i].Location == loc {
				m.Weapons[i].Destroyed = true
			}
		}
		if remaining > 0 {
			m.transferDamage(loc, remaining, false, rng)
		}
	}
}

// rollCritsCASEII rolls crits but each result gets a 2d6 filter; 8+ negates that crit [fix #10]
func (m *MechState) rollCritsCASEII(loc int, rng *rand.Rand) {
	critRoll := roll2d6(rng)
	if m.IsReinforced {
		critRoll--
	}

	numCrits := 0
	if loc == LocHD {
		switch {
		case critRoll >= 12:
			filterRoll := roll2d6(rng)
			if filterRoll < 8 {
				m.CockpitHit = true
			}
			return
		case critRoll >= 10:
			numCrits = 2
		case critRoll >= 8:
			numCrits = 1
		}
	} else {
		isLimb := loc == LocLA || loc == LocRA || loc == LocLL || loc == LocRL
		switch {
		case critRoll >= 12:
			if isLimb {
				filterRoll := roll2d6(rng)
				if filterRoll < 8 {
					m.IS[loc] = 0
					m.ISExposed[loc] = true
					for i := range m.Weapons {
						if m.Weapons[i].Location == loc {
							m.Weapons[i].Destroyed = true
						}
					}
				}
				return
			}
			numCrits = 3
		case critRoll >= 10:
			numCrits = 2
		case critRoll >= 8:
			numCrits = 1
		}
	}

	for i := 0; i < numCrits; i++ {
		filterRoll := roll2d6(rng)
		if filterRoll < 8 {
			m.applyCrit(loc, rng)
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

// ─── Clone mech state ───────────────────────────────────────────────────────

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

// ─── Ammo helpers ───────────────────────────────────────────────────────────

func estimateAmmoDamage(ammoKey string) int {
	k := strings.ToLower(ammoKey)
	switch {
	case strings.Contains(k, "ac/20"):
		return 20
	case strings.Contains(k, "ac/10"):
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

func mechAmmoExplosionDamage(m *MechState) float64 {
	totalDmg := 0.0
	for key, shots := range m.Ammo {
		if shots <= 0 {
			continue
		}
		k := strings.ToLower(key)
		if strings.Contains(k, "gauss") {
			continue
		}
		dmgPerShot := estimateAmmoDamage(key)
		locDmg := float64(shots * dmgPerShot)

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
			locDmg *= 0.1
		}
		totalDmg += locDmg
	}
	return totalDmg
}
