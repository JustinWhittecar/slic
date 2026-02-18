# SLIC Combat Rating v2: 2D Hex Grid Sim

## Overview

Replace the current 1D distance-based sim with a full 2D hex grid sim running
on real MegaMek mapsheets. Each Monte Carlo run plays out on a randomly selected
pair of 16×17 mapsheets (32×17 play area, standard BT setup). This captures
terrain, elevation, LOS, cover, and initiative — producing Combat Ratings that
reflect how mechs actually perform in real BattleTech games.

## Why

The current sim runs in a vacuum — flat open field, 1D distance. This:
- **Undervalues jump jets** (no terrain to traverse, no elevation to exploit)
- **Undervalues short-range brawlers** (no cover to close distance behind)
- **Overvalues LRM boats** (no LOS blocking on open field)
- **Ignores initiative** (no positional advantage from moving second)
- **Ignores speed for flanking** (rear arc access is probability-based, not positional)

Averaging across 50+ real maps washes out map-specific meta and captures how
well a mech performs across varied terrain — which IS combat effectiveness.

## Map Pool

- **180 official 16×17 mapsheets** from MegaMek mm-data
- **540+ unofficial** (use official only for consistency)
- Two mapsheets placed side-by-side = 32×17 hex play area
- ~100 random pairings per mech (shuffle from 180 pool)
- Filter out extreme maps (all-water, all-lava) by tag

### Board Format
```
size 16 17
hex XXYY elevation "terrain:level;terrain:level" "theme"
```

Terrain types we care about:
- `woods:1` (light) — +1 to-hit LOS, +1 MP to enter
- `woods:2` (heavy) — +2 to-hit LOS, +2 MP to enter, blocks LOS behind
- `water:1-3` — depth-dependent movement/combat effects
- `rough:1` — +1 MP to enter, PSR
- `pavement:1` — road movement bonus
- `building:1-4` — LOS blocking, CF-based cover
- Elevation (integer) — LOS, +1/-1 to-hit modifiers, movement cost

## Architecture

### 1. Hex Grid (`hexgrid.go`)

```go
type HexCoord struct {
    Col, Row int  // axial coordinates (MegaMek uses offset coords XXYY)
}

type Hex struct {
    Coord     HexCoord
    Elevation int
    Terrain   []TerrainFeature  // woods, water, rough, etc.
}

type TerrainFeature struct {
    Type  TerrainType  // Woods, Water, Rough, Pavement, Building, Sand
    Level int          // woods:1 vs woods:2, water depth, etc.
}

type Board struct {
    Width, Height int
    Hexes         map[HexCoord]*Hex
}
```

Standard BT hex grid: odd columns offset down by half. Hex adjacency follows
standard offset coordinate rules (6 neighbors per hex).

**Combined board**: Two 16×17 boards side-by-side = cols 1-32, rows 1-17.
Second board's X coords offset by +16.

### 2. Hex Math (`hexmath.go`)

- `Distance(a, b HexCoord) int` — hex distance (convert to cube coords, manhattan/2)
- `Neighbors(h HexCoord) []HexCoord` — 6 adjacent hexes
- `HexesInRange(center HexCoord, radius int) []HexCoord`
- `HexLine(a, b HexCoord) []HexCoord` — hexes along line of sight
- `Facing` — 0-5, which hexside the mech faces

### 3. Movement (`movement.go`)

```go
type ReachableHex struct {
    Coord     HexCoord
    MPSpent   int
    Facing    int       // 0-5
    MoveMode  MoveMode  // Stand, Walk, Run, Jump
    Path      []HexCoord
}

func ReachableHexes(board *Board, start HexCoord, facing int, walkMP int,
    runMP int, jumpMP int, mode MoveMode) []ReachableHex
```

- **Walking/Running**: Flood-fill with MP costs. Terrain costs:
  - Clear/pavement: 1 MP
  - Light woods: 2 MP
  - Heavy woods: 3 MP  
  - Rough: 2 MP
  - Water depth 1: 2 MP, depth 2: 3 MP, depth 3: 4 MP
  - Elevation change: +1 MP per level up, free going down
  - Each hex entered costs a facing change if needed (+1 MP per hexside turned)
- **Jumping**: Straight-line, ignores terrain. Costs 1 MP per hex. Lands in any hex within jump MP. Can change facing freely on landing.
- **Running**: 1.5× walk MP (round up), +2 to-hit attacker modifier

### 4. Line of Sight (`los.go`)

```go
func HasLOS(board *Board, from, to HexCoord, fromElev, toElev int) (bool, int)
// Returns: canSee, toHitModifier (from intervening terrain)
```

- Draw hex line from attacker to target
- Check each intervening hex for:
  - **Elevation**: Higher hex blocks LOS to lower hex behind it
  - **Woods**: Light = +1 to-hit (partial LOS), Heavy = +2 to-hit. Second woods hex blocks LOS.
  - **Buildings**: Block LOS based on CF level
- Partial cover: +1 to-hit if target is in woods/partial cover

Standard BT LOS rules (BMM p.22-23).

### 5. Firing Arcs (`arcs.go`)

```go
func InFiringArc(shooter HexCoord, facing int, target HexCoord) ArcType
// Returns: Front, LeftSide, RightSide, Rear
```

BT firing arcs based on facing:
- **Forward**: 3 hexsides centered on facing direction (120°)
- **Left/Right**: 1 hexside each
- **Rear**: 1 hexside (directly behind)

Arms fire into forward + adjacent side arcs.
Torso weapons fire into forward arc (+ side with torso twist).

For the sim: determine which arc the TARGET is relative to the SHOOTER
(for torso twist decisions) and which arc the SHOOTER is relative to the 
TARGET (for hit location table — front vs rear).

### 6. Initiative System

Each turn:
1. Both sides roll 1d6 for initiative
2. **Winner moves SECOND** (reacts to opponent's position)
3. Initiative winner can see where opponent moved, then choose position

This is huge for:
- **Fast mechs**: With initiative, circle behind slow mechs for rear shots
- **Strikers**: Speed + initiative = consistent rear arc access
- **Slow mechs**: Lose initiative often, get flanked

Implementation:
```go
// Each turn, roll initiative
atkInit := roll1d6(rng)
defInit := roll1d6(rng)

if atkInit >= defInit {
    // Attacker won — moves second (advantage)
    defPos := chooseBestHex(defender, ...)  // defender moves first (blind)
    atkPos := chooseBestHex(attacker, ..., defPos)  // attacker reacts
} else {
    // Defender won — moves second
    atkPos := chooseBestHex(attacker, ...)
    defPos := chooseBestHex(defender, ..., atkPos)
}
```

The mech that moves second can optimize position knowing where the opponent is.
The mech that moves first must predict/guess.

### 7. Tactical AI — Hex Scoring (`tactics.go`)

The core challenge. Each mech evaluates reachable hexes and picks the best one.

```go
func ScoreHex(board *Board, mech *MechState, hex ReachableHex, 
    opponent *MechState, opponentPos HexCoord, 
    knowsOpponentPos bool) float64
```

Score components:
1. **Damage I can deal**: Expected damage from this hex to opponent (accounting for range, LOS, arc)
2. **Damage I'll take**: Expected damage opponent can deal to me from their position (or estimated position)
3. **Cover bonus**: Am I in woods? (+1/+2 to-hit for opponent shooting at me)
4. **Elevation bonus**: Am I higher? (+1 to-hit advantage attacking down, -1 penalty attacking up)
5. **Arc advantage**: Can I shoot front but they shoot my rear? (rear = no arm weapons, weaker armor)
6. **Rear shot opportunity**: If I'm behind them, I hit rear armor (much thinner)
7. **Future mobility**: Don't paint myself into a corner

For the **first mover** (doesn't know opponent's final position):
- Estimate opponent will move toward their optimal range
- Prefer hexes with cover
- Prefer hexes that maintain flexibility

For the **second mover** (knows opponent position):
- Optimize directly: best damage, best arc, use cover
- Actively seek rear arc if speed allows

### 8. Deployment

Each sim run:
- Pick 2 random 16×17 boards, combine into 32×17
- Attacker deploys on rows 1-3, defender on rows 15-17 (or randomize edges)
- Starting facing toward opponent
- Alternatively: random hex in deployment zone, since we average over many runs

### 9. Performance Considerations

Current sim: 1,000 runs × 4,076 mechs × ~50 turns = ~200M turn evaluations.
Each turn evaluates ~10 movement options → 2B evaluations total. Runs in ~5 min.

New sim: Each turn evaluates 20-50 reachable hexes per mech. LOS checks per hex.
Roughly 5-10× slower per turn. But we can:
- **Reduce to 200-500 runs per mech** (map variety provides natural variance)
- **Cache LOS** between hex pairs per board
- **Precompute reachable hexes** once per movement phase
- **Parallelize** across maps (each map pair is independent)

Target: ~15-20 min for full sim (acceptable for offline batch computation).

### 10. Map Selection Strategy

Not all 180 maps are equally representative. Options:
1. **Random pairs**: Pick 2 random 16×17 maps per run. Simple. ~50-100 pairings per mech.
2. **Curated pool**: Select ~30 representative maps covering terrain archetypes (open, wooded, urban, hilly, mixed). More controlled.
3. **Weighted by tournament frequency**: If BTCC uses specific map packs, weight those.

Recommendation: Start with **all official 16×17 maps**, random pairing. Filter out
maps tagged with exotic terrain (lava, lunar). This gives the broadest average.

## Implementation Phases

### Phase A: Foundation (hex grid + movement + LOS)
- Board parser
- Hex coordinate math
- Movement pathfinding (flood fill with terrain costs)  
- LOS calculation
- Unit tests for all of the above

### Phase B: Combat Integration
- Port weapon fire to use 2D positions (range from hex distance, LOS check)
- Firing arcs + rear arc determination from actual positions
- Terrain to-hit modifiers (woods cover, elevation)
- Hit location table selection (front/rear/left/right from actual arc)

### Phase C: Tactical AI
- Hex scoring function
- Movement decision for first mover (no info) vs second mover (full info)
- Torso twist decisions
- Deploy positioning

### Phase D: Initiative
- Per-turn initiative roll
- First/second mover asymmetry
- Test that fast mechs benefit from initiative wins

### Phase E: Integration + Tuning
- Wire into Monte Carlo harness
- Map pool selection + filtering
- Performance optimization
- Recalibrate scoring (kFactor, baseline) for new distribution
- Validate results make sense (JJ mechs up, LRM boats more variance, etc.)

## Expected Impact on Combat Ratings

- **Jump jet mechs** ↑↑ (terrain traversal, elevation play)
- **Fast strikers** ↑ (initiative → rear arc, terrain navigation)
- **Short-range brawlers** ↑ (woods cover for approach)
- **LRM boats** ↓ on dense maps, ↑ on open maps (averages out, probably slight ↓)
- **Slow assaults** slight ↓ (harder to navigate terrain, lose initiative flanking)
- **Balanced mechs** stable (perform well across map types)

## Files

```
backend/cmd/calc-efficiency/
├── main.go          (orchestrator — mostly unchanged)
├── hexgrid.go       (Board, Hex, parsing)
├── hexmath.go       (distance, neighbors, cube coords)
├── movement.go      (ReachableHexes, terrain costs)
├── los.go           (LOS, terrain modifiers)
├── arcs.go          (firing arcs, rear determination)
├── tactics.go       (hex scoring, AI decisions)
└── initiative.go    (per-turn init rolls, move ordering)
```

## Decisions (Confirmed 2026-02-17)

1. **Torso twist**: YES — model it. 60° arc shift matters for asymmetric loadouts.
2. **Map orientation**: 32×17 (wide, standard BT layout)
3. **Map pool**: All 180 official 16×17 maps, random pairing
4. **Architecture**: Clean v2 rebuild (`calc-cr-v2/`), not incremental refactor
5. **Initiative**: YES — winner moves second, fast mechs extract more value
