package bvcalc

import "math"

// Armor type modifiers for defensive BV
var ArmorTypeModifier = map[string]float64{
	"Standard":                     1.0,
	"Standard(Inner Sphere)":       1.0,
	"Standard(Clan)":               1.0,
	"Ferro-Fibrous":                1.0,
	"Ferro-Fibrous(Inner Sphere)":  1.0,
	"Ferro-Fibrous(Clan)":          1.0,
	"Light Ferro-Fibrous":          1.0,
	"Light Ferro-Fibrous(Clan)":    1.0,
	"Heavy Ferro-Fibrous":          1.0,
	"Heavy Ferro-Fibrous(Clan)":    1.0,
	"Stealth":                      1.0,
	"Stealth(Inner Sphere)":        1.0,
	"Stealth Armor Type I":         1.0,
	"Stealth Armor Type II":        1.0,
	"Reactive":                     1.0,
	"Reactive(Inner Sphere)":       1.0,
	"Reactive(Clan)":               1.0,
	"Reflective":                   1.0,
	"Reflective(Inner Sphere)":     1.0,
	"Reflective(Clan)":             1.0,
	"Hardened":                     1.0,
	"Hardened(Inner Sphere)":       1.0,
	"Hardened(Clan)":               1.0,
	"Industrial":                   1.0,
	"Heavy Industrial":             1.0,
	"Commercial":                   0.5,
	"Primitive":                    1.0,
	"Patchwork":                    1.0, // handled per-location in full impl
}

// Structure type modifiers for defensive BV
var StructureTypeModifier = map[string]float64{
	"Standard":                     1.0,
	"IS Standard":                  1.0,
	"Clan Standard":                1.0,
	"Endo Steel":                   1.0,
	"IS Endo Steel":                1.0,
	"Clan Endo Steel":              1.0,
	"IS Endo-Composite":            1.0,
	"Clan Endo-Composite":          1.0,
	"Endo-Composite":               1.0,
	"IS Endo-Steel":                1.0,
	"Clan Endo-Steel":              1.0,
	"Composite":                    0.5,
	"Industrial":                   0.5,
	"IS Industrial":                0.5,
	"IS Reinforced":                2.0,
	"Reinforced":                   2.0,
}

// Engine type modifiers for structure BV
func EngineTypeModifier(engineType string) float64 {
	switch {
	case contains(engineType, "XL") && containsAny(engineType, "IS", "Inner Sphere"):
		return 0.5
	case contains(engineType, "XL") && contains(engineType, "Clan"):
		return 0.75
	case contains(engineType, "XL"):
		// Bare XL - check tech base later; default IS
		return 0.5
	case contains(engineType, "Light"):
		return 0.75
	case contains(engineType, "XXL"):
		return 0.5
	default: // Standard, Compact, ICE, Fuel Cell, Fission
		return 1.0
	}
}

// GyroModifier returns the BV modifier for gyro type
func GyroModifier(gyroType string) float64 {
	switch {
	case contains(gyroType, "Heavy-Duty"), contains(gyroType, "Heavy Duty"):
		return 1.0
	default: // Standard, Compact, XL
		return 0.5
	}
}

// TMM calculates Target Movement Modifier from MP
func TMM(mp int) int {
	switch {
	case mp <= 0:
		return 0
	case mp <= 2:
		return 0
	case mp <= 4:
		return 1
	case mp <= 6:
		return 2
	case mp <= 9:
		return 3
	case mp <= 12:
		return 4
	case mp <= 17:
		return 5
	case mp <= 24:
		return 6
	default:
		return 7
	}
}

// DefensiveFactor returns 1 + TMM/10
func DefensiveFactor(tmm int) float64 {
	return 1.0 + float64(tmm)/10.0
}

// SpeedFactor calculates the speed factor for OBR
func SpeedFactor(runMP, jumpMP int) float64 {
	speedMP := runMP
	if jumpMP > 0 {
		jmpBonus := int(math.Ceil(float64(jumpMP) / 2.0))
		if runMP+jmpBonus > speedMP {
			speedMP = runMP
		}
		// Use whichever is greater: run or run+ceil(jump/2)
		candidate := runMP + jmpBonus
		if candidate > speedMP {
			speedMP = candidate
		}
	}
	base := 1.0 + float64(speedMP-5)/10.0
	if base < 0.1 {
		base = 0.1
	}
	sf := math.Pow(base, 1.2)
	return math.Round(sf*100) / 100
}

// MovementHeat returns the movement heat for BV calculation
func MovementHeat(runMP, jumpMP int, hasStealth bool) int {
	runHeat := 2
	jumpHeat := 0
	if jumpMP > 0 {
		jumpHeat = jumpMP
		if jumpHeat < 3 {
			jumpHeat = 3
		}
	}
	heat := runHeat
	if jumpHeat > heat {
		heat = jumpHeat
	}
	if hasStealth {
		heat += 10
	}
	return heat
}

// InternalStructurePoints returns total IS points for a given tonnage
var ISPointsByTonnage = map[int]int{
	10: 17, 15: 25, 20: 33, 25: 42, 30: 50,
	35: 58, 40: 66, 45: 75, 50: 83, 55: 91,
	60: 99, 65: 107, 70: 116, 75: 124, 80: 132,
	85: 140, 90: 148, 95: 157, 100: 171,
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		containsLower(s, sub))
}

func containsLower(s, sub string) bool {
	sl := toLower(s)
	subl := toLower(sub)
	for i := 0; i <= len(sl)-len(subl); i++ {
		if sl[i:i+len(subl)] == subl {
			return true
		}
	}
	return false
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}
