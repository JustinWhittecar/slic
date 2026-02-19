package bvcalc

import "strings"

// AmmoBV returns the BV per ton for a given ammo type name from MTF files.
// Names follow MegaMek conventions like "IS Ammo LRM-20", "Clan Ammo SRM-6", etc.
func AmmoBV(ammoName string) int {
	n := strings.ToLower(ammoName)

	// Strip common prefixes
	n = strings.TrimPrefix(n, "is ")
	n = strings.TrimPrefix(n, "clan ")
	n = strings.TrimPrefix(n, "cl ")

	// Normalize - strip "ammo" prefix or suffix
	n = strings.TrimPrefix(n, "ammo ")
	n = strings.TrimSuffix(n, " ammo")

	// Handle special ammo variants (Artemis, Narc, Thunder, etc.)
	// These generally have the same BV as their base type
	n = strings.Replace(n, " artemis-capable", "", 1)
	n = strings.Replace(n, " artemis v-capable", "", 1)
	n = strings.Replace(n, " narc-capable", "", 1)
	n = strings.Replace(n, " torpedo", "", 1)

	// Check the lookup table
	if bv, ok := ammoBVTable[n]; ok {
		return bv
	}

	// Try partial matching for common patterns
	for pattern, bv := range ammoBVTable {
		if strings.Contains(n, pattern) {
			return bv
		}
	}

	return 0
}

// IsExplosiveAmmo returns true if this ammo type is explosive (for penalty calculation).
// Gauss ammo is NOT explosive for penalty purposes.
func IsExplosiveAmmo(ammoName string) bool {
	n := strings.ToLower(ammoName)
	if strings.Contains(n, "gauss") {
		return false
	}
	// Must actually be an ammo item (short name, starts with known prefix)
	if len(n) > 100 {
		return false
	}
	return (strings.HasPrefix(n, "is ammo") || strings.HasPrefix(n, "clan ammo") ||
		strings.HasPrefix(n, "cl ammo") || strings.HasPrefix(n, "ammo"))
}

// IsAMSAmmo returns true if this is AMS ammo
func IsAMSAmmo(ammoName string) bool {
	n := strings.ToLower(ammoName)
	return strings.Contains(n, "ams") || strings.Contains(n, "anti-missile")
}

// ammoBVTable maps normalized ammo names to BV per ton
var ammoBVTable = map[string]int{
	// Standard AC
	"ac/2":  5,
	"ac/5":  9,
	"ac/10": 15,
	"ac/20": 22,

	// LB-X AC (same as standard AC)
	"lb 2-x ac":      5,
	"lb 5-x ac":      9,
	"lb 10-x ac":     15,
	"lb 20-x ac":     22,
	"lb 2-x":         5,
	"lb 5-x":         9,
	"lb 10-x":        15,
	"lb 20-x":        22,
	"lb2-x ac":       5,
	"lb5-x ac":       9,
	"lb10-x ac":      15,
	"lb20-x ac":      22,
	"lbx ac 2":       5,
	"lbx ac 5":       9,
	"lbx ac 10":      15,
	"lbx ac 20":      22,

	// Ultra AC
	"ultra ac/2":  7,
	"ultra ac/5":  14,
	"ultra ac/10": 26,
	"ultra ac/20": 35,

	// Rotary AC
	"rotary ac/2": 15,
	"rotary ac/5": 31,

	// Light AC
	"light ac/2": 4,
	"light ac/5": 8,

	// Hyper Velocity AC
	"hyper velocity auto cannon/2":  5,
	"hyper velocity auto cannon/5":  9,
	"hyper velocity auto cannon/10": 15,
	"hvac/2":                        5,
	"hvac/5":                        9,
	"hvac/10":                       15,

	// Improved AC
	"improved autocannon/2":  5,
	"improved autocannon/5":  9,
	"improved autocannon/10": 15,
	"improved autocannon/20": 22,

	// LRM
	"lrm-5":  6,
	"lrm-10": 11,
	"lrm-15": 17,
	"lrm-20": 23,
	"lrm 5":  6,
	"lrm 10": 11,
	"lrm 15": 17,
	"lrm 20": 23,

	// SRM
	"srm-2": 3,
	"srm-4": 5,
	"srm-6": 7,
	"srm 2": 3,
	"srm 4": 5,
	"srm 6": 7,

	// Streak SRM
	"streak srm-2":  4,
	"streak srm-4":  7,
	"streak srm-6":  11,
	"streak srm 2":  4,
	"streak srm 4":  7,
	"streak srm 6":  11,

	// Streak LRM
	"streak lrm 5":  6,
	"streak lrm 10": 11,
	"streak lrm 15": 17,
	"streak lrm 20": 23,

	// MRM
	"mrm-10": 7,
	"mrm-20": 14,
	"mrm-30": 21,
	"mrm-40": 28,
	"mrm 10": 7,
	"mrm 20": 14,
	"mrm 30": 21,
	"mrm 40": 28,

	// MML
	"mml-3 lrm": 4,
	"mml-3 srm": 2,
	"mml-5 lrm": 6,
	"mml-5 srm": 3,
	"mml-7 lrm": 8,
	"mml-7 srm": 5,
	"mml-9 lrm": 11,
	"mml-9 srm": 7,
	"mml 3 lrm": 4,
	"mml 3 srm": 2,
	"mml 5 lrm": 6,
	"mml 5 srm": 3,
	"mml 7 lrm": 8,
	"mml 7 srm": 5,
	"mml 9 lrm": 11,
	"mml 9 srm": 7,

	// ATM
	"atm-3":    14,
	"atm-6":    26,
	"atm-9":    36,
	"atm-12":   52,
	"atm 3":    14,
	"atm 6":    26,
	"atm 9":    36,
	"atm 12":   52,

	// Gauss
	"gauss":             40,
	"gauss rifle":       40,
	"heavy gauss rifle": 43,
	"heavy gauss":       43,
	"light gauss rifle": 20,
	"light gauss":       20,
	"improved gauss":    40,
	"hyper-assault gauss rifle/20": 30,
	"hyper-assault gauss rifle/30": 30,
	"hyper-assault gauss rifle/40": 30,
	"hag/20":            30,
	"hag/30":            30,
	"hag/40":            30,
	"ap gauss rifle":    3,
	"magshot":           2,

	// Machine Guns
	"machine gun":           1,
	"mg":                    1,
	"light machine gun":     1,
	"heavy machine gun":     1,

	// AMS
	"ams":                   11,
	"anti-missile system":   11,

	// Narc / iNarc
	"narc":          0,
	"narc beacon":   0,
	"inarc":         0,

	// Thunderbolt
	"thunderbolt-5":  5,
	"thunderbolt-10": 10,
	"thunderbolt-15": 15,
	"thunderbolt-20": 20,

	// Arrow IV
	"arrow iv": 10,

	// Plasma
	"plasma rifle": 26,
	"plasma cannon": 21,

	// Sniper/Thumper/Long Tom
	"sniper":  6,
	"thumper": 5,
	"long tom": 25,

	// Extended LRM
	"extended lrm-5":  6,
	"extended lrm-10": 11,
	"extended lrm-15": 17,
	"extended lrm-20": 23,

	// Enhanced LRM
	"enhanced lrm-5":  6,
	"enhanced lrm-10": 11,
	"enhanced lrm-15": 17,
	"enhanced lrm-20": 23,

	// iATM
	"iatm 3":  14,
	"iatm 6":  26,
	"iatm 9":  36,
	"iatm 12": 52,

	// Clan LRM/SRM (Clan uses same BV per ton)
	// Handled by the same entries above after prefix stripping

	// ProtoMech AC
	"protomech ac/2": 4,
	"protomech ac/4": 6,
	"protomech ac/8": 12,

	// Silver Bullet
	"silver bullet": 22,

	// Rifle
	"rifle (cannon, light)":  3,
	"rifle (cannon, medium)": 6,
	"rifle (cannon, heavy)":  9,
}
