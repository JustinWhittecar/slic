package main

import (
	"math"
)

// ─── Line of Sight ──────────────────────────────────────────────────────────
// BT LOS: Draw line from center of attacker hex to center of target hex.
// Check all intervening hexes for blocking terrain.

// LOSResult contains the result of a LOS check.
type LOSResult struct {
	CanSee       bool
	WoodsMod     int  // +1 per light woods, +2 per heavy woods in LOS
	TargetCover  int  // +1 if target is in woods (partial cover)
	ElevationMod int  // -1 if attacker higher, +1 if lower (for to-hit)
}

// LOSCache provides per-decision LOS caching (single-goroutine, no mutex).
type LOSCache struct {
	cache map[uint64]LOSResult
}

func newLOSCache() *LOSCache {
	return &LOSCache{cache: make(map[uint64]LOSResult, 4096)}
}

func losKey(from, to HexCoord) uint64 {
	return uint64(from.Col)<<48 | uint64(from.Row)<<32 | uint64(to.Col)<<16 | uint64(to.Row)
}

// CheckLOSCached checks LOS with caching. Use for tactical AI scoring.
func CheckLOSCached(board *Board, from, to HexCoord, cache *LOSCache) LOSResult {
	if cache == nil {
		return CheckLOS(board, from, to)
	}
	key := losKey(from, to)
	if r, ok := cache.cache[key]; ok {
		return r
	}
	r := CheckLOS(board, from, to)
	cache.cache[key] = r
	return r
}

// CheckLOS determines line of sight between two hexes on a board.
func CheckLOS(board *Board, from, to HexCoord) LOSResult {
	result := LOSResult{CanSee: true}

	fromHex := board.Get(from)
	toHex := board.Get(to)
	if fromHex == nil || toHex == nil {
		result.CanSee = false
		return result
	}

	// Mech height: elevation + 1 level (mechs are ~2 levels tall, eyes at top)
	fromLevel := fromHex.Elevation + 1
	toLevel := toHex.Elevation + 1

	// Elevation modifier
	if fromHex.Elevation > toHex.Elevation {
		result.ElevationMod = -1 // attacker higher = bonus
	} else if fromHex.Elevation < toHex.Elevation {
		result.ElevationMod = 1 // attacker lower = penalty
	}

	// Target in woods = partial cover
	if hasWoods, level := toHex.HasTerrain(TerrainWoods); hasWoods {
		result.TargetCover = level // +1 light, +2 heavy
	}

	// Walk hex line between from and to (no allocation)
	dist := HexDistance(from, to)
	ac := OffsetToCube(from)
	bc := OffsetToCube(to)

	woodsCount := 0
	for i := 1; i < dist; i++ {
		t := float64(i) / float64(dist)
		q := lerp(float64(ac.Q), float64(bc.Q), t)
		r := lerp(float64(ac.R), float64(bc.R), t)
		s := lerp(float64(ac.S), float64(bc.S), t)
		h := CubeToOffset(cubeRound(q, r, s))

		hex := board.Get(h)
		if hex == nil {
			continue
		}

		// Elevation blocking: intervening hex blocks LOS if its elevation
		// is higher than both the line-of-sight endpoints at that point
		interLevel := hex.Elevation
		if hasWoods, _ := hex.HasTerrain(TerrainWoods); hasWoods {
			interLevel += 2 // woods add 2 levels of height
		}
		if hasBuilding, level := hex.HasTerrain(TerrainBuilding); hasBuilding {
			interLevel += level // building levels
		}

		// Simple LOS blocking: if intervening terrain is taller than both endpoints
		if interLevel > fromLevel && interLevel > toLevel {
			result.CanSee = false
			return result
		}

		// Check if intervening hex is at same or higher elevation and has woods
		// This blocks LOS if there are 2+ intervening woods hexes
		if hasWoods, level := hex.HasTerrain(TerrainWoods); hasWoods {
			if hex.Elevation >= minInt(fromHex.Elevation, toHex.Elevation) {
				woodsCount++
				if level >= 2 {
					woodsCount++ // heavy woods count as 2
				}
			}
		}

		// Heavy building blocks LOS
		if hasBuilding, level := hex.HasTerrain(TerrainBuilding); hasBuilding && level >= 3 {
			if hex.Elevation >= minInt(fromHex.Elevation, toHex.Elevation) {
				result.CanSee = false
				return result
			}
		}
	}

	// 2+ woods levels in LOS = blocked
	if woodsCount >= 2 {
		result.CanSee = false
		return result
	}

	// 1 intervening woods = +1 to-hit
	result.WoodsMod = woodsCount

	return result
}

// HexLine returns the hexes along a line from a to b (exclusive of a and b).
// Uses cube coordinate interpolation.
func HexLine(a, b HexCoord) []HexCoord {
	dist := HexDistance(a, b)
	if dist <= 1 {
		return nil
	}

	ac := OffsetToCube(a)
	bc := OffsetToCube(b)

	var result []HexCoord
	for i := 1; i < dist; i++ {
		t := float64(i) / float64(dist)
		// Interpolate in cube coords
		q := lerp(float64(ac.Q), float64(bc.Q), t)
		r := lerp(float64(ac.R), float64(bc.R), t)
		s := lerp(float64(ac.S), float64(bc.S), t)
		// Round to nearest hex
		hex := cubeRound(q, r, s)
		result = append(result, CubeToOffset(hex))
	}
	return result
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func cubeRound(q, r, s float64) CubeCoord {
	rq := math.Round(q)
	rr := math.Round(r)
	rs := math.Round(s)

	dq := math.Abs(rq - q)
	dr := math.Abs(rr - r)
	ds := math.Abs(rs - s)

	if dq > dr && dq > ds {
		rq = -rr - rs
	} else if dr > ds {
		rr = -rq - rs
	} else {
		rs = -rq - rr
	}

	return CubeCoord{Q: int(rq), R: int(rr), S: int(rs)}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
