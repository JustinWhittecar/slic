package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
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
			   c.tonnage, COALESCE(vs.internal_structure_total, 0)
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
	}

	var variants []variantData
	for rows.Next() {
		var v variantData
		rows.Scan(&v.ID, &v.WalkMP, &v.RunMP, &v.JumpMP,
			&v.ArmorTotal, &v.HeatSinkCount, &v.HeatSinkType, &v.Tonnage, &v.ISTotal)
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
			SELECT e.expected_damage, e.heat, e.damage_per_heat, 
				   COALESCE(e.min_range,0), e.short_range, e.medium_range, e.long_range,
				   e.to_hit_modifier, e.effective_damage_short, 
				   e.effective_damage_medium, e.effective_damage_long,
				   ve.quantity
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
			wRows.Scan(&w.ExpectedDamage, &w.Heat, &w.DamagePerHeat,
				&w.MinRange, &w.ShortRange, &w.MediumRange, &w.LongRange,
				&w.ToHitModifier, &w.EffDamageShort, &w.EffDamageMedium, &w.EffDamageLong,
				&w.Quantity)
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

		// Heat neutral optimal range: check short/medium/long
		bestRange := "medium"
		bestDmg := 0.0
		for _, rng := range []string{"short", "medium", "long"} {
			heatBudget = availableHeat
			dmg := 0.0
			// Sort by effective DPH at this range
			type scoredWeapon struct {
				weaponInfo
				effDmg float64
				effDPH float64
			}
			var scored []scoredWeapon
			for _, w := range weapons {
				var ed float64
				switch rng {
				case "short":
					ed = w.EffDamageShort
				case "medium":
					ed = w.EffDamageMedium
				case "long":
					ed = w.EffDamageLong
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
				bestRange = rng
			}
		}

		// Effective heat neutral damage at optimal range
		effHeatNeutralDmg := math.Round(bestDmg*100) / 100

		heatNeutralDmg = math.Round(heatNeutralDmg*100) / 100
		maxDamage = math.Round(maxDamage*100) / 100

		// 12-turn game simulation on 2 mapsheets (34 hex separation)
		// Both mechs walk toward each other. Ref opponent: walk 4.
		// Fire after movement each turn.
		const (
			boardLength   = 34
			refOpponentMP = 4
			gameTurns     = 12
			baseTarget    = 7 // gunnery 4 + attacker walked (+1) + target TMM +2 (+2) = 4+1+2=7
		)
		gameDmg := 0.0
		for turn := 1; turn <= gameTurns; turn++ {
			dist := boardLength - (v.WalkMP+refOpponentMP)*turn
			if dist < 1 {
				dist = 1
			}

			// Score each weapon at this distance
			type turnWeapon struct {
				expectedDmg float64
				effDmg      float64
				heat        int
				effDPH      float64
			}
			var tw []turnWeapon
			for _, w := range weapons {
				// Out of range?
				if dist > w.LongRange || w.LongRange == 0 {
					continue
				}
				// Range modifier
				rangeMod := 0
				switch {
				case dist <= w.ShortRange:
					rangeMod = 0
				case dist <= w.MediumRange:
					rangeMod = 2
				default:
					rangeMod = 4
				}
				// Min range penalty
				minRangePen := 0
				if w.MinRange > 0 && dist < w.MinRange {
					minRangePen = w.MinRange - dist
				}
				target := baseTarget + rangeMod + w.ToHitModifier + minRangePen
				prob := hitProb(target)
				ed := w.ExpectedDamage * prob
				dph := 0.0
				if w.Heat > 0 {
					dph = ed / float64(w.Heat)
				} else {
					dph = 999.0
				}
				tw = append(tw, turnWeapon{w.ExpectedDamage, ed, w.Heat, dph})
			}

			// Greedy heat-neutral selection
			sort.Slice(tw, func(i, j int) bool {
				return tw[i].effDPH > tw[j].effDPH
			})
			heatBudget = availableHeat
			for _, w := range tw {
				if w.heat == 0 {
					gameDmg += w.effDmg
					continue
				}
				if heatBudget >= w.heat {
					gameDmg += w.effDmg
					heatBudget -= w.heat
				}
			}
		}
		gameDmg = math.Round(gameDmg*100) / 100

		_, err = pool.Exec(ctx, `
			UPDATE variant_stats SET 
				tmm = $2, armor_coverage_pct = $3, heat_neutral_damage = $4,
				heat_neutral_range = $5, max_damage = $6, effective_heat_neutral_damage = $7,
				game_damage = $8
			WHERE variant_id = $1`,
			v.ID, tmm, armorPct, heatNeutralDmg, bestRange, maxDamage, effHeatNeutralDmg, gameDmg)
		if err != nil {
			log.Printf("Update %d: %v", v.ID, err)
			continue
		}
		updated++
	}

	fmt.Printf("Updated calculated stats for %d variants\n", updated)
}
