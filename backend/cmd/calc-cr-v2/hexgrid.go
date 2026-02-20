package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ─── Hex Coordinates ────────────────────────────────────────────────────────
// MegaMek uses offset coordinates (col, row) with odd-column offset (odd cols shift down).
// We convert to cube coordinates for distance/line calculations.

type HexCoord struct {
	Col, Row int // 1-indexed offset coords matching MegaMek XXYY format
}

type CubeCoord struct {
	Q, R, S int
}

// OffsetToCube converts offset coords (odd-q layout) to cube coords.
func OffsetToCube(h HexCoord) CubeCoord {
	q := h.Col - 1 // 0-index
	r := h.Row - 1
	x := q
	z := r - (q-(q&1))/2
	y := -x - z
	return CubeCoord{Q: x, R: y, S: z}
}

// CubeToOffset converts cube coords back to offset coords (odd-q, 1-indexed).
func CubeToOffset(c CubeCoord) HexCoord {
	col := c.Q
	row := c.S + (c.Q-(c.Q&1))/2
	return HexCoord{Col: col + 1, Row: row + 1}
}

// HexDistance returns the hex distance between two offset coordinates.
func HexDistance(a, b HexCoord) int {
	ac := OffsetToCube(a)
	bc := OffsetToCube(b)
	return (abs(ac.Q-bc.Q) + abs(ac.R-bc.R) + abs(ac.S-bc.S)) / 2
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ─── Facing & Neighbors ─────────────────────────────────────────────────────
// Facing 0-5: 0=N, 1=NE, 2=SE, 3=S, 4=SW, 5=NW (clockwise from top)

// Neighbors returns the 6 adjacent hex coordinates (odd-q offset grid).
func Neighbors(h HexCoord) [6]HexCoord {
	col := h.Col
	row := h.Row
	odd := col%2 == 1 // odd columns (1-indexed: cols 1,3,5... are odd)

	if odd {
		return [6]HexCoord{
			{col, row - 1},     // 0: N
			{col + 1, row},     // 1: NE
			{col + 1, row + 1}, // 2: SE
			{col, row + 1},     // 3: S
			{col - 1, row + 1}, // 4: SW
			{col - 1, row},     // 5: NW
		}
	}
	return [6]HexCoord{
		{col, row - 1},     // 0: N
		{col + 1, row - 1}, // 1: NE
		{col + 1, row},     // 2: SE
		{col, row + 1},     // 3: S
		{col - 1, row},     // 4: SW
		{col - 1, row - 1}, // 5: NW
	}
}

// FacingDirection returns which facing (0-5) from 'from' most closely
// points toward 'to'.
func FacingDirection(from, to HexCoord) int {
	fc := OffsetToCube(from)
	tc := OffsetToCube(to)
	dq := float64(tc.Q - fc.Q)
	ds := float64(tc.S - fc.S)
	// Convert to angle; hex grid q axis is 30° from horizontal
	angle := math.Atan2(ds+dq/2, dq*math.Sqrt(3)/2)
	// Normalize to 0-2π, then map to facing 0-5
	deg := angle * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	// Facing 0 = North (up) = -90° in standard coords
	// Adjust: MegaMek facing 0 = N, angle 0 = E
	// Facing 0 (N) = 90°, 1 (NE) = 30°, 2 (SE) = 330°, etc.
	// Simpler: use cube coord direction
	facing := int(math.Round(deg/60)) % 6
	// Map from math angle to BT facing
	// This needs careful mapping — let's use the direct approach
	return facing
}

// ArcType represents which arc a target is in relative to a mech's facing.
type ArcType int

const (
	ArcFront ArcType = iota
	ArcLeft
	ArcRight
	ArcRear
)

// DetermineArc returns which arc 'target' is in relative to 'mech' with given facing.
// BT arcs: Forward = facing ±1 hexside (3 hexsides = 180°), rear = opposite 1 hexside,
// left/right = 1 hexside each.
// Actually: Forward = 3 hexsides centered on facing. Left = 1 hexside. Right = 1 hexside. Rear = 1 hexside.
// Total = 6 hexsides.
// Forward arc: facing-1, facing, facing+1 (mod 6)
// Left arc: facing+2 (mod 6)  — note: BT left = counterclockwise
// Right arc: facing-2 (mod 6) — BT right = clockwise
// Rear arc: facing+3 (mod 6)
func DetermineArc(mech HexCoord, facing int, target HexCoord) ArcType {
	dir := bearingToFacing(mech, target)
	diff := ((dir - facing) % 6 + 6) % 6
	switch diff {
	case 0, 1, 5:
		return ArcFront
	case 2:
		return ArcRight
	case 4:
		return ArcLeft
	case 3:
		return ArcRear
	}
	return ArcFront
}

// bearingToFacing returns which of the 6 hex directions target is from source.
// Uses integer dot products against cube direction vectors.
func bearingToFacing(from, to HexCoord) int {
	if from == to {
		return 0
	}
	fc := OffsetToCube(from)
	tc := OffsetToCube(to)
	dq := tc.Q - fc.Q
	dr := tc.R - fc.R
	ds := tc.S - fc.S

	// Cube direction vectors for each facing
	// 0(N): (0,+1,-1), 1(NE): (+1,0,-1), 2(SE): (+1,-1,0)
	// 3(S): (0,-1,+1), 4(SW): (-1,0,+1), 5(NW): (-1,+1,0)
	type dir3 struct{ q, r, s int }
	dirs := [6]dir3{
		{0, 1, -1}, {1, 0, -1}, {1, -1, 0},
		{0, -1, 1}, {-1, 0, 1}, {-1, 1, 0},
	}

	bestFacing := 0
	bestDot := -(1 << 30)
	for i, d := range dirs {
		dot := dq*d.q + dr*d.r + ds*d.s
		if dot > bestDot {
			bestDot = dot
			bestFacing = i
		}
	}
	return bestFacing
}

// bearingToFacingOLD is the old float64 version, kept for reference.
func bearingToFacingOLD(from, to HexCoord) int {
	if from == to {
		return 0
	}
	bestFacing := 0
	bestDist := math.MaxFloat64
	fc := OffsetToCube(from)
	tc := OffsetToCube(to)
	dq := float64(tc.Q - fc.Q)
	dr := float64(tc.R - fc.R)
	ds := float64(tc.S - fc.S)

	// Unit vectors for each facing in cube coords
	// Facing 0 (N): q=0, r=+1, s=-1
	// Facing 1 (NE): q=+1, r=0, s=-1
	// Facing 2 (SE): q=+1, r=-1, s=0
	// Facing 3 (S): q=0, r=-1, s=+1
	// Facing 4 (SW): q=-1, r=0, s=+1
	// Facing 5 (NW): q=-1, r=+1, s=0
	dirs := [6][3]float64{
		{0, 1, -1},  // N
		{1, 0, -1},  // NE
		{1, -1, 0},  // SE
		{0, -1, 1},  // S
		{-1, 0, 1},  // SW
		{-1, 1, 0},  // NW
	}

	for i, d := range dirs {
		// Dot product (higher = more aligned)
		dot := dq*d[0] + dr*d[1] + ds*d[2]
		// Use negative dot as "distance" (we want max alignment)
		dist := -dot
		if dist < bestDist {
			bestDist = dist
			bestFacing = i
		}
	}
	return bestFacing
}

// ─── Terrain ────────────────────────────────────────────────────────────────

type TerrainType int

const (
	TerrainWoods    TerrainType = iota // level 1=light, 2=heavy
	TerrainWater                       // level = depth
	TerrainRough                       // level 1 or 2
	TerrainPavement
	TerrainRoad
	TerrainBuilding // level = CF class (1-4)
	TerrainSand
	TerrainSwamp
	TerrainMud
)

type TerrainFeature struct {
	Type  TerrainType
	Level int
}

type Hex struct {
	Coord     HexCoord
	Elevation int
	Terrain   []TerrainFeature
}

func (h *Hex) HasTerrain(t TerrainType) (bool, int) {
	for _, f := range h.Terrain {
		if f.Type == t {
			return true, f.Level
		}
	}
	return false, 0
}

// ─── Board ──────────────────────────────────────────────────────────────────

type Board struct {
	Width, Height int
	Hexes         map[HexCoord]*Hex // kept for parsing compatibility
	Grid          []Hex             // flat 2D grid: (col-1)*Height + (row-1)
}

func NewBoard(w, h int) *Board {
	return &Board{
		Width:  w,
		Height: h,
		Hexes:  make(map[HexCoord]*Hex, w*h),
	}
}

// buildGrid converts the map-based Hexes into a flat slice for O(1) lookups.
func (b *Board) buildGrid() {
	b.Grid = make([]Hex, b.Width*b.Height)
	// Initialize all hexes (so Get never returns nil for in-bounds coords)
	for col := 1; col <= b.Width; col++ {
		for row := 1; row <= b.Height; row++ {
			idx := (col-1)*b.Height + (row - 1)
			b.Grid[idx] = Hex{Coord: HexCoord{Col: col, Row: row}}
		}
	}
	for _, hex := range b.Hexes {
		if hex.Coord.Col >= 1 && hex.Coord.Col <= b.Width && hex.Coord.Row >= 1 && hex.Coord.Row <= b.Height {
			idx := (hex.Coord.Col-1)*b.Height + (hex.Coord.Row - 1)
			b.Grid[idx] = *hex
		}
	}
	b.Hexes = nil // free map; all lookups use flat Grid from here
}

func (b *Board) InBounds(h HexCoord) bool {
	return h.Col >= 1 && h.Col <= b.Width && h.Row >= 1 && h.Row <= b.Height
}

func (b *Board) Get(h HexCoord) *Hex {
	if b.Grid != nil {
		if h.Col >= 1 && h.Col <= b.Width && h.Row >= 1 && h.Row <= b.Height {
			return &b.Grid[(h.Col-1)*b.Height+(h.Row-1)]
		}
		return nil
	}
	return b.Hexes[h]
}

// ─── Board Parser ───────────────────────────────────────────────────────────

func ParseBoard(path string) (*Board, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var board *Board
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' || line == "end" {
			continue
		}

		if strings.HasPrefix(line, "size ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				w, _ := strconv.Atoi(parts[1])
				h, _ := strconv.Atoi(parts[2])
				board = NewBoard(w, h)
			}
			continue
		}

		if strings.HasPrefix(line, "tag ") {
			continue
		}

		if strings.HasPrefix(line, "hex ") {
			if board == nil {
				continue
			}
			parseHexLine(board, line)
		}
	}
	if board != nil {
		board.buildGrid()
	}
	return board, scanner.Err()
}

func parseHexLine(board *Board, line string) {
	// Format: hex XXYY elevation "terrain;terrain" "theme"
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return
	}

	coord := parts[1]
	if len(coord) != 4 {
		return
	}
	col, _ := strconv.Atoi(coord[:2])
	row, _ := strconv.Atoi(coord[2:])
	elev, _ := strconv.Atoi(parts[2])

	hex := &Hex{
		Coord:     HexCoord{Col: col, Row: row},
		Elevation: elev,
	}

	// Parse terrain string (in quotes)
	if len(parts) >= 4 {
		terrainStr := strings.Trim(parts[3], "\"")
		if terrainStr != "" {
			for _, feat := range strings.Split(terrainStr, ";") {
				feat = strings.TrimSpace(feat)
				if feat == "" {
					continue
				}
				tf := parseTerrainFeature(feat)
				if tf != nil {
					hex.Terrain = append(hex.Terrain, *tf)
				}
			}
		}
	}

	board.Hexes[hex.Coord] = hex
	// Invalidate grid cache so it gets rebuilt
	board.Grid = nil
}

func parseTerrainFeature(s string) *TerrainFeature {
	// Format: "type:level:extra" or "type:level"
	parts := strings.Split(s, ":")
	if len(parts) < 1 {
		return nil
	}

	name := strings.ToLower(parts[0])
	level := 1
	if len(parts) >= 2 {
		level, _ = strconv.Atoi(parts[1])
	}

	switch {
	case name == "woods":
		return &TerrainFeature{Type: TerrainWoods, Level: level}
	case name == "water":
		return &TerrainFeature{Type: TerrainWater, Level: level}
	case name == "rough":
		return &TerrainFeature{Type: TerrainRough, Level: level}
	case name == "pavement":
		return &TerrainFeature{Type: TerrainPavement, Level: level}
	case name == "road":
		return &TerrainFeature{Type: TerrainRoad, Level: level}
	case name == "building":
		return &TerrainFeature{Type: TerrainBuilding, Level: level}
	case name == "sand":
		return &TerrainFeature{Type: TerrainSand, Level: level}
	case name == "swamp":
		return &TerrainFeature{Type: TerrainSwamp, Level: level}
	case name == "mud":
		return &TerrainFeature{Type: TerrainMud, Level: level}
	default:
		// ground_fluff, foliage_elev, bridge, etc. — cosmetic, skip
		return nil
	}
}

// CombineBoards places two 16×17 boards side-by-side into a 32×17 board.
// Board B's columns are offset by boardA.Width.
func CombineBoards(a, b *Board) *Board {
	combined := NewBoard(a.Width+b.Width, max(a.Height, b.Height))
	// Copy from Grid (flat array) since Hexes may be nil after buildGrid
	if a.Grid != nil {
		for col := 1; col <= a.Width; col++ {
			for row := 1; row <= a.Height; row++ {
				idx := (col-1)*a.Height + (row - 1)
				hex := a.Grid[idx]
				combined.Hexes[hex.Coord] = &hex
			}
		}
	} else {
		for coord, hex := range a.Hexes {
			combined.Hexes[coord] = hex
		}
	}
	if b.Grid != nil {
		for col := 1; col <= b.Width; col++ {
			for row := 1; row <= b.Height; row++ {
				idx := (col-1)*b.Height + (row - 1)
				hex := b.Grid[idx]
				newCoord := HexCoord{Col: hex.Coord.Col + a.Width, Row: hex.Coord.Row}
				hex.Coord = newCoord
				combined.Hexes[newCoord] = &hex
			}
		}
	} else {
		for coord, hex := range b.Hexes {
			newCoord := HexCoord{Col: coord.Col + a.Width, Row: coord.Row}
			newHex := *hex
			newHex.Coord = newCoord
			combined.Hexes[newCoord] = &newHex
		}
	}
	combined.buildGrid()
	return combined
}

// ─── Board Pool ─────────────────────────────────────────────────────────────

func LoadBoardPool(dir string) ([]*Board, error) {
	var boards []*Board
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".board") {
			return err
		}
		// Skip unofficial maps
		if strings.Contains(path, "/unofficial/") {
			return nil
		}
		b, err := ParseBoard(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: skipping %s: %v\n", path, err)
			return nil
		}
		if b != nil && b.Width == 16 && b.Height == 17 {
			boards = append(boards, b)
		}
		return nil
	})
	return boards, err
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
