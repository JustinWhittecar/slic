package main

import (
	"encoding/json"
	"math/rand/v2"
	"strconv"
	"strings"
)

// ─── Replay data structures ─────────────────────────────────────────────────

type ReplayHex struct {
	Col       int    `json:"col"`
	Row       int    `json:"row"`
	Elevation int    `json:"elevation"`
	Terrain   string `json:"terrain,omitempty"`
}

type ReplayMechSnapshot struct {
	Name       string  `json:"name"`
	Col        int     `json:"col"`
	Row        int     `json:"row"`
	Facing     int     `json:"facing"`
	Twist      int     `json:"twist"`
	Heat       int     `json:"heat"`
	Armor      [8]int  `json:"armor"`
	RearArmor  [3]int  `json:"rearArmor"`
	IS         [8]int  `json:"is"`
	MaxIS      [8]int  `json:"maxIS"`
	Prone      bool    `json:"prone,omitempty"`
	Shutdown   bool    `json:"shutdown,omitempty"`
	Destroyed  bool    `json:"destroyed,omitempty"`
	EngineHits int     `json:"engineHits,omitempty"`
	GyroHits   int     `json:"gyroHits,omitempty"`
	PilotDmg   int     `json:"pilotDmg,omitempty"`
	WalkMP     int     `json:"walkMP"`
	RunMP      int     `json:"runMP"`
	JumpMP     int     `json:"jumpMP"`
	MoveMode   string  `json:"moveMode"`
	HexesMoved int     `json:"hexesMoved"`
	ForcedWD   bool    `json:"forcedWithdrawal,omitempty"`
	CRScore    float64 `json:"crScore,omitempty"`
}

type ReplayWeaponFire struct {
	Weapon   string `json:"weapon"`
	Actor    string `json:"actor"`
	Target   int    `json:"target"`
	Roll     int    `json:"roll,omitempty"`
	Hit      bool   `json:"hit"`
	Damage   int    `json:"damage"`
	Location string `json:"location,omitempty"`
	Crit     string `json:"crit,omitempty"`
}

type ReplayEvent struct {
	Type    string `json:"type"` // "move", "fire", "physical", "heat", "psr", "crit", "destroyed", "fall", "info"
	Actor   string `json:"actor"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type ReplayTurn struct {
	Turn     int                  `json:"turn"`
	Attacker ReplayMechSnapshot   `json:"attacker"`
	Defender ReplayMechSnapshot   `json:"defender"`
	Events   []ReplayEvent        `json:"events"`
	Weapons  []ReplayWeaponFire   `json:"weapons,omitempty"`
}

type ReplayData struct {
	AttackerName string       `json:"attackerName"`
	DefenderName string       `json:"defenderName"`
	BoardWidth   int          `json:"boardWidth"`
	BoardHeight  int          `json:"boardHeight"`
	Hexes        []ReplayHex  `json:"hexes"`
	Turns        []ReplayTurn `json:"turns"`
	Result       string       `json:"result"`
}

// ─── Snapshot helpers ───────────────────────────────────────────────────────

func moveModeStr(m MoveMode) string {
	switch m {
	case ModeStand:
		return "stand"
	case ModeWalk:
		return "walk"
	case ModeRun:
		return "run"
	case ModeJump:
		return "jump"
	default:
		return "none"
	}
}

func snapshotMech(m *MechState) ReplayMechSnapshot {
	return ReplayMechSnapshot{
		Name:       m.DebugName,
		Col:        m.Pos.Col,
		Row:        m.Pos.Row,
		Facing:     m.Facing,
		Twist:      m.TorsoTwist,
		Heat:       m.Heat,
		Armor:      m.Armor,
		RearArmor:  m.RearArmor,
		IS:         m.IS,
		MaxIS:      m.MaxIS,
		Prone:      m.Prone,
		Shutdown:   m.IsShutdown,
		Destroyed:  m.isDestroyed(),
		EngineHits: m.EngineHits,
		GyroHits:   m.GyroHits,
		PilotDmg:   m.PilotDamage,
		WalkMP:     m.effectiveWalkMP(),
		RunMP:      m.effectiveRunMP(),
		JumpMP:     m.JumpMP,
		MoveMode:   moveModeStr(m.LastMoveMode),
		HexesMoved: m.LastHexMoved,
		ForcedWD:   m.isForcedWithdrawal(),
	}
}

func boardToReplayHexes(board *Board) []ReplayHex {
	var hexes []ReplayHex
	for _, h := range board.Hexes {
		terrain := ""
		var parts []string
		for _, f := range h.Terrain {
			switch f.Type {
			case TerrainWoods:
				if f.Level >= 2 {
					parts = append(parts, "heavy_woods")
				} else {
					parts = append(parts, "light_woods")
				}
			case TerrainWater:
				parts = append(parts, "water")
			case TerrainRough:
				parts = append(parts, "rough")
			case TerrainBuilding:
				parts = append(parts, "building")
			case TerrainPavement:
				parts = append(parts, "pavement")
			case TerrainRoad:
				parts = append(parts, "road")
			}
		}
		if len(parts) > 0 {
			terrain = strings.Join(parts, ",")
		}
		hexes = append(hexes, ReplayHex{
			Col: h.Coord.Col, Row: h.Coord.Row,
			Elevation: h.Elevation, Terrain: terrain,
		})
	}
	return hexes
}

// ─── Replay-enabled simulation ──────────────────────────────────────────────

func simulateReplay(board *Board, attackerTemplate, defenderTemplate *MechState, rng *rand.Rand) *ReplayData {
	attacker := cloneMech(attackerTemplate)
	defender := cloneMech(defenderTemplate)

	replay := &ReplayData{
		AttackerName: attacker.DebugName,
		DefenderName: defender.DebugName,
		BoardWidth:   board.Width,
		BoardHeight:  board.Height,
		Hexes:        boardToReplayHexes(board),
	}

	attacker.Pos = HexCoord{Col: board.Width/2 + 1, Row: 2}
	attacker.Facing = 3
	defender.Pos = HexCoord{Col: board.Width/2 + 1, Row: board.Height - 1}
	defender.Facing = 0

	for turn := 1; turn <= maxTurns; turn++ {
		turnData := ReplayTurn{Turn: turn}
		var events []ReplayEvent

		if attacker.isDestroyed() || defender.isDestroyed() {
			replay.Result = "ended"
			break
		}
		if defender.isForcedWithdrawal() {
			events = append(events, ReplayEvent{Type: "info", Actor: "defender", Message: "Forced withdrawal"})
			turnData.Events = events
			turnData.Attacker = snapshotMech(attacker)
			turnData.Defender = snapshotMech(defender)
			replay.Turns = append(replay.Turns, turnData)
			replay.Result = "forced_withdrawal"
			break
		}

		// Shutdown recovery
		if attacker.IsShutdown {
			attacker.IsShutdown = false
			events = append(events, ReplayEvent{Type: "heat", Actor: "attacker", Message: "Recovering from shutdown"})
			if !attacker.Prone {
				attacker.applyFall(rng)
				events = append(events, ReplayEvent{Type: "fall", Actor: "attacker", Message: "Falls from shutdown"})
			}
			if attacker.isDestroyed() {
				turnData.Events = events
				turnData.Attacker = snapshotMech(attacker)
				turnData.Defender = snapshotMech(defender)
				replay.Turns = append(replay.Turns, turnData)
				replay.Result = "attacker_destroyed"
				break
			}
			attacker.Heat += attacker.HeatPenalty; attacker.HeatPenalty = 0
			attacker.Heat -= attacker.Dissipation
			if attacker.Heat < 0 { attacker.Heat = 0 }
			defender.Heat += defender.HeatPenalty; defender.HeatPenalty = 0
			defender.Heat -= defender.Dissipation
			if defender.Heat < 0 { defender.Heat = 0 }
			turnData.Events = events
			turnData.Attacker = snapshotMech(attacker)
			turnData.Defender = snapshotMech(defender)
			replay.Turns = append(replay.Turns, turnData)
			continue
		}

		// Standing from prone
		if attacker.Prone {
			attacker.Heat += 1
			if attacker.rollPSRForStanding(rng) {
				attacker.Prone = false
				events = append(events, ReplayEvent{Type: "psr", Actor: "attacker", Message: "Stands from prone"})
			} else {
				attacker.applyFall(rng)
				events = append(events, ReplayEvent{Type: "psr", Actor: "attacker", Message: "Failed to stand, falls again"})
			}
		}
		if defender.Prone {
			defender.Heat += 1
			if defender.rollPSRForStanding(rng) {
				defender.Prone = false
				events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Stands from prone"})
			} else {
				defender.applyFall(rng)
				events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Failed to stand, falls again"})
			}
		}

		// Initiative
		atkInit := rng.IntN(6) + 1
		defInit := rng.IntN(6) + 1
		atkMovesFirst := atkInit < defInit
		if atkInit == defInit {
			atkMovesFirst = rng.IntN(2) == 0
		}
		initWinner := "attacker"
		if !atkMovesFirst {
			initWinner = "defender"
		}
		events = append(events, ReplayEvent{
			Type: "info", Actor: "system",
			Message: "Initiative: " + initWinner + " moves second (reacts)",
			Detail:  "Attacker: " + itoa(atkInit) + " Defender: " + itoa(defInit),
		})

		// Movement (same logic as main sim)
		atkWalk, atkRun := 0, 0
		if !attacker.Prone {
			atkWalk = attacker.effectiveWalkMP()
			atkRun = attacker.effectiveRunMP()
		}
		defWalk, defRun := 0, 0
		if !defender.Prone {
			defWalk = defender.effectiveWalkMP()
			defRun = defender.effectiveRunMP()
		}

		atkM2 := &MechState2{Pos: attacker.Pos, Facing: attacker.Facing, WalkMP: atkWalk, RunMP: atkRun, JumpMP: attacker.JumpMP, Tonnage: attacker.Tonnage, GunnerySkill: gunnerySkill, Heat: attacker.Heat}
		defM2 := &MechState2{Pos: defender.Pos, Facing: defender.Facing, WalkMP: defWalk, RunMP: defRun, JumpMP: defender.JumpMP, Tonnage: defender.Tonnage, GunnerySkill: gunnerySkill, Heat: defender.Heat}
		for _, w := range attacker.Weapons {
			if !w.Destroyed && !w.Jammed {
				atkM2.Weapons = append(atkM2.Weapons, SimWeapon2{Name: w.Name, Damage: w.Damage, Heat: w.Heat, MinRange: w.MinRange, ShortRange: w.ShortRange, MedRange: w.MedRange, LongRange: w.LongRange, Location: w.Location, ToHitMod: w.ToHitMod})
			}
		}
		for _, w := range defender.Weapons {
			if !w.Destroyed && !w.Jammed {
				defM2.Weapons = append(defM2.Weapons, SimWeapon2{Name: w.Name, Damage: w.Damage, Heat: w.Heat, MinRange: w.MinRange, ShortRange: w.ShortRange, MedRange: w.MedRange, LongRange: w.LongRange, Location: w.Location, ToHitMod: w.ToHitMod})
			}
		}

		defOptions := collectAllMoveOptions(board, defM2)
		atkOptions := collectAllMoveOptions(board, atkM2)

		var atkChoice, defChoice ReachableHex
		if atkMovesFirst {
			atkChoice = ChooseMovement(board, atkM2, defM2, false, defM2.Pos, defM2.Facing, defOptions, rng, atkOptions)
			defChoice = ChooseMovement(board, defM2, atkM2, true, atkChoice.Coord, atkChoice.Facing, atkOptions, rng, defOptions)
		} else {
			defChoice = ChooseMovement(board, defM2, atkM2, false, atkM2.Pos, atkM2.Facing, atkOptions, rng, defOptions)
			atkChoice = ChooseMovement(board, atkM2, defM2, true, defChoice.Coord, defChoice.Facing, defOptions, rng, atkOptions)
		}

		attacker.Pos = atkChoice.Coord
		attacker.Facing = atkChoice.Facing
		attacker.LastMoveMode = atkChoice.Mode
		attacker.LastHexMoved = atkChoice.HexesMoved
		attacker.Heat += atkChoice.MoveHeat

		defender.Pos = defChoice.Coord
		defender.Facing = defChoice.Facing
		defender.LastMoveMode = defChoice.Mode
		defender.LastHexMoved = defChoice.HexesMoved
		defender.Heat += defChoice.MoveHeat

		events = append(events, ReplayEvent{
			Type: "move", Actor: "attacker",
			Message: "Moves to (" + itoa(attacker.Pos.Col) + "," + itoa(attacker.Pos.Row) + ") " + moveModeStr(atkChoice.Mode),
			Detail:  itoa(atkChoice.HexesMoved) + " hexes, +" + itoa(atkChoice.MoveHeat) + " heat",
		})
		events = append(events, ReplayEvent{
			Type: "move", Actor: "defender",
			Message: "Moves to (" + itoa(defender.Pos.Col) + "," + itoa(defender.Pos.Row) + ") " + moveModeStr(defChoice.Mode),
			Detail:  itoa(defChoice.HexesMoved) + " hexes, +" + itoa(defChoice.MoveHeat) + " heat",
		})

		// Torso twist
		attacker.TorsoTwist = BestTorsoTwist(attacker.Pos, attacker.Facing, defender.Pos)
		defender.TorsoTwist = BestTorsoTwist(defender.Pos, defender.Facing, attacker.Pos)

		defender.AMSUsedThisTurn = false

		dist := HexDistance(attacker.Pos, defender.Pos)
		los := CheckLOS(board, attacker.Pos, defender.Pos)

		if los.CanSee && dist > 0 {
			defEffFacing := ((defender.Facing + defender.TorsoTwist) % 6 + 6) % 6
			arcToDefender := DetermineArc(defender.Pos, defEffFacing, attacker.Pos)
			isRear := arcToDefender == ArcRear

			defTMM := tmmFromHexesMoved(defChoice.HexesMoved, defChoice.Mode)
			heatThisMod := heatToHitMod(attacker.Heat)
			baseTarget := gunnerySkill + attacker.SensorHits*2 + heatThisMod

			switch atkChoice.Mode {
			case ModeWalk:
				baseTarget += 1
			case ModeRun:
				baseTarget += 2
			case ModeJump:
				baseTarget += 3
			}
			baseTarget += defTMM
			baseTarget += los.WoodsMod
			baseTarget += los.TargetCover
			baseTarget += los.ElevationMod

			if attacker.Prone { baseTarget += 2 }
			if defender.Prone {
				if dist <= 1 { baseTarget += 1 } else { baseTarget -= 2 }
			}

			arcStr := "front"
			if isRear { arcStr = "REAR" }
			events = append(events, ReplayEvent{
				Type: "info", Actor: "attacker",
				Message: "Firing at range " + itoa(dist) + " (" + arcStr + ") base TN " + itoa(baseTarget),
				Detail:  "TMM:" + itoa(defTMM) + " woods:" + itoa(los.WoodsMod) + " heat:" + itoa(heatThisMod),
			})

			firingWeapons, weaponHeatTotal := selectWeaponsEV(attacker, board, defender, dist, baseTarget)
			attacker.Heat += weaponHeatTotal

			totalDmgDealt := 0
			for _, wi := range firingWeapons {
				w := &attacker.Weapons[wi]
				if w.AmmoKey != "" {
					if attacker.Ammo[w.AmmoKey] <= 0 { continue }
					attacker.Ammo[w.AmmoKey]--
				}

				target := baseTarget + w.ToHitMod + attacker.ArmActuatorHit[w.Location]
				rm := rangeModifier(w, dist)
				if rm < 0 { continue }
				target += rm
				if w.MinRange > 0 && dist <= w.MinRange {
					target += w.MinRange - dist + 1
				}

				dmg := resolveWeaponFire2D(w, target, isRear, attacker, defender, rng)
				totalDmgDealt += dmg

				hitStr := "MISS"
				if dmg > 0 { hitStr = itoa(dmg) + " dmg" }
				events = append(events, ReplayEvent{
					Type: "fire", Actor: "attacker",
					Message: w.Name + " (TN " + itoa(target) + "): " + hitStr,
				})

				if defender.isDestroyed() {
					events = append(events, ReplayEvent{Type: "destroyed", Actor: "defender", Message: "DESTROYED"})
					turnData.Events = events
					turnData.Attacker = snapshotMech(attacker)
					turnData.Defender = snapshotMech(defender)
					replay.Turns = append(replay.Turns, turnData)
					replay.Result = "defender_destroyed_turn_" + itoa(turn)
					return replay
				}
			}

			if totalDmgDealt >= 20 && !defender.Prone {
				psrOk := defender.rollPSR(1, rng)
				if !psrOk {
					defender.applyFall(rng)
					events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Failed 20+ dmg PSR, falls!"})
				} else {
					events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Passed 20+ dmg PSR"})
				}
			}

			if defender.NeedsPSRFromCrit && !defender.isDestroyed() {
				defender.NeedsPSRFromCrit = false
				if !defender.rollPSR(0, rng) {
					defender.applyFall(rng)
					events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Failed crit PSR, falls!"})
				}
			}
		} else if dist == 0 {
			events = append(events, ReplayEvent{Type: "info", Actor: "system", Message: "Same hex - no fire"})
		} else {
			events = append(events, ReplayEvent{Type: "info", Actor: "system", Message: "No LOS"})
		}

		// Physical
		if dist == 1 {
			prevIS := defender.IS
			resolvePhysical(attacker, defender, rng)
			kicked := false
			for i := 0; i < NumLoc; i++ {
				if defender.IS[i] < prevIS[i] { kicked = true; break }
			}
			if kicked {
				events = append(events, ReplayEvent{Type: "physical", Actor: "attacker", Message: "Kick!"})
			}
		}

		// Heat dissipation (including plasma heat from enemy)
		attacker.Heat += attacker.HeatPenalty; attacker.HeatPenalty = 0
		attacker.Heat -= attacker.Dissipation
		if attacker.Heat < 0 { attacker.Heat = 0 }
		defender.Heat += defender.HeatPenalty; defender.HeatPenalty = 0
		defender.Heat -= defender.Dissipation
		if defender.Heat < 0 { defender.Heat = 0 }

		// Shutdown check
		shutdownP := heatShutdownProb(attacker.Heat)
		if shutdownP >= 1.0 || (shutdownP > 0 && rng.Float64() < shutdownP) {
			attacker.IsShutdown = true
			events = append(events, ReplayEvent{Type: "heat", Actor: "attacker", Message: "SHUTDOWN at heat " + itoa(attacker.Heat)})
		}

		ammoExpP := heatAmmoExpProb(attacker.Heat)
		if ammoExpP > 0 && rng.Float64() < ammoExpP {
			events = append(events, ReplayEvent{Type: "heat", Actor: "attacker", Message: "Ammo explosion from heat!"})
			type ammoBin struct { key string; loc int }
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
			}
		}

		turnData.Events = events
		turnData.Attacker = snapshotMech(attacker)
		turnData.Defender = snapshotMech(defender)
		replay.Turns = append(replay.Turns, turnData)

		if attacker.isDestroyed() {
			replay.Result = "attacker_destroyed"
			break
		}
		if defender.isDestroyed() {
			replay.Result = "defender_destroyed_turn_" + itoa(turn)
			break
		}
	}

	if replay.Result == "" {
		replay.Result = "timeout_" + itoa(maxTurns)
	}
	return replay
}

// ─── Duel replay (mutual combat) ────────────────────────────────────────────

// fireWeaponsReplay handles one side firing at the other, returning events, total damage, and whether target was destroyed.
func fireWeaponsReplay(shooter, target *MechState, board *Board, shooterChoice, targetChoice ReachableHex, actorName string, events []ReplayEvent, rng *rand.Rand) ([]ReplayEvent, int, bool) {
	dist := HexDistance(shooter.Pos, target.Pos)
	los := CheckLOS(board, shooter.Pos, target.Pos)

	if !los.CanSee || dist <= 0 {
		return events, 0, false
	}

	targetEffFacing := ((target.Facing + target.TorsoTwist) % 6 + 6) % 6
	arcToTarget := DetermineArc(target.Pos, targetEffFacing, shooter.Pos)
	isRear := arcToTarget == ArcRear

	targetTMM := tmmFromHexesMoved(targetChoice.HexesMoved, targetChoice.Mode)
	heatMod := heatToHitMod(shooter.Heat)
	baseTarget := gunnerySkill + shooter.SensorHits*2 + heatMod

	switch shooterChoice.Mode {
	case ModeWalk:
		baseTarget += 1
	case ModeRun:
		baseTarget += 2
	case ModeJump:
		baseTarget += 3
	}
	baseTarget += targetTMM
	baseTarget += los.WoodsMod
	baseTarget += los.TargetCover
	baseTarget += los.ElevationMod

	if shooter.Prone {
		baseTarget += 2
	}
	if target.Prone {
		if dist <= 1 {
			baseTarget += 1
		} else {
			baseTarget -= 2
		}
	}

	arcStr := "front"
	if isRear {
		arcStr = "REAR"
	}
	events = append(events, ReplayEvent{
		Type: "info", Actor: actorName,
		Message: "Firing at range " + itoa(dist) + " (" + arcStr + ") base TN " + itoa(baseTarget),
		Detail:  "TMM:" + itoa(targetTMM) + " woods:" + itoa(los.WoodsMod) + " heat:" + itoa(heatMod),
	})

	firingWeapons, weaponHeatTotal := selectWeaponsEV(shooter, board, target, dist, baseTarget)
	shooter.Heat += weaponHeatTotal

	totalDmg := 0
	for _, wi := range firingWeapons {
		w := &shooter.Weapons[wi]
		if w.AmmoKey != "" {
			if shooter.Ammo[w.AmmoKey] <= 0 {
				continue
			}
			shooter.Ammo[w.AmmoKey]--
		}

		tn := baseTarget + w.ToHitMod + shooter.ArmActuatorHit[w.Location]
		rm := rangeModifier(w, dist)
		if rm < 0 {
			continue
		}
		tn += rm
		if w.MinRange > 0 && dist <= w.MinRange {
			tn += w.MinRange - dist + 1
		}

		dmg := resolveWeaponFire2D(w, tn, isRear, shooter, target, rng)
		totalDmg += dmg

		hitStr := "MISS"
		if dmg > 0 {
			hitStr = itoa(dmg) + " dmg"
		}
		events = append(events, ReplayEvent{
			Type: "fire", Actor: actorName,
			Message: w.Name + " (TN " + itoa(tn) + "): " + hitStr,
		})

		if target.isDestroyed() {
			targetName := "defender"
			if actorName == "defender" {
				targetName = "attacker"
			}
			events = append(events, ReplayEvent{Type: "destroyed", Actor: targetName, Message: "DESTROYED"})
			return events, totalDmg, true
		}
	}

	return events, totalDmg, false
}

func simulateDuelReplay(board *Board, attackerTemplate, defenderTemplate *MechState, rng *rand.Rand) *ReplayData {
	attacker := cloneMech(attackerTemplate)
	defender := cloneMech(defenderTemplate)

	replay := &ReplayData{
		AttackerName: attacker.DebugName,
		DefenderName: defender.DebugName,
		BoardWidth:   board.Width,
		BoardHeight:  board.Height,
		Hexes:        boardToReplayHexes(board),
	}

	attacker.Pos = HexCoord{Col: board.Width/2 + 1, Row: 2}
	attacker.Facing = 3
	defender.Pos = HexCoord{Col: board.Width/2 + 1, Row: board.Height - 1}
	defender.Facing = 0

	for turn := 1; turn <= maxTurns; turn++ {
		turnData := ReplayTurn{Turn: turn}
		var events []ReplayEvent

		if attacker.isDestroyed() || defender.isDestroyed() {
			replay.Result = "ended"
			break
		}
		if attacker.isForcedWithdrawal() {
			events = append(events, ReplayEvent{Type: "info", Actor: "attacker", Message: "Forced withdrawal"})
			turnData.Events = events
			turnData.Attacker = snapshotMech(attacker)
			turnData.Defender = snapshotMech(defender)
			replay.Turns = append(replay.Turns, turnData)
			replay.Result = "attacker_forced_withdrawal"
			break
		}
		if defender.isForcedWithdrawal() {
			events = append(events, ReplayEvent{Type: "info", Actor: "defender", Message: "Forced withdrawal"})
			turnData.Events = events
			turnData.Attacker = snapshotMech(attacker)
			turnData.Defender = snapshotMech(defender)
			replay.Turns = append(replay.Turns, turnData)
			replay.Result = "defender_forced_withdrawal"
			break
		}

		// Shutdown recovery — attacker
		if attacker.IsShutdown {
			attacker.IsShutdown = false
			events = append(events, ReplayEvent{Type: "heat", Actor: "attacker", Message: "Recovering from shutdown"})
			if !attacker.Prone {
				attacker.applyFall(rng)
				events = append(events, ReplayEvent{Type: "fall", Actor: "attacker", Message: "Falls from shutdown"})
			}
		}
		// Shutdown recovery — defender
		if defender.IsShutdown {
			defender.IsShutdown = false
			events = append(events, ReplayEvent{Type: "heat", Actor: "defender", Message: "Recovering from shutdown"})
			if !defender.Prone {
				defender.applyFall(rng)
				events = append(events, ReplayEvent{Type: "fall", Actor: "defender", Message: "Falls from shutdown"})
			}
		}

		// Check deaths from shutdown falls
		if attacker.isDestroyed() || defender.isDestroyed() {
			turnData.Events = events
			turnData.Attacker = snapshotMech(attacker)
			turnData.Defender = snapshotMech(defender)
			replay.Turns = append(replay.Turns, turnData)
			if attacker.isDestroyed() && defender.isDestroyed() {
				replay.Result = "mutual_destruction"
			} else if attacker.isDestroyed() {
				replay.Result = "attacker_destroyed"
			} else {
				replay.Result = "defender_destroyed_turn_" + itoa(turn)
			}
			break
		}

		// Standing from prone
		if attacker.Prone {
			attacker.Heat += 1
			if attacker.rollPSRForStanding(rng) {
				attacker.Prone = false
				events = append(events, ReplayEvent{Type: "psr", Actor: "attacker", Message: "Stands from prone"})
			} else {
				attacker.applyFall(rng)
				events = append(events, ReplayEvent{Type: "psr", Actor: "attacker", Message: "Failed to stand, falls again"})
			}
		}
		if defender.Prone {
			defender.Heat += 1
			if defender.rollPSRForStanding(rng) {
				defender.Prone = false
				events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Stands from prone"})
			} else {
				defender.applyFall(rng)
				events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Failed to stand, falls again"})
			}
		}

		// Initiative
		atkInit := rng.IntN(6) + 1
		defInit := rng.IntN(6) + 1
		atkMovesFirst := atkInit < defInit
		if atkInit == defInit {
			atkMovesFirst = rng.IntN(2) == 0
		}
		initWinner := "attacker"
		if !atkMovesFirst {
			initWinner = "defender"
		}
		events = append(events, ReplayEvent{
			Type: "info", Actor: "system",
			Message: "Initiative: " + initWinner + " moves second (reacts)",
			Detail:  "Attacker: " + itoa(atkInit) + " Defender: " + itoa(defInit),
		})

		// Movement
		atkWalk, atkRun := 0, 0
		if !attacker.Prone {
			atkWalk = attacker.effectiveWalkMP()
			atkRun = attacker.effectiveRunMP()
		}
		defWalk, defRun := 0, 0
		if !defender.Prone {
			defWalk = defender.effectiveWalkMP()
			defRun = defender.effectiveRunMP()
		}

		atkM2 := &MechState2{Pos: attacker.Pos, Facing: attacker.Facing, WalkMP: atkWalk, RunMP: atkRun, JumpMP: attacker.JumpMP, Tonnage: attacker.Tonnage, GunnerySkill: gunnerySkill, Heat: attacker.Heat}
		defM2 := &MechState2{Pos: defender.Pos, Facing: defender.Facing, WalkMP: defWalk, RunMP: defRun, JumpMP: defender.JumpMP, Tonnage: defender.Tonnage, GunnerySkill: gunnerySkill, Heat: defender.Heat}
		for _, w := range attacker.Weapons {
			if !w.Destroyed && !w.Jammed {
				atkM2.Weapons = append(atkM2.Weapons, SimWeapon2{Name: w.Name, Damage: w.Damage, Heat: w.Heat, MinRange: w.MinRange, ShortRange: w.ShortRange, MedRange: w.MedRange, LongRange: w.LongRange, Location: w.Location, ToHitMod: w.ToHitMod})
			}
		}
		for _, w := range defender.Weapons {
			if !w.Destroyed && !w.Jammed {
				defM2.Weapons = append(defM2.Weapons, SimWeapon2{Name: w.Name, Damage: w.Damage, Heat: w.Heat, MinRange: w.MinRange, ShortRange: w.ShortRange, MedRange: w.MedRange, LongRange: w.LongRange, Location: w.Location, ToHitMod: w.ToHitMod})
			}
		}

		defOptions := collectAllMoveOptions(board, defM2)
		atkOptions := collectAllMoveOptions(board, atkM2)

		var atkChoice, defChoice ReachableHex
		if atkMovesFirst {
			atkChoice = ChooseMovement(board, atkM2, defM2, false, defM2.Pos, defM2.Facing, defOptions, rng, atkOptions)
			defChoice = ChooseMovement(board, defM2, atkM2, true, atkChoice.Coord, atkChoice.Facing, atkOptions, rng, defOptions)
		} else {
			defChoice = ChooseMovement(board, defM2, atkM2, false, atkM2.Pos, atkM2.Facing, atkOptions, rng, defOptions)
			atkChoice = ChooseMovement(board, atkM2, defM2, true, defChoice.Coord, defChoice.Facing, defOptions, rng, atkOptions)
		}

		attacker.Pos = atkChoice.Coord
		attacker.Facing = atkChoice.Facing
		attacker.LastMoveMode = atkChoice.Mode
		attacker.LastHexMoved = atkChoice.HexesMoved
		attacker.Heat += atkChoice.MoveHeat

		defender.Pos = defChoice.Coord
		defender.Facing = defChoice.Facing
		defender.LastMoveMode = defChoice.Mode
		defender.LastHexMoved = defChoice.HexesMoved
		defender.Heat += defChoice.MoveHeat

		events = append(events, ReplayEvent{
			Type: "move", Actor: "attacker",
			Message: "Moves to (" + itoa(attacker.Pos.Col) + "," + itoa(attacker.Pos.Row) + ") " + moveModeStr(atkChoice.Mode),
			Detail:  itoa(atkChoice.HexesMoved) + " hexes, +" + itoa(atkChoice.MoveHeat) + " heat",
		})
		events = append(events, ReplayEvent{
			Type: "move", Actor: "defender",
			Message: "Moves to (" + itoa(defender.Pos.Col) + "," + itoa(defender.Pos.Row) + ") " + moveModeStr(defChoice.Mode),
			Detail:  itoa(defChoice.HexesMoved) + " hexes, +" + itoa(defChoice.MoveHeat) + " heat",
		})

		// Torso twist
		attacker.TorsoTwist = BestTorsoTwist(attacker.Pos, attacker.Facing, defender.Pos)
		defender.TorsoTwist = BestTorsoTwist(defender.Pos, defender.Facing, attacker.Pos)

		defender.AMSUsedThisTurn = false
		attacker.AMSUsedThisTurn = false

		dist := HexDistance(attacker.Pos, defender.Pos)

		// --- Weapon fire: attacker fires, then defender fires ---
		var atkDmg, defDmg int
		destroyed := false

		if dist > 0 {
			events, atkDmg, destroyed = fireWeaponsReplay(attacker, defender, board, atkChoice, defChoice, "attacker", events, rng)
			if destroyed {
				turnData.Events = events
				turnData.Attacker = snapshotMech(attacker)
				turnData.Defender = snapshotMech(defender)
				replay.Turns = append(replay.Turns, turnData)
				replay.Result = "defender_destroyed_turn_" + itoa(turn)
				return replay
			}

			events, defDmg, destroyed = fireWeaponsReplay(defender, attacker, board, defChoice, atkChoice, "defender", events, rng)
			if destroyed {
				turnData.Events = events
				turnData.Attacker = snapshotMech(attacker)
				turnData.Defender = snapshotMech(defender)
				replay.Turns = append(replay.Turns, turnData)
				replay.Result = "attacker_destroyed"
				return replay
			}
		} else {
			events = append(events, ReplayEvent{Type: "info", Actor: "system", Message: "Same hex - no fire"})
		}

		// PSR for 20+ damage
		if atkDmg >= 20 && !defender.Prone {
			if !defender.rollPSR(1, rng) {
				defender.applyFall(rng)
				events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Failed 20+ dmg PSR, falls!"})
			} else {
				events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Passed 20+ dmg PSR"})
			}
		}
		if defDmg >= 20 && !attacker.Prone {
			if !attacker.rollPSR(1, rng) {
				attacker.applyFall(rng)
				events = append(events, ReplayEvent{Type: "psr", Actor: "attacker", Message: "Failed 20+ dmg PSR, falls!"})
			} else {
				events = append(events, ReplayEvent{Type: "psr", Actor: "attacker", Message: "Passed 20+ dmg PSR"})
			}
		}

		// PSR from crits
		if defender.NeedsPSRFromCrit && !defender.isDestroyed() {
			defender.NeedsPSRFromCrit = false
			if !defender.rollPSR(0, rng) {
				defender.applyFall(rng)
				events = append(events, ReplayEvent{Type: "psr", Actor: "defender", Message: "Failed crit PSR, falls!"})
			}
		}
		if attacker.NeedsPSRFromCrit && !attacker.isDestroyed() {
			attacker.NeedsPSRFromCrit = false
			if !attacker.rollPSR(0, rng) {
				attacker.applyFall(rng)
				events = append(events, ReplayEvent{Type: "psr", Actor: "attacker", Message: "Failed crit PSR, falls!"})
			}
		}

		// Physical attacks — both sides
		if dist == 1 {
			prevDefIS := defender.IS
			resolvePhysical(attacker, defender, rng)
			kicked := false
			for i := 0; i < NumLoc; i++ {
				if defender.IS[i] < prevDefIS[i] {
					kicked = true
					break
				}
			}
			if kicked {
				events = append(events, ReplayEvent{Type: "physical", Actor: "attacker", Message: "Kick!"})
			}

			prevAtkIS := attacker.IS
			resolvePhysical(defender, attacker, rng)
			kickedBack := false
			for i := 0; i < NumLoc; i++ {
				if attacker.IS[i] < prevAtkIS[i] {
					kickedBack = true
					break
				}
			}
			if kickedBack {
				events = append(events, ReplayEvent{Type: "physical", Actor: "defender", Message: "Kick!"})
			}
		}

		// Heat dissipation — both (including plasma heat from enemy)
		attacker.Heat += attacker.HeatPenalty; attacker.HeatPenalty = 0
		attacker.Heat -= attacker.Dissipation
		if attacker.Heat < 0 {
			attacker.Heat = 0
		}
		defender.Heat += defender.HeatPenalty; defender.HeatPenalty = 0
		defender.Heat -= defender.Dissipation
		if defender.Heat < 0 {
			defender.Heat = 0
		}

		// Shutdown/ammo explosion checks — attacker
		shutdownP := heatShutdownProb(attacker.Heat)
		if shutdownP >= 1.0 || (shutdownP > 0 && rng.Float64() < shutdownP) {
			attacker.IsShutdown = true
			events = append(events, ReplayEvent{Type: "heat", Actor: "attacker", Message: "SHUTDOWN at heat " + itoa(attacker.Heat)})
		}
		ammoExpP := heatAmmoExpProb(attacker.Heat)
		if ammoExpP > 0 && rng.Float64() < ammoExpP {
			events = append(events, ReplayEvent{Type: "heat", Actor: "attacker", Message: "Ammo explosion from heat!"})
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
			}
		}

		// Shutdown/ammo explosion checks — defender
		shutdownPD := heatShutdownProb(defender.Heat)
		if shutdownPD >= 1.0 || (shutdownPD > 0 && rng.Float64() < shutdownPD) {
			defender.IsShutdown = true
			events = append(events, ReplayEvent{Type: "heat", Actor: "defender", Message: "SHUTDOWN at heat " + itoa(defender.Heat)})
		}
		ammoExpPD := heatAmmoExpProb(defender.Heat)
		if ammoExpPD > 0 && rng.Float64() < ammoExpPD {
			events = append(events, ReplayEvent{Type: "heat", Actor: "defender", Message: "Ammo explosion from heat!"})
			type ammoBin struct {
				key string
				loc int
			}
			var bins []ammoBin
			for loc := 0; loc < NumLoc; loc++ {
				for _, slot := range defender.Slots[loc] {
					sLower := strings.ToLower(slot)
					if strings.Contains(sLower, "ammo") && !strings.Contains(sLower, "gauss") {
						key := parseAmmoSlotKey(slot)
						if defender.Ammo[key] > 0 {
							bins = append(bins, ammoBin{key, loc})
						}
					}
				}
			}
			if len(bins) > 0 {
				bin := bins[rng.IntN(len(bins))]
				defender.ammoExplosion(bin.loc, bin.key, rng)
			}
		}

		turnData.Events = events
		turnData.Attacker = snapshotMech(attacker)
		turnData.Defender = snapshotMech(defender)
		replay.Turns = append(replay.Turns, turnData)

		if attacker.isDestroyed() {
			replay.Result = "attacker_destroyed"
			break
		}
		if defender.isDestroyed() {
			replay.Result = "defender_destroyed_turn_" + itoa(turn)
			break
		}
	}

	if replay.Result == "" {
		replay.Result = "timeout_" + itoa(maxTurns)
	}
	return replay
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func replayToJSON(r *ReplayData) ([]byte, error) {
	return json.Marshal(r)
}
