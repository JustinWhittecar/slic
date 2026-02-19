package bvcalc

import (
	"math"
	"testing"
)

func TestAmmoBV(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"IS Ammo LRM-20", 23},
		{"IS Ammo LRM-5", 6},
		{"Clan Ammo SRM-6", 7},
		{"IS Ammo AC/20", 22},
		{"IS Ammo AC/2", 5},
		{"IS Gauss Ammo", 40},
		{"Clan Streak SRM-6 Ammo", 11},
		{"IS Ammo MRM-40", 28},
		{"Clan Ultra AC/5 Ammo", 14},
		{"IS LB 10-X AC Ammo", 15},
	}
	for _, tt := range tests {
		got := AmmoBV(tt.name)
		if got != tt.want {
			t.Errorf("AmmoBV(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestSpeedFactor(t *testing.T) {
	tests := []struct {
		runMP, jumpMP int
		want          float64
	}{
		{6, 0, 1.12},   // Archer
		{5, 0, 1.0},    // Walk 3/Run 5
		{8, 0, 1.37},   // Walk 5/Run 8
		{6, 6, 1.50},   // Run 6 + Jump 6 → speedMP=9
		{4, 0, 0.88},   // Slow mech
	}
	for _, tt := range tests {
		got := SpeedFactor(tt.runMP, tt.jumpMP)
		if math.Abs(got-tt.want) > 0.02 {
			t.Errorf("SpeedFactor(%d,%d) = %.4f, want ~%.2f", tt.runMP, tt.jumpMP, got, tt.want)
		}
	}
}

func TestTMM(t *testing.T) {
	tests := []struct {
		mp   int
		want int
	}{
		{0, 0}, {2, 0}, {3, 1}, {4, 1}, {5, 2}, {6, 2}, {7, 3}, {9, 3}, {10, 4},
	}
	for _, tt := range tests {
		got := TMM(tt.mp)
		if got != tt.want {
			t.Errorf("TMM(%d) = %d, want %d", tt.mp, got, tt.want)
		}
	}
}

func TestMovementHeat(t *testing.T) {
	tests := []struct {
		runMP, jumpMP int
		stealth       bool
		want          int
	}{
		{6, 0, false, 2},   // Running only
		{6, 4, false, 4},   // Jump 4 > run heat
		{6, 2, false, 3},   // Jump 2 → min 3
		{6, 0, true, 12},   // Stealth +10
	}
	for _, tt := range tests {
		got := MovementHeat(tt.runMP, tt.jumpMP, tt.stealth)
		if got != tt.want {
			t.Errorf("MovementHeat(%d,%d,%v) = %d, want %d", tt.runMP, tt.jumpMP, tt.stealth, got, tt.want)
		}
	}
}

func TestISPoints(t *testing.T) {
	if ISPointsByTonnage[70] != 116 {
		t.Errorf("IS points for 70 tons = %d, want 116", ISPointsByTonnage[70])
	}
	if ISPointsByTonnage[100] != 171 {
		t.Errorf("IS points for 100 tons = %d, want 171", ISPointsByTonnage[100])
	}
}
