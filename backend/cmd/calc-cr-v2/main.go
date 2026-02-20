package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"bytes"
	"compress/gzip"
	"database/sql"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/JustinWhittecar/slic/internal/db"
	"github.com/JustinWhittecar/slic/internal/ingestion"
	_ "modernc.org/sqlite"
)

// ─── Constants ──────────────────────────────────────────────────────────────

const (
	numSimsPerBoard = 10
	numBoardPairs   = 20
	maxTurns        = 200
	gunnerySkill    = 4
	pilotingSkill   = 5
	kFactor         = 3.5
)

// ─── Simulation core ────────────────────────────────────────────────────────

// simulateCombat2D runs one sim on a 2D hex board.
// Attacker tries to destroy defender. Returns turns until defender destroyed/withdrawn.
func simulateCombat2D(board *Board, attackerTemplate, defenderTemplate *MechState, rng *rand.Rand) int {
	attacker := cloneMech(attackerTemplate)
	defender := cloneMech(defenderTemplate)

	// Deploy: attacker rows 1-3, defender rows 15-17
	attacker.Pos = HexCoord{Col: board.Width/2 + 1, Row: 2}
	attacker.Facing = 3 // face south
	defender.Pos = HexCoord{Col: board.Width/2 + 1, Row: board.Height - 1}
	defender.Facing = 0 // face north

	for turn := 1; turn <= maxTurns; turn++ {
		if attacker.isDestroyed() || defender.isDestroyed() {
			return turn - 1
		}
		if defender.isForcedWithdrawal() {
			return turn
		}

		// Unjam RAC weapons (RAC clears jam after 1 turn)
		for i := range attacker.Weapons {
			w := &attacker.Weapons[i]
			if w.Jammed && w.Category == catRotaryAC {
				w.Jammed = false
			}
		}
		for i := range defender.Weapons {
			w := &defender.Weapons[i]
			if w.Jammed && w.Category == catRotaryAC {
				w.Jammed = false
			}
		}

		// Handle shutdown — attacker
		if attacker.IsShutdown {
			// Restart requires a roll against shutdown avoidance TN (BMM p.52)
			tn := heatShutdownTN(attacker.Heat)
			if tn >= 13 || roll2d6(rng) < tn {
				// Failed restart — stay shutdown, dissipate heat, skip turn
				attacker.Heat += attacker.HeatPenalty
				attacker.HeatPenalty = 0
				attacker.Heat -= attacker.Dissipation
				if attacker.Heat < 0 {
					attacker.Heat = 0
				}
				// Still process defender heat
				defender.Heat += defender.HeatPenalty
				defender.HeatPenalty = 0
				defender.Heat -= defender.Dissipation
				if defender.Heat < 0 {
					defender.Heat = 0
				}
				continue
			}
			// Successful restart
			attacker.IsShutdown = false
			// Involuntary shutdown PSR: piloting + 3 modifier (BMM p.52)
			if !attacker.Prone {
				psrTN := pilotingSkill + 3
				if roll2d6(rng) < psrTN {
					attacker.applyFall(rng)
				}
			}
			if attacker.isDestroyed() {
				return maxTurns // attacker died
			}
		}

		// Handle shutdown — defender
		if defender.IsShutdown {
			tn := heatShutdownTN(defender.Heat)
			if tn >= 13 || roll2d6(rng) < tn {
				// Failed restart — stay shutdown, dissipate heat, skip turn for defender
				defender.Heat += defender.HeatPenalty
				defender.HeatPenalty = 0
				defender.Heat -= defender.Dissipation
				if defender.Heat < 0 {
					defender.Heat = 0
				}
				attacker.Heat += attacker.HeatPenalty
				attacker.HeatPenalty = 0
				attacker.Heat -= attacker.Dissipation
				if attacker.Heat < 0 {
					attacker.Heat = 0
				}
				continue
			}
			defender.IsShutdown = false
			if !defender.Prone {
				psrTN := pilotingSkill + 3
				if roll2d6(rng) < psrTN {
					defender.applyFall(rng)
				}
			}
			if defender.isDestroyed() {
				return turn
			}
		}

		// Stand from prone
		if attacker.Prone {
			attacker.Heat += 1
			if attacker.rollPSRForStanding(rng) {
				attacker.Prone = false
			} else {
				attacker.applyFall(rng)
				if attacker.isDestroyed() {
					return maxTurns
				}
			}
		}
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

		// Initiative
		atkInit := rng.IntN(6) + 1
		defInit := rng.IntN(6) + 1
		atkMovesFirst := atkInit < defInit
		if atkInit == defInit {
			atkMovesFirst = rng.IntN(2) == 0
		}

		// Movement
		atkWalk := 0
		atkRun := 0
		if !attacker.Prone {
			atkWalk = attacker.effectiveWalkMP()
			atkRun = attacker.effectiveRunMP()
		}
		defWalk := 0
		defRun := 0
		if !defender.Prone {
			defWalk = defender.effectiveWalkMP()
			defRun = defender.effectiveRunMP()
		}

		// Convert MechState to MechState2-like for tactics
		atkM2 := &MechState2{
			Pos: attacker.Pos, Facing: attacker.Facing,
			WalkMP: atkWalk, RunMP: atkRun, JumpMP: attacker.JumpMP,
			Tonnage: attacker.Tonnage, GunnerySkill: gunnerySkill,
			Heat: attacker.Heat,
		}
		defM2 := &MechState2{
			Pos: defender.Pos, Facing: defender.Facing,
			WalkMP: defWalk, RunMP: defRun, JumpMP: defender.JumpMP,
			Tonnage: defender.Tonnage, GunnerySkill: gunnerySkill,
			Heat: defender.Heat,
		}
		// Copy weapons for damage estimation
		for _, w := range attacker.Weapons {
			if !w.Destroyed && !w.Jammed {
				atkM2.Weapons = append(atkM2.Weapons, SimWeapon2{
					Name: w.Name, Damage: w.Damage, Heat: w.Heat,
					MinRange: w.MinRange, ShortRange: w.ShortRange,
					MedRange: w.MedRange, LongRange: w.LongRange,
					Location: w.Location, ToHitMod: w.ToHitMod,
				})
			}
		}
		for _, w := range defender.Weapons {
			if !w.Destroyed && !w.Jammed {
				defM2.Weapons = append(defM2.Weapons, SimWeapon2{
					Name: w.Name, Damage: w.Damage, Heat: w.Heat,
					MinRange: w.MinRange, ShortRange: w.ShortRange,
					MedRange: w.MedRange, LongRange: w.LongRange,
					Location: w.Location, ToHitMod: w.ToHitMod,
				})
			}
		}

		defOptions := collectAllMoveOptions(board, defM2)
		atkOptions := collectAllMoveOptions(board, atkM2)

		var atkChoice, defChoice ReachableHex

		if atkMovesFirst {
			// Attacker moves first (blind), defender sees
			atkChoice = ChooseMovement(board, atkM2, defM2,
				false, defM2.Pos, defM2.Facing, defOptions, rng)
			defChoice = ChooseMovement(board, defM2, atkM2,
				true, atkChoice.Coord, atkChoice.Facing, atkOptions, rng)
		} else {
			// Defender moves first (blind), attacker sees
			defChoice = ChooseMovement(board, defM2, atkM2,
				false, atkM2.Pos, atkM2.Facing, atkOptions, rng)
			atkChoice = ChooseMovement(board, atkM2, defM2,
				true, defChoice.Coord, defChoice.Facing, defOptions, rng)
		}

		// Apply movement
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

		// Torso twist
		attacker.TorsoTwist = BestTorsoTwist(attacker.Pos, attacker.Facing, defender.Pos)
		defender.TorsoTwist = BestTorsoTwist(defender.Pos, defender.Facing, attacker.Pos)

		// Reset AMS
		defender.AMSUsedThisTurn = false

		// LOS check
		dist := HexDistance(attacker.Pos, defender.Pos)
		los := CheckLOS(board, attacker.Pos, defender.Pos)

		if los.CanSee && dist > 0 {
			// Determine arc for hit table
			defEffFacing := ((defender.Facing + defender.TorsoTwist) % 6 + 6) % 6
			arcToDefender := DetermineArc(defender.Pos, defEffFacing, attacker.Pos)
			isRear := arcToDefender == ArcRear

			// Compute target number
			defTMM := tmmFromHexesMoved(defChoice.HexesMoved, defChoice.Mode)
			heatThisMod := heatToHitMod(attacker.Heat)
			// BMM p.49: 2+ sensor hits = weapon fire impossible
			if attacker.SensorHits >= 2 {
				continue
			}
			baseTarget := gunnerySkill + attacker.SensorHits*2 + heatThisMod

			// Attacker movement modifier
			switch atkChoice.Mode {
			case ModeWalk:
				baseTarget += 1
			case ModeRun:
				baseTarget += 2
			case ModeJump:
				baseTarget += 3
			}

			// Target movement modifier
			baseTarget += defTMM

			// Terrain modifiers
			baseTarget += los.WoodsMod
			baseTarget += los.TargetCover
			baseTarget += los.ElevationMod

			// Prone modifiers
			if attacker.Prone {
				baseTarget += 2
			}
			if defender.Prone {
				if dist <= 1 {
					baseTarget -= 2 // adjacent: easier to hit (BMM p.28)
				} else {
					baseTarget += 1 // non-adjacent: harder to hit (BMM p.28)
				}
			}

			// Select and fire weapons
			firingWeapons, weaponHeatTotal := selectWeaponsEV(attacker, board, defender, dist, baseTarget)
			attacker.Heat += weaponHeatTotal

			totalDmgDealt := 0
			for _, wi := range firingWeapons {
				w := &attacker.Weapons[wi]
				if w.AmmoKey != "" {
					if attacker.Ammo[w.AmmoKey] <= 0 {
						continue
					}
					attacker.Ammo[w.AmmoKey]--
				}

				target := baseTarget + w.ToHitMod + attacker.ArmActuatorHit[w.Location]
				// Artemis V: -1 to-hit in addition to cluster bonus (BMM p.110)
				if attacker.HasArtemisV && (w.Category == catLRM || w.Category == catSRM || w.Category == catMML || w.Category == catATM) {
					target -= 1
				}
				rm := rangeModifier(w, dist)
				if rm < 0 {
					continue
				}
				target += rm
				if w.MinRange > 0 && dist <= w.MinRange {
					target += w.MinRange - dist + 1
				}

				dmg := resolveWeaponFire2D(w, target, isRear, attacker, defender, rng)
				totalDmgDealt += dmg

				if defender.isDestroyed() {
					return turn
				}
			}

			// PSR for 20+ damage
			if totalDmgDealt >= 20 && !defender.Prone {
				if !defender.rollPSR(1, rng) {
					defender.applyFall(rng)
					if defender.isDestroyed() {
						return turn
					}
				}
			}

			// PSR from crits
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

		// Physical attacks
		if dist == 1 {
			resolvePhysical(attacker, defender, rng)
			if defender.isDestroyed() {
				return turn
			}
		}

		// End of turn: engine crit heat + heat dissipation
		// Engine hits do not produce heat if the 'Mech is shut down (BMM p.47)
		if !attacker.IsShutdown {
			attacker.Heat += attacker.EngineHits * 5
		}
		// Outside heat sources capped at 15 per turn (BMM p.52)
		if attacker.HeatPenalty > 15 {
			attacker.HeatPenalty = 15
		}
		attacker.Heat += attacker.HeatPenalty // plasma weapon heat from enemy
		attacker.HeatPenalty = 0
		attacker.Heat -= attacker.Dissipation
		if attacker.Heat < 0 {
			attacker.Heat = 0
		}
		if !defender.IsShutdown {
			defender.Heat += defender.EngineHits * 5
		}
		if defender.HeatPenalty > 15 {
			defender.HeatPenalty = 15
		}
		defender.Heat += defender.HeatPenalty // plasma weapon heat from enemy
		defender.HeatPenalty = 0
		defender.Heat -= defender.Dissipation
		if defender.Heat < 0 {
			defender.Heat = 0
		}

		// Heat shutdown/ammo explosion for attacker
		shutdownP := heatShutdownProb(attacker.Heat)
		if shutdownP >= 1.0 || (shutdownP > 0 && rng.Float64() < shutdownP) {
			attacker.IsShutdown = true
		}

		ammoExpP := heatAmmoExpProb(attacker.Heat)
		if ammoExpP > 0 && rng.Float64() < ammoExpP {
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

		// Heat shutdown/ammo explosion for defender (BMM p.52)
		shutdownPDef := heatShutdownProb(defender.Heat)
		if shutdownPDef >= 1.0 || (shutdownPDef > 0 && rng.Float64() < shutdownPDef) {
			defender.IsShutdown = true
		}

		ammoExpPDef := heatAmmoExpProb(defender.Heat)
		if ammoExpPDef > 0 && rng.Float64() < ammoExpPDef {
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
				if defender.isDestroyed() {
					return turn
				}
			}
		}
	}

	return maxTurns
}

// (collectMoveOptions moved to tactics.go as collectAllMoveOptions)

// ─── MechState2 — lightweight state for tactical AI ─────────────────────────

type MechState2 struct {
	Name          string
	Pos           HexCoord
	Facing        int
	WalkMP        int
	RunMP         int
	JumpMP        int
	Tonnage       int
	GunnerySkill  int
	PilotingSkill int
	Weapons       []SimWeapon2
	Prone         bool
	Heat          int
}

type SimWeapon2 struct {
	Name       string
	Damage     int
	Heat       int
	MinRange   int
	ShortRange int
	MedRange   int
	LongRange  int
	Location   int
	ToHitMod   int
	Destroyed  bool
}

// ─── Batch sim ──────────────────────────────────────────────────────────────

// PrecomputedBoards holds pre-combined board pairs to avoid re-combining per variant.
type PrecomputedBoards struct {
	Boards []*Board
}

func precomputeBoardPairs(boards []*Board, nPairs int, rng *rand.Rand) *PrecomputedBoards {
	pb := &PrecomputedBoards{Boards: make([]*Board, nPairs)}
	for i := 0; i < nPairs; i++ {
		b1 := boards[rng.IntN(len(boards))]
		b2 := boards[rng.IntN(len(boards))]
		pb.Boards[i] = CombineBoards(b1, b2)
	}
	return pb
}

func runSimsBatch2D(boards []*Board, attackerTemplate, defenderTemplate *MechState, nBoardPairs int, nSimsPerBoard int, rng *rand.Rand) float64 {
	var results []int

	for bp := 0; bp < nBoardPairs; bp++ {
		b1 := boards[rng.IntN(len(boards))]
		b2 := boards[rng.IntN(len(boards))]
		combined := CombineBoards(b1, b2)

		for s := 0; s < nSimsPerBoard; s++ {
			turns := simulateCombat2D(combined, attackerTemplate, defenderTemplate, rng)
			results = append(results, turns)
		}
	}

	sort.Ints(results)
	n := len(results)
	if n == 0 {
		return float64(maxTurns)
	}
	if n%2 == 0 {
		return float64(results[n/2-1]+results[n/2]) / 2.0
	}
	return float64(results[n/2])
}

func runSimsBatch2DPre(preBoards *PrecomputedBoards, attackerTemplate, defenderTemplate *MechState, nSimsPerBoard int, rng *rand.Rand) float64 {
	var results []int

	for _, combined := range preBoards.Boards {
		for s := 0; s < nSimsPerBoard; s++ {
			turns := simulateCombat2D(combined, attackerTemplate, defenderTemplate, rng)
			results = append(results, turns)
		}
	}

	sort.Ints(results)
	n := len(results)
	if n == 0 {
		return float64(maxTurns)
	}
	if n%2 == 0 {
		return float64(results[n/2-1]+results[n/2]) / 2.0
	}
	return float64(results[n/2])
}

// ─── Main ───────────────────────────────────────────────────────────────────

func main() {
	cpuprofile := flag.String("cpuprofile", "", "Write CPU profile to file")
	mechFilter := flag.String("mech", "", "Comma-separated mech names to test")
	testMode := flag.Bool("test", false, "Run test with 5 mechs only")
	replayMode := flag.String("replay", "", "Run replay: 'attacker_name vs defender_name'")
	replayOut := flag.String("replay-out", "", "Output file for replay JSON")
	replaySeed := flag.Int64("replay-seed", 42, "RNG seed for replay")
	genReplays := flag.Bool("gen-replays", false, "Generate replay for every variant and store in SQLite")
	genReplaysDB := flag.String("gen-replays-db", "/Users/puckopenclaw/projects/slic/slic.db", "SQLite DB path for storing replays")
	genReplaysLimit := flag.Int("gen-replays-limit", 0, "Limit number of variants to process (0=all)")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	fmt.Println("SLIC Combat Rating v2 — 2D Hex Grid Sim")
	fmt.Println("=========================================")

	// Load boards
	boardDirPath := os.Getenv("SLIC_BOARD_DIR")
	if boardDirPath == "" {
		boardDirPath = "/Users/puckopenclaw/projects/slic/data/megamek-data/data/boards"
	}

	log.Println("Loading boards...")
	boards, err := LoadBoardPool(boardDirPath)
	if err != nil {
		log.Fatalf("Error loading boards: %v", err)
	}
	log.Printf("Loaded %d standard 16x17 boards", len(boards))
	if len(boards) < 2 {
		log.Fatalf("Need at least 2 boards")
	}

	// Connect to DB
	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("DB: %v", err)
	}
	defer pool.Close()

	// Load MTF files
	mtfDir := "/Users/puckopenclaw/projects/slic/data/megamek-data/data/mekfiles"
	log.Println("Loading MTF files...")
	mtfMap := make(map[string]*ingestion.MTFData)
	_ = filepath.Walk(mtfDir, func(path string, info os.FileInfo, err error) error {
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
	log.Printf("Loaded %d MTF files", len(mtfMap))

	// Replay mode
	if *replayMode != "" {
		parts := strings.SplitN(*replayMode, " vs ", 2)
		if len(parts) != 2 {
			log.Fatalf("Replay format: 'Attacker Name vs Defender Name'")
		}
		atkName := strings.TrimSpace(parts[0])
		defName := strings.TrimSpace(parts[1])

		atkVariants, err := loadVariantsFromDB(ctx, pool, atkName)
		if err != nil || len(atkVariants) == 0 {
			log.Fatalf("Attacker not found: %s", atkName)
		}
		defVariants, err := loadVariantsFromDB(ctx, pool, defName)
		if err != nil || len(defVariants) == 0 {
			log.Fatalf("Defender not found: %s", defName)
		}

		loadWeaponsForVariants(ctx, pool, atkVariants)
		loadWeaponsForVariants(ctx, pool, defVariants)

		atkMTF := mtfMap[atkVariants[0].Name+" "+atkVariants[0].ModelCode]
		if atkMTF == nil {
			atkMTF = mtfMap[atkVariants[0].Name]
		}
		defMTF := mtfMap[defVariants[0].Name+" "+defVariants[0].ModelCode]
		if defMTF == nil {
			defMTF = mtfMap[defVariants[0].Name]
		}

		atkTemplate := buildMechState(&atkVariants[0], atkMTF)
		defTemplate := buildMechState(&defVariants[0], defMTF)
		atkTemplate.DebugName = atkName
		defTemplate.DebugName = defName

		rng := rand.New(rand.NewPCG(uint64(*replaySeed), 0))
		b1 := boards[rng.IntN(len(boards))]
		b2 := boards[rng.IntN(len(boards))]
		combined := CombineBoards(b1, b2)

		log.Printf("Running duel replay: %s vs %s (seed %d)", atkName, defName, *replaySeed)
		replay := simulateDuelReplay(combined, atkTemplate, defTemplate, rng)

		data, err := replayToJSON(replay)
		if err != nil {
			log.Fatalf("JSON: %v", err)
		}

		outFile := *replayOut
		if outFile == "" {
			outFile = "/Users/puckopenclaw/projects/slic/backend/cmd/calc-cr-v2/replay_output.json"
		}
		if err := os.WriteFile(outFile, data, 0644); err != nil {
			log.Fatalf("Write: %v", err)
		}
		log.Printf("Replay written to %s (%d turns, result: %s)", outFile, len(replay.Turns), replay.Result)
		return
	}

	// Gen-replays mode: generate a replay for every variant, store in SQLite
	if *genReplays {
		log.Println("=== GEN-REPLAYS MODE ===")

		// Open SQLite DB for writing
		slDB, err := sql.Open("sqlite", *genReplaysDB)
		if err != nil {
			log.Fatalf("Open SQLite: %v", err)
		}
		defer slDB.Close()
		slDB.Exec("PRAGMA journal_mode=WAL")
		slDB.Exec("PRAGMA synchronous=NORMAL")

		// Create table
		slDB.Exec(`CREATE TABLE IF NOT EXISTS variant_replays (
			variant_id INTEGER PRIMARY KEY,
			replay_data BLOB NOT NULL
		)`)

		// Load all variants from Postgres
		allVariants, err := loadVariantsFromDB(ctx, pool, "")
		if err != nil {
			log.Fatalf("Load variants: %v", err)
		}
		loadWeaponsForVariants(ctx, pool, allVariants)

		limit := len(allVariants)
		if *genReplaysLimit > 0 && *genReplaysLimit < limit {
			limit = *genReplaysLimit
		}

		hbkTemplate := buildHBK4P()

		log.Printf("Generating replays for %d variants...", limit)

		var genProcessed atomic.Int64
		var mu sync.Mutex

		numW := runtime.NumCPU()
		genJobs := make(chan int, limit)
		var genWg sync.WaitGroup

		// Prepare insert statement
		tx, err := slDB.Begin()
		if err != nil {
			log.Fatalf("Begin tx: %v", err)
		}
		stmt, err := tx.Prepare(`INSERT OR REPLACE INTO variant_replays (variant_id, replay_data) VALUES (?, ?)`)
		if err != nil {
			log.Fatalf("Prepare: %v", err)
		}

		type replayResult struct {
			variantID int
			data      []byte
		}
		results := make(chan replayResult, 100)

		// Writer goroutine
		done := make(chan struct{})
		go func() {
			count := 0
			for r := range results {
				mu.Lock()
				_, err := stmt.Exec(r.variantID, r.data)
				mu.Unlock()
				if err != nil {
					log.Printf("Insert variant %d: %v", r.variantID, err)
				}
				count++
				if count%100 == 0 {
					log.Printf("  Stored %d replays", count)
				}
			}
			close(done)
		}()

		for w := 0; w < numW; w++ {
			genWg.Add(1)
			go func() {
				defer genWg.Done()
				for idx := range genJobs {
					v := &allVariants[idx]
					mtf := mtfMap[v.Name+" "+v.ModelCode]
					if mtf == nil {
						mtf = mtfMap[v.Name]
					}
					mechTemplate := buildMechState(v, mtf)

					// Run 5 sims with different seeds, pick median by turn count
					const numDuelSims = 5
					type simResult struct {
						replay *ReplayData
						turns  int
					}
					var simResults []simResult
					for s := 0; s < numDuelSims; s++ {
						rng := rand.New(rand.NewPCG(uint64(v.ID), uint64(s)))
						b1 := boards[rng.IntN(len(boards))]
						b2 := boards[rng.IntN(len(boards))]
						combined := CombineBoards(b1, b2)
						r := simulateDuelReplay(combined, mechTemplate, hbkTemplate, rng)
						simResults = append(simResults, simResult{r, len(r.Turns)})
					}
					// Sort by turns, pick median
					sort.Slice(simResults, func(i, j int) bool { return simResults[i].turns < simResults[j].turns })
					replay := simResults[numDuelSims/2].replay

					jsonData, err := replayToJSON(replay)
					if err != nil {
						log.Printf("JSON %s: %v", v.Name, err)
						continue
					}

					// Gzip compress
					var buf bytes.Buffer
					gz := gzip.NewWriter(&buf)
					gz.Write(jsonData)
					gz.Close()

					results <- replayResult{v.ID, buf.Bytes()}

					n := genProcessed.Add(1)
					if n%100 == 0 {
						log.Printf("  Generated %d/%d replays", n, limit)
					}
				}
			}()
		}

		for i := 0; i < limit; i++ {
			genJobs <- i
		}
		close(genJobs)
		genWg.Wait()
		close(results)
		<-done

		stmt.Close()
		if err := tx.Commit(); err != nil {
			log.Fatalf("Commit: %v", err)
		}
		slDB.Close()

		log.Printf("Done! Generated %d replays in %s", genProcessed.Load(), *genReplaysDB)
		return
	}

	// Determine filter
	filter := *mechFilter
	if *testMode {
		filter = "HBK-4P,AS7-D,LCT-1V,Timber Wolf Prime,Dire Wolf Prime"
	}

	// Load variants
	log.Println("Loading variants from DB...")
	variants, err := loadVariantsFromDB(ctx, pool, filter)
	if err != nil {
		log.Fatalf("Load variants: %v", err)
	}
	log.Printf("Loaded %d variants", len(variants))

	// Load weapons
	log.Println("Loading weapons...")
	loadWeaponsForVariants(ctx, pool, variants)

	// Build HBK-4P baseline
	log.Println("Running HBK-4P baseline...")
	hbkTemplate := buildHBK4P()

	baseRng := rand.New(rand.NewPCG(42, 0))
	// Run more baseline sims for stability (50 board pairs × 10 sims = 500 runs)
	baselineOffense := runSimsBatch2D(boards, hbkTemplate, hbkTemplate, 50, numSimsPerBoard, baseRng)
	baselineDefense := runSimsBatch2D(boards, hbkTemplate, hbkTemplate, 50, numSimsPerBoard, baseRng)
	baselineRatio := baselineDefense / baselineOffense
	if baselineRatio == 0 {
		baselineRatio = 1.0
	}
	log.Printf("HBK-4P baseline: offense=%.1f defense=%.1f ratio=%.3f",
		baselineOffense, baselineDefense, baselineRatio)

	// Process variants
	numWorkers := runtime.NumCPU()
	log.Printf("Processing %d variants with %d workers...", len(variants), numWorkers)

	results := make(chan simResult, len(variants))
	jobs := make(chan int, len(variants))

	var processed atomic.Int64
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localRng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
			// Pre-combine board pairs per worker (avoids re-combining per variant)
			preBoards := precomputeBoardPairs(boards, numBoardPairs, localRng)
			for idx := range jobs {
				v := &variants[idx]
				mtf := mtfMap[v.Name]
				if mtf == nil {
					altName := strings.TrimSuffix(v.Name, " "+v.ModelCode)
					altName = altName + " " + v.ModelCode
					mtf = mtfMap[altName]
				}

				mechTemplate := buildMechState(v, mtf)

				offTurns := runSimsBatch2DPre(preBoards, mechTemplate, hbkTemplate, numSimsPerBoard, localRng)
				defTurns := runSimsBatch2DPre(preBoards, hbkTemplate, mechTemplate, numSimsPerBoard, localRng)

				ratio := defTurns / offTurns
				score := 5.0 + kFactor*math.Log(ratio/baselineRatio)
				if score < 1 {
					score = 1
				}
				if score > 10 {
					score = 10
				}

				results <- simResult{v.ID, v.Name + " " + v.ModelCode, offTurns, defTurns, score}

				n := processed.Add(1)
				if n%50 == 0 || *testMode || filter != "" {
					log.Printf("  [%d/%d] %s %s: off=%.1f def=%.1f CR=%.2f",
						n, len(variants), v.Name, v.ModelCode, offTurns, defTurns, score)
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

	// Collect results
	var allResults []simResult
	updated := 0
	for r := range results {
		allResults = append(allResults, r)
		if !*testMode {
			_, err := pool.Exec(ctx, `
				UPDATE variant_stats SET combat_rating = $2, offense_turns = $3, defense_turns = $4
				WHERE variant_id = $1`, r.id, r.score, r.offense, r.defense)
			if err != nil {
				log.Printf("Update %d: %v", r.id, err)
				continue
			}
			updated++
		}
	}

	if *testMode || filter != "" {
		// Sort by score
		sort.Slice(allResults, func(i, j int) bool {
			return allResults[i].score > allResults[j].score
		})

		fmt.Println("\n═══════════════════════════════════════════")
		fmt.Println("V2 Test Results")
		fmt.Println("═══════════════════════════════════════════")
		fmt.Printf("%-35s %8s %8s %6s\n", "Mech", "Offense", "Defense", "CR")
		fmt.Println("───────────────────────────────────────────")
		for _, r := range allResults {
			fmt.Printf("%-35s %8.1f %8.1f %6.2f\n", r.name, r.offense, r.defense, r.score)
		}

		// Write results file
		writeTestResults(allResults)
	}

	log.Printf("Done! Updated %d variants", updated)
}

func writeTestResults(results []simResult) {
	f, err := os.Create("/Users/puckopenclaw/projects/slic/backend/cmd/calc-cr-v2/V2_TEST_RESULTS.md")
	if err != nil {
		log.Printf("Failed to write results: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprintln(f, "# SLIC Combat Rating V2 — Test Results")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "## Configuration")
	fmt.Fprintf(f, "- Sims per board pair: %d\n", numSimsPerBoard)
	fmt.Fprintf(f, "- Board pairs per mech: %d\n", numBoardPairs)
	fmt.Fprintf(f, "- Total sims per mech: %d\n", numSimsPerBoard*numBoardPairs)
	fmt.Fprintf(f, "- Max turns: %d\n", maxTurns)
	fmt.Fprintf(f, "- Gunnery/Piloting: %d/%d\n", gunnerySkill, pilotingSkill)
	fmt.Fprintln(f, "- Board: 2x 16x17 standard boards combined (32x17)")
	fmt.Fprintln(f, "- Deployment: attacker rows 1-3, defender rows 15-17")
	fmt.Fprintln(f, "- 2D hex grid with terrain, LOS, arcs, torso twist")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "## Results")
	fmt.Fprintln(f, "")
	fmt.Fprintf(f, "| %-35s | %8s | %8s | %6s |\n", "Mech", "Offense", "Defense", "CR")
	fmt.Fprintf(f, "|%-37s|%10s|%10s|%8s|\n", strings.Repeat("-", 37), strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 8))
	for _, r := range results {
		fmt.Fprintf(f, "| %-35s | %8.1f | %8.1f | %6.2f |\n", r.name, r.offense, r.defense, r.score)
	}
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "## Key Changes from V1")
	fmt.Fprintln(f, "- Real 2D hex grid with terrain costs and LOS blocking")
	fmt.Fprintln(f, "- Arc-based hit tables (front/rear) from actual positions")
	fmt.Fprintln(f, "- Torso twist for weapon arc management")
	fmt.Fprintln(f, "- Initiative determines movement order (second mover advantage)")
	fmt.Fprintln(f, "- Woods cover (+1/+2 to-hit), elevation advantage (-1/+1)")
	fmt.Fprintln(f, "- Terrain movement costs affect positioning")

	log.Println("Wrote results to V2_TEST_RESULTS.md")
}

type simResult struct {
	id      int
	name    string
	offense float64
	defense float64
	score   float64
}
