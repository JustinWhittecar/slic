package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/JustinWhittecar/slic/internal/db"
)

// TMM table
func tmmFromMP(mp int) int {
	switch {
	case mp <= 2:
		return 0
	case mp <= 4:
		return 1
	case mp <= 6:
		return 2
	case mp <= 9:
		return 3
	case mp <= 17:
		return 4
	case mp <= 24:
		return 5
	default:
		return 6
	}
}

// Max armor by tonnage. Format: HD, LA, RA, LT, RT, CT, LL, RL (IS values)
// Max armor = 2*IS per location, head capped at 9
// Standard BattleTech max armor formula:
// Total max armor = (tonnage * 2 * 16) / 10 + 9 for head
// Simplified: each point of IS gets 2 points of armor, head capped at 9.
// The standard formula is: max armor points = tonnage × 2 + 40 ... no.
// Actually: IS points per location are fixed by tonnage. Max armor = 2×IS per location.
// Easiest: use the universal formula. Total IS = tonnage÷10 rounded per TW table, then sum 2×IS per location + 9 for head.
// Simpler approach: just use the actual armor_total from the DB vs the theoretical max from IS.
// The IS total IS in the DB already. Max armor = 2 * IS_total + 3 (head gets 9 max but has 3 IS, so +6... actually head is special)
// 
// Correct formula: max armor = 2 × (IS_total - head_IS) + 9
// where head_IS = 3 for all mechs. So max armor = 2 × (IS_total - 3) + 9 = 2×IS_total - 6 + 9 = 2×IS_total + 3
func maxArmorFromIS(isTotal int) int {
	if isTotal <= 0 {
		return 0
	}
	// Head has 3 IS but max 9 armor (not 6). All other locations: max armor = 2 × IS.
	// So total max = 2*(isTotal - 3) + 9 = 2*isTotal + 3
	return 2*isTotal + 3
}

// 2d6 hit probability
var pHit = map[int]float64{
	2: 1.0, 3: 0.972, 4: 0.917, 5: 0.833, 6: 0.722,
	7: 0.583, 8: 0.417, 9: 0.278, 10: 0.167, 11: 0.083, 12: 0.028,
}

func hitProb(target int) float64 {
	if target <= 2 {
		return 1.0
	}
	if target >= 13 {
		return 0.0
	}
	return pHit[target]
}

type weaponInfo struct {
	Name           string
	ExpectedDamage float64
	Heat           int
	DamagePerHeat  float64
	MinRange       int
	ShortRange     int
	MediumRange    int
	LongRange      int
	ToHitModifier  int
	// For effective damage calc
	EffDamageShort  float64
	EffDamageMedium float64
	EffDamageLong   float64
	Quantity        int
	RackSize        int
	Type            string
}

func main() {
	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("DB: %v", err)
	}
	defer pool.Close()

	// Get all variants with stats
	rows, err := pool.Query(ctx, `
		SELECT vs.variant_id, vs.walk_mp, vs.run_mp, vs.jump_mp, 
			   vs.armor_total, vs.heat_sink_count, vs.heat_sink_type,
			   c.tonnage, COALESCE(vs.internal_structure_total, 0),
			   COALESCE(vs.has_targeting_computer, false)
		FROM variant_stats vs
		JOIN variants v ON v.id = vs.variant_id
		JOIN chassis c ON c.id = v.chassis_id`)
	if err != nil {
		log.Fatalf("Query: %v", err)
	}

	type variantData struct {
		ID            int
		WalkMP        int
		RunMP         int
		JumpMP        int
		ArmorTotal    int
		HeatSinkCount int
		HeatSinkType  string
		Tonnage       int
		ISTotal       int
		HasTC         bool
	}

	var variants []variantData
	for rows.Next() {
		var v variantData
		rows.Scan(&v.ID, &v.WalkMP, &v.RunMP, &v.JumpMP,
			&v.ArmorTotal, &v.HeatSinkCount, &v.HeatSinkType, &v.Tonnage, &v.ISTotal,
			&v.HasTC)
		variants = append(variants, v)
	}
	rows.Close()

	updated := 0
	for _, v := range variants {
		// TMM
		runTMM := tmmFromMP(v.RunMP)
		jumpTMM := 0
		if v.JumpMP > 0 {
			jumpTMM = tmmFromMP(v.JumpMP) + 1
		}
		tmm := runTMM
		if jumpTMM > tmm {
			tmm = jumpTMM
		}

		// Armor coverage — use actual IS total from the variant
		// Max armor = 2×IS per location + 6 extra for head (9 max armor vs 3 IS)
		// For non-standard configs (quads, tripods) this is approximate; cap at 100%
		maxArmor := maxArmorFromIS(v.ISTotal)
		armorPct := 0.0
		if maxArmor > 0 {
			armorPct = math.Round(float64(v.ArmorTotal)/float64(maxArmor)*10000) / 100
			if armorPct > 100.0 {
				armorPct = 100.0
			}
		}

		// Get weapons for this variant
		wRows, err := pool.Query(ctx, `
			SELECT e.name, e.expected_damage, e.heat, e.damage_per_heat, 
				   COALESCE(e.min_range,0), e.short_range, e.medium_range, e.long_range,
				   e.to_hit_modifier, e.effective_damage_short, 
				   e.effective_damage_medium, e.effective_damage_long,
				   ve.quantity, COALESCE(e.rack_size, 0), COALESCE(e.type, '')
			FROM variant_equipment ve
			JOIN equipment e ON e.id = ve.equipment_id
			WHERE ve.variant_id = $1`, v.ID)
		if err != nil {
			continue
		}

		var weapons []weaponInfo
		maxDamage := 0.0
		for wRows.Next() {
			var w weaponInfo
			wRows.Scan(&w.Name, &w.ExpectedDamage, &w.Heat, &w.DamagePerHeat,
				&w.MinRange, &w.ShortRange, &w.MediumRange, &w.LongRange,
				&w.ToHitModifier, &w.EffDamageShort, &w.EffDamageMedium, &w.EffDamageLong,
				&w.Quantity, &w.RackSize, &w.Type)
			// Targeting Computer: -1 to-hit for direct-fire weapons (energy, ballistic)
			if v.HasTC && (w.Type == "energy" || w.Type == "ballistic") {
				w.ToHitModifier -= 1
			}
			for i := 0; i < w.Quantity; i++ {
				weapons = append(weapons, w)
				maxDamage += w.ExpectedDamage
			}
		}
		wRows.Close()

		// Heat dissipation
		hsCapacity := v.HeatSinkCount
		hsLower := strings.ToLower(v.HeatSinkType)
		if strings.Contains(hsLower, "double") || strings.Contains(hsLower, "laser") {
			hsCapacity = v.HeatSinkCount * 2
		} else if strings.Contains(hsLower, "compact") {
			// Compact heat sinks dissipate 1 heat but weigh less
			hsCapacity = v.HeatSinkCount
		}

		// Movement heat: walking = 1 (matches game sim where both mechs walk)
		moveHeat := 1

		availableHeat := hsCapacity - moveHeat
		if availableHeat < 0 {
			availableHeat = 0
		}

		// Heat neutral damage: greedily pick weapons by damage_per_heat
		sort.Slice(weapons, func(i, j int) bool {
			return weapons[i].DamagePerHeat > weapons[j].DamagePerHeat
		})

		heatBudget := availableHeat
		heatNeutralDmg := 0.0
		for _, w := range weapons {
			if w.Heat == 0 {
				heatNeutralDmg += w.ExpectedDamage
				continue
			}
			if heatBudget >= w.Heat {
				heatNeutralDmg += w.ExpectedDamage
				heatBudget -= w.Heat
			}
		}

		// Heat neutral optimal range: evaluate each hex 1-30, find best
		bestRangeHex := 0
		bestDmg := 0.0
		for hex := 1; hex <= 30; hex++ {
			heatBudget = availableHeat
			dmg := 0.0
			type scoredWeapon struct {
				weaponInfo
				effDmg float64
				effDPH float64
			}
			var scored []scoredWeapon
			for _, w := range weapons {
				// Calculate effective damage at this exact hex
				ed := 0.0
				if hex > w.MinRange || w.MinRange == 0 {
					if hex <= w.ShortRange {
						ed = w.EffDamageShort
					} else if hex <= w.MediumRange {
						ed = w.EffDamageMedium
					} else if hex <= w.LongRange {
						ed = w.EffDamageLong
					}
				}
				if ed <= 0 {
					continue
				}
				dph := 0.0
				if w.Heat > 0 {
					dph = ed / float64(w.Heat)
				} else {
					dph = 999
				}
				scored = append(scored, scoredWeapon{w, ed, dph})
			}
			sort.Slice(scored, func(i, j int) bool {
				return scored[i].effDPH > scored[j].effDPH
			})
			for _, sw := range scored {
				if sw.Heat == 0 {
					dmg += sw.effDmg
					continue
				}
				if heatBudget >= sw.Heat {
					dmg += sw.effDmg
					heatBudget -= sw.Heat
				}
			}
			if dmg > bestDmg {
				bestDmg = dmg
				bestRangeHex = hex
			}
		}

		// Effective heat neutral damage at optimal range
		effHeatNeutralDmg := math.Round(bestDmg*100) / 100

		heatNeutralDmg = math.Round(heatNeutralDmg*100) / 100
		maxDamage = math.Round(maxDamage*100) / 100

		// 12-turn game simulation on 2 mapsheets (34 hex separation)
		// Ref opponent: 4/5 pilot in 4/6 mech (Hunchback 4P-style, medium lasers)
		// Ref tries to maintain optimal range of 6-8 hexes (MLas short range)
		// Subject moves intelligently to maximize damage output
		//
		// To-hit for subject: Gunnery 4 + movement_mod + ref_TMM + range_mod
		//   movement_mod: 0 if stood, 1 if walked
		//   ref_TMM: 0 if ref stood, +1 if ref walked (tmmFromMP(4)=+1)
		//
		// MMLs: evaluate both LRM mode (min 6, 7/14/21, rack×0.58 dmg)
		//       and SRM mode (no min, 3/6/9, rack×2×0.58 dmg), pick best per turn
		const (
			boardLength     = 34
			refOpponentWalk = 4
			refOptimalLow   = 6  // ref wants to be in this range band
			refOptimalHigh  = 8
			gameTurns       = 12
			gunnery         = 4
		)

		// Build sim weapons: for MMLs, we'll evaluate both modes dynamically
		type simWeapon struct {
			ExpectedDamage float64 // LRM mode (or normal)
			SRMDamage      float64 // SRM mode expected damage (0 if not MML)
			Heat           int
			MinRange       int
			ShortRange     int
			MediumRange    int
			LongRange      int
			SRMShort       int // SRM mode ranges
			SRMMedium      int
			SRMLong        int
			ToHitModifier  int
			RackSize       int
			IsMML          bool
			IsArtillery    bool
		}
		var simWeapons []simWeapon
		for _, w := range weapons {
			sw := simWeapon{
				ExpectedDamage: w.ExpectedDamage,
				Heat:           w.Heat,
				MinRange:       w.MinRange,
				ShortRange:     w.ShortRange,
				MediumRange:    w.MediumRange,
				LongRange:      w.LongRange,
				ToHitModifier:  w.ToHitModifier,
				IsArtillery:    w.Type == "artillery",
				RackSize:       w.RackSize,
			}
			nameUpper := strings.ToUpper(w.Name)
			if strings.Contains(nameUpper, "MML") && w.RackSize > 0 {
				sw.IsMML = true
				// LRM mode: rack × 0.58 (already in ExpectedDamage)
				// SRM mode: rack × 2 × 0.58 (SRMs do 2 damage per missile)
				sw.SRMDamage = float64(w.RackSize) * 2.0 * 0.58
				sw.SRMShort = 3
				sw.SRMMedium = 6
				sw.SRMLong = 9
			}
			simWeapons = append(simWeapons, sw)
		}

		// Compute heat-neutral damage at a given distance with given base target
		// refTMM is passed separately so artillery weapons can ignore it
		calcTurnDmg := func(dist int, baseTarget int, heatAvail int, refTMM int) float64 {
			type scored struct {
				effDmg float64
				heat   int
				dph    float64
			}
			var tw []scored
			for _, w := range simWeapons {
				bestED := 0.0

				if w.IsArtillery {
					// Artillery direct fire (Tac Ops pp. 150-153):
					// To-hit: gunnery + 4 + attacker_movement. NO range/target mods.
					// Hit: full damage (20 for Arrow IV), applied in 5-pt groups
					//   to random hit locations. All groups land — no cluster roll.
					// Miss: scatters 1D6 hexes in random direction.
					//   1 hex scatter (1/6 chance): impact adjacent to target,
					//   target takes adjacent damage (rackSize - 10).
					//   2+ hex scatter: target outside blast radius, 0 damage.
					//   Expected miss damage = (1/6) * (rackSize - 10)
					// Min range 6 hexes, max 17 hexes.
					if dist <= w.LongRange && dist > w.MinRange {
						target := baseTarget - refTMM + w.ToHitModifier
						pHit := hitProb(target)
						hitDmg := float64(w.RackSize)
						missDmg := float64(w.RackSize-10) / 6.0 // 1/6 chance of adjacent hit
						bestED = hitDmg*pHit + missDmg*(1.0-pHit)
					}
				} else if dist <= w.LongRange && w.LongRange > 0 {
					// Normal/LRM mode
					rangeMod := 0
					switch {
					case dist <= w.ShortRange:
						rangeMod = 0
					case dist <= w.MediumRange:
						rangeMod = 2
					default:
						rangeMod = 4
					}
					minRangePen := 0
					if w.MinRange > 0 && dist <= w.MinRange {
						minRangePen = w.MinRange - dist + 1
					}
					target := baseTarget + rangeMod + w.ToHitModifier + minRangePen
					bestED = w.ExpectedDamage * hitProb(target)
				}

				// MML SRM mode
				if w.IsMML && dist <= w.SRMLong {
					rangeMod := 0
					switch {
					case dist <= w.SRMShort:
						rangeMod = 0
					case dist <= w.SRMMedium:
						rangeMod = 2
					default:
						rangeMod = 4
					}
					target := baseTarget + rangeMod + w.ToHitModifier
					srmED := w.SRMDamage * hitProb(target)
					if srmED > bestED {
						bestED = srmED
					}
				}

				if bestED <= 0 {
					continue
				}
				dph := 0.0
				if w.Heat > 0 {
					dph = bestED / float64(w.Heat)
				} else {
					dph = 999.0
				}
				tw = append(tw, scored{bestED, w.Heat, dph})
			}
			sort.Slice(tw, func(i, j int) bool {
				return tw[i].dph > tw[j].dph
			})
			hb := heatAvail
			dmg := 0.0
			for _, w := range tw {
				if w.heat == 0 {
					dmg += w.effDmg
					continue
				}
				if hb >= w.heat {
					dmg += w.effDmg
					hb -= w.heat
				}
			}
			return dmg
		}

		// Heat available when walking (movement heat = 1) vs standing (movement heat = 0)
		heatWalking := hsCapacity - 1
		if heatWalking < 0 { heatWalking = 0 }
		heatStanding := hsCapacity // no movement heat
		if heatStanding < 0 { heatStanding = 0 }

		gameDmg := 0.0
		mechPos := 0           // subject starts at hex 0
		oppPos := boardLength  // opponent starts at hex 34
		for turn := 1; turn <= gameTurns; turn++ {
			curDist := oppPos - mechPos

			// Ref opponent moves first: tries to reach range 6-8
			refWalked := false
			if curDist > refOptimalHigh {
				// Too far, walk toward subject
				oppPos -= refOpponentWalk
				if oppPos < mechPos { oppPos = mechPos }
				refWalked = true
			} else if curDist < refOptimalLow {
				// Too close, walk away
				oppPos += refOpponentWalk
				if oppPos > boardLength { oppPos = boardLength }
				refWalked = true
			}
			// else: in optimal range, stand still

			// Ref TMM: +1 if walked, 0 if stood
			refTMM := 0
			if refWalked {
				refTMM = tmmFromMP(refOpponentWalk) // +1
			}

			// Subject evaluates 3 options:
			// Base to-hit = gunnery + movement_mod + ref_TMM
			// Walk: +1 movement, Stand: +0 movement
			baseWalked := gunnery + 1 + refTMM
			baseStood := gunnery + 0 + refTMM

			// Option 1: advance (walk toward)
			advPos := mechPos + v.WalkMP
			if advPos > oppPos { advPos = oppPos }
			advDist := oppPos - advPos
			if advDist < 1 { advDist = 1 }
			advDmg := calcTurnDmg(advDist, baseWalked, heatWalking, refTMM)

			// Option 2: stand still
			standDist := oppPos - mechPos
			if standDist < 1 { standDist = 1 }
			standDmg := calcTurnDmg(standDist, baseStood, heatStanding, refTMM)

			// Option 3: retreat (walk away)
			retPos := mechPos - v.WalkMP
			if retPos < 0 { retPos = 0 }
			retDist := oppPos - retPos
			if retDist < 1 { retDist = 1 }
			retDmg := calcTurnDmg(retDist, baseWalked, heatWalking, refTMM)

			// Pick best
			if advDmg >= standDmg && advDmg >= retDmg {
				gameDmg += advDmg
				mechPos = advPos
			} else if standDmg >= retDmg {
				gameDmg += standDmg
			} else {
				gameDmg += retDmg
				mechPos = retPos
			}
		}
		gameDmg = math.Round(gameDmg*100) / 100

		_, err = pool.Exec(ctx, `
			UPDATE variant_stats SET 
				tmm = $2, armor_coverage_pct = $3, heat_neutral_damage = $4,
				heat_neutral_range = $5, max_damage = $6, effective_heat_neutral_damage = $7,
				game_damage = $8
			WHERE variant_id = $1`,
			v.ID, tmm, armorPct, heatNeutralDmg, strconv.Itoa(bestRangeHex), maxDamage, effHeatNeutralDmg, gameDmg)
		if err != nil {
			log.Printf("Update %d: %v", v.ID, err)
			continue
		}
		updated++
	}

	fmt.Printf("Updated calculated stats for %d variants\n", updated)
}
