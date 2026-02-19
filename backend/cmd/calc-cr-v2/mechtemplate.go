package main

import (
	"context"
	"strings"

	"github.com/JustinWhittecar/slic/internal/ingestion"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Ammo tables ────────────────────────────────────────────────────────────

var ammoPerTon = map[string]int{
	"AC/2": 45, "AC/5": 20, "AC/10": 10, "AC/20": 5,
	"LRM-5": 24, "LRM-10": 12, "LRM-15": 8, "LRM-20": 6,
	"SRM-2": 50, "SRM-4": 25, "SRM-6": 15,
	"Streak SRM-2": 50, "Streak SRM-4": 25, "Streak SRM-6": 15,
	"MRM-10": 24, "MRM-20": 12, "MRM-30": 8, "MRM-40": 6,
	"Ultra AC/2": 45, "Ultra AC/5": 20, "Ultra AC/10": 10, "Ultra AC/20": 5,
	"LB 2-X AC": 45, "LB 5-X AC": 20, "LB 10-X AC": 10, "LB 20-X AC": 5,
	"Rotary AC/2": 45, "Rotary AC/5": 20,
	"Gauss Rifle": 8, "Light Gauss": 16, "Heavy Gauss": 4,
	"Machine Gun": 200, "Light Machine Gun": 200, "Heavy Machine Gun": 100,
	"ATM 3": 20, "ATM 6": 10, "ATM 9": 7, "ATM 12": 5,
	"MML-3": 40, "MML-5": 24, "MML-7": 17, "MML-9": 13,
	"HAG/20": 6, "HAG/30": 4, "HAG/40": 3,
	"Arrow IV": 5,
	"Plasma Rifle": 10, "Plasma Cannon": 10,
	"Sniper": 10, "Thumper": 20, "Long Tom": 5,
	"Light AC/2": 45, "Light AC/5": 20,
	"Thunderbolt 5": 12, "Thunderbolt 10": 6, "Thunderbolt 15": 4, "Thunderbolt 20": 3,
}

var canonicalAmmoPerTon map[string]int

func init() {
	canonicalAmmoPerTon = make(map[string]int, len(ammoPerTon))
	for k, v := range ammoPerTon {
		canonicalAmmoPerTon[canonicalAmmoType(k)] = v
	}
}

// canonicalAmmoType maps any ammo/weapon name to a canonical key
func canonicalAmmoType(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "(omnipod)", "")
	s = strings.ReplaceAll(s, "- full", "")
	s = strings.ReplaceAll(s, "- half", "")
	s = strings.ReplaceAll(s, "cluster", "")
	s = strings.ReplaceAll(s, "ammo", "")
	s = strings.ReplaceAll(s, "clan ", "")
	s = strings.ReplaceAll(s, "is ", "")
	s = strings.ReplaceAll(s, "inner sphere ", "")
	s = strings.TrimSpace(s)

	norm := strings.NewReplacer(" ", "", "-", "", "/", "", "_", "", ".", "").Replace(s)

	switch {
	case strings.Contains(norm, "heavygauss"):
		return "heavy gauss"
	case strings.Contains(norm, "lightgauss"):
		return "light gauss"
	case strings.Contains(norm, "apgauss"):
		return "ap gauss"
	case strings.Contains(norm, "gaussrifle") || norm == "gauss" || strings.Contains(norm, "clgauss") || strings.Contains(norm, "isgauss"):
		return "gauss rifle"
	case strings.Contains(norm, "lbx") || strings.Contains(norm, "lb2x") || strings.Contains(norm, "lb5x") || strings.Contains(norm, "lb10x") || strings.Contains(norm, "lb20x"):
		if strings.Contains(norm, "20") {
			return "lb 20-x ac"
		} else if strings.Contains(norm, "10") {
			return "lb 10-x ac"
		} else if strings.Contains(norm, "5") {
			return "lb 5-x ac"
		} else if strings.Contains(norm, "2") {
			return "lb 2-x ac"
		}
		return "lb 10-x ac"
	case strings.Contains(norm, "ultra"):
		if strings.Contains(norm, "20") {
			return "ultra ac/20"
		} else if strings.Contains(norm, "10") {
			return "ultra ac/10"
		} else if strings.Contains(norm, "5") {
			return "ultra ac/5"
		} else if strings.Contains(norm, "2") {
			return "ultra ac/2"
		}
		return "ultra ac/5"
	case strings.Contains(norm, "rotary"):
		if strings.Contains(norm, "5") {
			return "rotary ac/5"
		}
		return "rotary ac/2"
	case strings.Contains(norm, "lightac") || strings.Contains(norm, "lac"):
		if strings.Contains(norm, "5") {
			return "light ac/5"
		}
		return "light ac/2"
	case strings.Contains(norm, "ac"):
		if strings.Contains(norm, "20") {
			return "ac/20"
		} else if strings.Contains(norm, "10") {
			return "ac/10"
		} else if strings.Contains(norm, "5") {
			return "ac/5"
		} else if strings.Contains(norm, "2") {
			return "ac/2"
		}
	case strings.Contains(norm, "lrm"):
		if strings.Contains(norm, "20") {
			return "lrm-20"
		} else if strings.Contains(norm, "15") {
			return "lrm-15"
		} else if strings.Contains(norm, "10") {
			return "lrm-10"
		} else if strings.Contains(norm, "5") {
			return "lrm-5"
		}
		return "lrm-10"
	case strings.Contains(norm, "streak"):
		if strings.Contains(norm, "6") {
			return "streak srm-6"
		} else if strings.Contains(norm, "4") {
			return "streak srm-4"
		} else if strings.Contains(norm, "2") {
			return "streak srm-2"
		}
		return "streak srm-4"
	case strings.Contains(norm, "srm"):
		if strings.Contains(norm, "6") {
			return "srm-6"
		} else if strings.Contains(norm, "4") {
			return "srm-4"
		} else if strings.Contains(norm, "2") {
			return "srm-2"
		}
		return "srm-4"
	case strings.Contains(norm, "mrm"):
		if strings.Contains(norm, "40") {
			return "mrm-40"
		} else if strings.Contains(norm, "30") {
			return "mrm-30"
		} else if strings.Contains(norm, "20") {
			return "mrm-20"
		} else if strings.Contains(norm, "10") {
			return "mrm-10"
		}
		return "mrm-20"
	case strings.Contains(norm, "mml"):
		if strings.Contains(norm, "9") {
			return "mml-9"
		} else if strings.Contains(norm, "7") {
			return "mml-7"
		} else if strings.Contains(norm, "5") {
			return "mml-5"
		} else if strings.Contains(norm, "3") {
			return "mml-3"
		}
		return "mml-5"
	case strings.Contains(norm, "atm"):
		if strings.Contains(norm, "12") {
			return "atm-12"
		} else if strings.Contains(norm, "9") {
			return "atm-9"
		} else if strings.Contains(norm, "6") {
			return "atm-6"
		} else if strings.Contains(norm, "3") {
			return "atm-3"
		}
		return "atm-6"
	case strings.Contains(norm, "hag"):
		if strings.Contains(norm, "40") {
			return "hag/40"
		} else if strings.Contains(norm, "30") {
			return "hag/30"
		}
		return "hag/20"
	case strings.Contains(norm, "heavymachinegun") || strings.Contains(norm, "heavymg"):
		return "heavy machine gun"
	case strings.Contains(norm, "lightmachinegun") || strings.Contains(norm, "lightmg"):
		return "light machine gun"
	case strings.Contains(norm, "machinegun") || norm == "mg" || strings.Contains(norm, "clmg") || strings.Contains(norm, "ismg") || strings.Contains(norm, "ismachinegun"):
		return "machine gun"
	case strings.Contains(norm, "arrow"):
		return "arrow iv"
	case strings.Contains(norm, "thunderbolt"):
		if strings.Contains(norm, "20") {
			return "thunderbolt-20"
		} else if strings.Contains(norm, "15") {
			return "thunderbolt-15"
		} else if strings.Contains(norm, "10") {
			return "thunderbolt-10"
		}
		return "thunderbolt-5"
	case strings.Contains(norm, "plasma"):
		if strings.Contains(norm, "cannon") {
			return "plasma cannon"
		}
		return "plasma rifle"
	case strings.Contains(norm, "ams") || strings.Contains(norm, "antimissile"):
		return "ams"
	case strings.Contains(norm, "longtom"):
		return "long tom"
	case strings.Contains(norm, "sniper"):
		return "sniper"
	case strings.Contains(norm, "thumper"):
		return "thumper"
	}

	return strings.TrimSpace(s)
}

func parseAmmoSlotKey(slotName string) string {
	return canonicalAmmoType(slotName)
}

func guessAmmoShots(slotName string) int {
	key := canonicalAmmoType(slotName)
	if v, ok := canonicalAmmoPerTon[key]; ok {
		return v
	}
	for k, v := range canonicalAmmoPerTon {
		if strings.Contains(key, k) || strings.Contains(k, key) {
			return v
		}
	}
	return 10
}

// ─── DB types ───────────────────────────────────────────────────────────────

type DBWeapon struct {
	Name     string
	Type     string
	Damage   int
	Heat     int
	MinRange int
	Short    int
	Medium   int
	Long     int
	ToHitMod int
	RackSize int
	Location string
	Quantity int
}

type DBVariant struct {
	ID         int
	Name       string
	ModelCode  string
	TechBase   string
	Tonnage    int
	WalkMP     int
	RunMP      int
	JumpMP     int
	HSCount    int
	HSType     string
	EngineType string
	StructType string
	HasTC      bool
	Weapons    []DBWeapon
}

func locNameToIndex(name string) int {
	switch name {
	case "HD", "Head":
		return LocHD
	case "CT", "Center Torso":
		return LocCT
	case "LT", "Left Torso":
		return LocLT
	case "RT", "Right Torso":
		return LocRT
	case "LA", "Left Arm", "FLL", "Front Left Leg":
		return LocLA
	case "RA", "Right Arm", "FRL", "Front Right Leg":
		return LocRA
	case "LL", "Left Leg", "RLL", "Rear Left Leg":
		return LocLL
	case "RL", "Right Leg", "RRL", "Rear Right Leg":
		return LocRL
	default:
		return -1
	}
}

func weaponToAmmoKey(name string) string {
	return canonicalAmmoType(name)
}

// ─── Build mech state from DB ───────────────────────────────────────────────

func buildMechState(v *DBVariant, mtf *ingestion.MTFData) *MechState {
	m := &MechState{
		DebugName:     v.Name + " " + v.ModelCode,
		Tonnage:       v.Tonnage,
		WalkMP:        v.WalkMP,
		RunMP:         v.RunMP,
		JumpMP:        v.JumpMP,
		HeatSinkCount: v.HSCount,
		TechBase:      v.TechBase,
		Ammo:          make(map[string]int),
	}

	hsLower := strings.ToLower(v.HSType)
	if strings.Contains(hsLower, "double") || strings.Contains(hsLower, "laser") {
		m.Dissipation = v.HSCount * 2
	} else {
		m.Dissipation = v.HSCount
	}

	engLower := strings.ToLower(v.EngineType)
	if strings.Contains(engLower, "xl") {
		m.IsXL = true
		if strings.Contains(strings.ToLower(v.TechBase), "clan") {
			m.IsClanXL = true
		}
	}

	if v.StructType != "" {
		stLower := strings.ToLower(v.StructType)
		if strings.Contains(stLower, "reinforced") {
			m.IsReinforced = true
		} else if strings.Contains(stLower, "composite") {
			m.IsComposite = true
		}
	}

	isVals := getISForTonnage(v.Tonnage)
	m.MaxIS = isVals
	m.IS = isVals

	if mtf != nil {
		for locStr, val := range mtf.ArmorValues {
			switch locStr {
			case "HD":
				m.Armor[LocHD] = val
			case "CT":
				m.Armor[LocCT] = val
			case "LT":
				m.Armor[LocLT] = val
			case "RT":
				m.Armor[LocRT] = val
			case "LA":
				m.Armor[LocLA] = val
			case "RA":
				m.Armor[LocRA] = val
			case "LL":
				m.Armor[LocLL] = val
			case "RL":
				m.Armor[LocRL] = val
			case "FLL":
				m.Armor[LocLA] = val
			case "FRL":
				m.Armor[LocRA] = val
			case "RLL":
				m.Armor[LocLL] = val
			case "RRL":
				m.Armor[LocRL] = val
			case "RTC":
				m.RearArmor[0] = val
			case "RTL":
				m.RearArmor[1] = val
			case "RTR":
				m.RearArmor[2] = val
			}
		}

		for locStr, slots := range mtf.LocationEquipment {
			li := locNameToIndex(locStr)
			if li < 0 {
				continue
			}
			m.Slots[li] = make([]string, len(slots))
			copy(m.Slots[li], slots)

			for _, slot := range slots {
				sLower := strings.ToLower(slot)
				if strings.Contains(sLower, "case ii") {
					m.HasCASEII[li] = true
				} else if strings.Contains(sLower, "case") && !strings.Contains(sLower, "ammo") {
					m.HasCASE[li] = true
				}
				if strings.Contains(sLower, "ammo") {
					key := parseAmmoSlotKey(slot)
					if key == "ams" {
						m.AMSAmmo += 12
					} else {
						shots := guessAmmoShots(slot)
						m.Ammo[key] += shots
					}
				}
				if strings.Contains(sLower, "targeting computer") || strings.Contains(sLower, "istargeting computer") || strings.Contains(sLower, "cltargeting computer") {
					m.HasTargetingComputer = true
				}
				if strings.Contains(sLower, "anti-missile") || (strings.Contains(sLower, "ams") && !strings.Contains(sLower, "ammo")) {
					m.HasAMS = true
					if strings.Contains(sLower, "laser") {
						m.IsLaserAMS = true
					}
				}
				if strings.Contains(sLower, "artemis v") && !strings.Contains(sLower, "artemis iv") {
					m.HasArtemisV = true
				} else if strings.Contains(sLower, "artemis iv") {
					m.HasArtemisIV = true
				}
				if strings.Contains(sLower, "apollo") {
					m.HasApollo = true
				}
			}
		}
	}

	for _, w := range v.Weapons {
		cat := categorizeWeapon(w.Name)
		li := locNameToIndex(w.Location)
		if li < 0 {
			li = LocCT
		}

		ammoKey := ""
		if w.Type == "ballistic" || w.Type == "missile" || w.Type == "artillery" {
			ammoKey = weaponToAmmoKey(w.Name)
		}
		if w.Type == "energy" {
			ammoKey = ""
		}

		for q := 0; q < w.Quantity; q++ {
			thm := w.ToHitMod
			if m.HasTargetingComputer && isDirectFire(cat) {
				thm--
			}
			if cat == catLBX {
				thm--
			}
			if cat == catMRM && m.HasApollo {
				thm-- // Apollo FCS: -1 to-hit for MRMs
			}
			sw := SimWeapon{
				Name:       w.Name,
				Category:   cat,
				Location:   li,
				Damage:     w.Damage,
				Heat:       w.Heat,
				MinRange:   w.MinRange,
				ShortRange: w.Short,
				MedRange:   w.Medium,
				LongRange:  w.Long,
				ToHitMod:   thm,
				RackSize:   w.RackSize,
				Type:       w.Type,
				AmmoKey:    ammoKey,
			}
			m.Weapons = append(m.Weapons, sw)
		}
	}

	return m
}

// ─── HBK-4P hardcoded baseline ──────────────────────────────────────────────

func buildHBK4P() *MechState {
	m := &MechState{
		DebugName:     "Hunchback HBK-4P",
		Tonnage:       50,
		WalkMP:        4,
		RunMP:         6,
		JumpMP:        0,
		HeatSinkCount: 23,
		Dissipation:   23,
		TechBase:      "Inner Sphere",
		Ammo:          make(map[string]int),
	}

	m.IS = [NumLoc]int{3, 16, 12, 12, 8, 8, 12, 12}
	m.MaxIS = m.IS
	m.Armor = [NumLoc]int{9, 26, 20, 20, 16, 16, 20, 20}
	m.RearArmor = [3]int{5, 4, 4}

	m.Slots[LocHD] = []string{"Life Support", "Sensors", "Cockpit", "Small Laser", "Sensors", "Life Support"}
	m.Slots[LocCT] = []string{"Engine", "Engine", "Engine", "Gyro", "Gyro", "Gyro", "Gyro", "Engine", "Engine", "Engine", "Heat Sink", "Heat Sink"}
	m.Slots[LocLT] = []string{"Heat Sink", "Heat Sink", "Heat Sink", "Heat Sink", "Heat Sink", "Heat Sink", "Heat Sink"}
	m.Slots[LocRT] = []string{"Heat Sink", "Heat Sink", "Medium Laser", "Medium Laser", "Medium Laser", "Medium Laser", "Medium Laser", "Medium Laser"}
	m.Slots[LocLA] = []string{"Shoulder", "Upper Arm", "Lower Arm", "Hand", "Medium Laser"}
	m.Slots[LocRA] = []string{"Shoulder", "Upper Arm", "Lower Arm", "Hand", "Medium Laser"}
	m.Slots[LocLL] = []string{"Hip", "Upper Leg", "Lower Leg", "Foot", "Heat Sink", "Heat Sink"}
	m.Slots[LocRL] = []string{"Hip", "Upper Leg", "Lower Leg", "Foot", "Heat Sink", "Heat Sink"}

	for i := 0; i < 6; i++ {
		m.Weapons = append(m.Weapons, SimWeapon{
			Name: "Medium Laser", Category: catNormal, Location: LocRT,
			Damage: 5, Heat: 3, ShortRange: 3, MedRange: 6, LongRange: 9,
			Type: "energy",
		})
	}
	m.Weapons = append(m.Weapons, SimWeapon{
		Name: "Medium Laser", Category: catNormal, Location: LocLA,
		Damage: 5, Heat: 3, ShortRange: 3, MedRange: 6, LongRange: 9,
		Type: "energy",
	})
	m.Weapons = append(m.Weapons, SimWeapon{
		Name: "Medium Laser", Category: catNormal, Location: LocRA,
		Damage: 5, Heat: 3, ShortRange: 3, MedRange: 6, LongRange: 9,
		Type: "energy",
	})
	m.Weapons = append(m.Weapons, SimWeapon{
		Name: "Small Laser", Category: catNormal, Location: LocHD,
		Damage: 3, Heat: 1, ShortRange: 1, MedRange: 2, LongRange: 3,
		Type: "energy",
	})

	return m
}

// ─── Load variants from DB ──────────────────────────────────────────────────

func loadVariantsFromDB(ctx context.Context, pool *pgxpool.Pool, filter string) ([]DBVariant, error) {
	query := `
		SELECT v.id, v.name, v.model_code, COALESCE(c.tech_base, 'Inner Sphere'),
			   c.tonnage, vs.walk_mp, vs.run_mp, vs.jump_mp,
			   vs.heat_sink_count, vs.heat_sink_type, vs.engine_type,
			   COALESCE(vs.structure_type, 'Standard'), COALESCE(vs.has_targeting_computer, false)
		FROM variants v
		JOIN chassis c ON c.id = v.chassis_id
		JOIN variant_stats vs ON vs.variant_id = v.id`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []DBVariant
	for rows.Next() {
		var v DBVariant
		err := rows.Scan(&v.ID, &v.Name, &v.ModelCode, &v.TechBase,
			&v.Tonnage, &v.WalkMP, &v.RunMP, &v.JumpMP,
			&v.HSCount, &v.HSType, &v.EngineType,
			&v.StructType, &v.HasTC)
		if err != nil {
			continue
		}
		variants = append(variants, v)
	}

	if filter != "" {
		filters := strings.Split(filter, ",")
		var filtered []DBVariant
		for _, v := range variants {
			for _, f := range filters {
				f = strings.TrimSpace(f)
				if strings.Contains(v.ModelCode, f) || strings.Contains(v.Name+" "+v.ModelCode, f) {
					filtered = append(filtered, v)
					break
				}
			}
		}
		variants = filtered
	}

	return variants, nil
}

func loadWeaponsForVariants(ctx context.Context, pool *pgxpool.Pool, variants []DBVariant) {
	for i := range variants {
		v := &variants[i]
		wRows, err := pool.Query(ctx, `
			SELECT e.name, COALESCE(e.type,''), COALESCE(e.damage,0), COALESCE(e.heat,0),
				   COALESCE(e.min_range,0), COALESCE(e.short_range,0), COALESCE(e.medium_range,0),
				   COALESCE(e.long_range,0), COALESCE(e.to_hit_modifier,0), COALESCE(e.rack_size,0),
				   ve.location, ve.quantity
			FROM variant_equipment ve
			JOIN equipment e ON e.id = ve.equipment_id
			WHERE ve.variant_id = $1
			  AND e.type IN ('energy','ballistic','missile','artillery')`, v.ID)
		if err != nil {
			continue
		}
		for wRows.Next() {
			var w DBWeapon
			wRows.Scan(&w.Name, &w.Type, &w.Damage, &w.Heat,
				&w.MinRange, &w.Short, &w.Medium, &w.Long,
				&w.ToHitMod, &w.RackSize, &w.Location, &w.Quantity)
			v.Weapons = append(v.Weapons, w)
		}
		wRows.Close()
	}
}
