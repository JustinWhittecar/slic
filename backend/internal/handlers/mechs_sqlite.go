package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/JustinWhittecar/slic/internal/models"
)

type MechHandlerSQLite struct {
	DB *sql.DB
}

func (h *MechHandlerSQLite) List(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT v.id, v.model_code, v.name, c.name, COALESCE(c.alternate_name,''), COALESCE(vs.tonnage, c.tonnage), c.tech_base,
		       v.battle_value, v.intro_year, COALESCE(v.era,''), COALESCE(v.role,''),
		       COALESCE(vs.tmm,0), COALESCE(vs.armor_coverage_pct,0), COALESCE(vs.heat_neutral_damage,0),
		       COALESCE(vs.walk_mp,0), COALESCE(vs.jump_mp,0), COALESCE(vs.armor_total,0),
		       COALESCE(vs.max_damage,0),
		       COALESCE(vs.effective_heat_neutral_damage,0), COALESCE(vs.heat_neutral_range,''),
		       COALESCE(vs.game_damage,0),
		       COALESCE(vs.engine_type,''), COALESCE(vs.engine_rating,0),
		       COALESCE(vs.heat_sink_count,0), COALESCE(vs.heat_sink_type,''),
		       COALESCE(vs.run_mp,0),
		       COALESCE(v.rules_level,0), COALESCE(v.source,''), COALESCE(v.config,''),
		       COALESCE(vs.combat_rating,0)
		FROM variants v
		JOIN chassis c ON c.id = v.chassis_id
		LEFT JOIN variant_stats vs ON vs.variant_id = v.id
		WHERE v.mul_id IS NOT NULL AND v.mul_id > 0 AND v.battle_value > 0`

	args := []any{}

	if v := r.URL.Query().Get("tonnage_min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND COALESCE(vs.tonnage, c.tonnage) >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("tonnage_max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND COALESCE(vs.tonnage, c.tonnage) <= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("era"); v != "" {
		query += " AND v.intro_year <= COALESCE((SELECT end_year FROM eras WHERE name = ?), 9999)"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("role"); v != "" {
		query += " AND v.role = ?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("name"); v != "" {
		query += " AND (v.name LIKE ? OR c.name LIKE ? OR c.alternate_name LIKE ?)"
		p := "%" + v + "%"
		args = append(args, p, p, p)
	}
	if v := r.URL.Query().Get("bv_min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND v.battle_value >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("bv_max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND v.battle_value <= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("tmm_min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND vs.tmm >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("armor_pct_min"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			query += " AND vs.armor_coverage_pct >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("heat_neutral_min"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			query += " AND vs.heat_neutral_damage >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("max_damage_min"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			query += " AND vs.max_damage >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("game_damage_min"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			query += " AND vs.game_damage >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("combat_rating_min"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			query += " AND vs.combat_rating >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("combat_rating_max"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			query += " AND vs.combat_rating <= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("intro_year_min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND v.intro_year >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("intro_year_max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND v.intro_year <= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("walk_mp_min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND vs.walk_mp >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("jump_mp_min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND vs.jump_mp >= ?"
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("heat_sink_type"); v != "" {
		query += " AND vs.heat_sink_type = ?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("engine_types"); v != "" {
		types := strings.Split(v, ",")
		clauses := []string{}
		for _, t := range types {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			switch t {
			case "ICE":
				clauses = append(clauses, "(vs.engine_type LIKE ? OR vs.engine_type LIKE ?)")
				args = append(args, "%ICE%", "%I.C.E%")
			case "Fuel Cell":
				clauses = append(clauses, "(vs.engine_type LIKE ? OR vs.engine_type LIKE ?)")
				args = append(args, "%Fuel Cell%", "%Fuel-Cell%")
			case "Fusion":
				clauses = append(clauses, "(vs.engine_type LIKE ? AND vs.engine_type NOT LIKE ? AND vs.engine_type NOT LIKE ? AND vs.engine_type NOT LIKE ? AND vs.engine_type NOT LIKE ? AND vs.engine_type NOT LIKE ?)")
				args = append(args, "%Fusion%", "%XL%", "%XXL%", "%Light%", "%Compact%", "%Primitive%")
			case "XL":
				clauses = append(clauses, "(vs.engine_type LIKE ? AND vs.engine_type NOT LIKE ?)")
				args = append(args, "%XL%", "%XXL%")
			default:
				clauses = append(clauses, "vs.engine_type LIKE ?")
				args = append(args, "%"+t+"%")
			}
		}
		if len(clauses) > 0 {
			query += " AND (" + strings.Join(clauses, " OR ") + ")"
		}
	}
	if v := r.URL.Query().Get("tech_base"); v != "" {
		query += " AND c.tech_base = ?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("faction"); v != "" {
		query += ` AND EXISTS (
			SELECT 1 FROM variant_era_factions vef
			JOIN factions f ON f.id = vef.faction_id
			WHERE vef.variant_id = v.id AND (f.abbreviation = ? OR f.name = ?))`
		args = append(args, v, v)
	}

	query += " ORDER BY COALESCE(vs.tonnage, c.tonnage), c.name, v.model_code LIMIT 5000"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	mechs := []models.MechListItem{}
	for rows.Next() {
		var m models.MechListItem
		if err := rows.Scan(&m.ID, &m.ModelCode, &m.Name, &m.Chassis, &m.AlternateName, &m.Tonnage,
			&m.TechBase, &m.BV, &m.IntroYear, &m.Era, &m.Role,
			&m.TMM, &m.ArmorCoveragePct, &m.HeatNeutralDamage,
			&m.WalkMP, &m.JumpMP, &m.ArmorTotal, &m.MaxDamage,
			&m.EffHeatNeutralDamage, &m.HeatNeutralRange,
			&m.GameDamage,
			&m.EngineType, &m.EngineRating,
			&m.HeatSinkCount, &m.HeatSinkType,
			&m.RunMP, &m.RulesLevel, &m.Source, &m.Config,
			&m.CombatRating); err != nil {
			http.Error(w, "scan error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		mechs = append(mechs, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mechs)
}

func (h *MechHandlerSQLite) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var m models.MechDetail
	err = h.DB.QueryRow(`
		SELECT v.id, v.model_code, v.name, c.name, COALESCE(c.alternate_name,''), COALESCE(vs2.tonnage, c.tonnage), c.tech_base,
		       v.battle_value, v.intro_year, COALESCE(v.era,''), COALESCE(v.role,''), COALESCE(c.sarna_url,''),
		       COALESCE(vs2.game_damage, 0), COALESCE(vs2.combat_rating, 0)
		FROM variants v
		JOIN chassis c ON c.id = v.chassis_id
		LEFT JOIN variant_stats vs2 ON vs2.variant_id = v.id
		WHERE v.id = ?`, id).Scan(
		&m.ID, &m.ModelCode, &m.Name, &m.Chassis, &m.AlternateName, &m.Tonnage, &m.TechBase,
		&m.BV, &m.IntroYear, &m.Era, &m.Role, &m.SarnaURL,
		&m.GameDamage, &m.CombatRating)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Load stats
	var stats models.VariantStats
	err = h.DB.QueryRow(`
		SELECT walk_mp, run_mp, jump_mp, armor_total, internal_structure_total,
		       heat_sink_count, heat_sink_type, engine_type, engine_rating,
		       COALESCE(cockpit_type,''), COALESCE(gyro_type,''), COALESCE(myomer_type,''),
		       COALESCE(structure_type,''), COALESCE(armor_type,''),
		       COALESCE(tmm,0), COALESCE(armor_coverage_pct,0), COALESCE(heat_neutral_damage,0),
		       COALESCE(heat_neutral_range,''), COALESCE(max_damage,0), COALESCE(effective_heat_neutral_damage,0),
		       COALESCE(has_targeting_computer, 0),
		       COALESCE(combat_rating, 0), COALESCE(offense_turns, 0), COALESCE(defense_turns, 0)
		FROM variant_stats WHERE variant_id = ?`, id).Scan(
		&stats.WalkMP, &stats.RunMP, &stats.JumpMP, &stats.ArmorTotal, &stats.ISTotal,
		&stats.HeatSinkCount, &stats.HeatSinkType, &stats.EngineType, &stats.EngineRating,
		&stats.CockpitType, &stats.GyroType, &stats.MyomerType,
		&stats.StructureType, &stats.ArmorType,
		&stats.TMM, &stats.ArmorCoveragePct, &stats.HeatNeutralDamage,
		&stats.HeatNeutralRange, &stats.MaxDamage, &stats.EffHeatNeutralDamage,
		&stats.HasTargetingComputer,
		&stats.CombatRating, &stats.OffenseTurns, &stats.DefenseTurns)
	if err == nil {
		m.Stats = &stats
	}

	// Generate sourcing links
	sarnaName := m.Chassis
	if idx := strings.Index(sarnaName, "("); idx > 0 {
		sarnaName = strings.TrimSpace(sarnaName[:idx])
	}
	chassisURL := strings.ReplaceAll(sarnaName, " ", "_")
	m.SarnaURL = "https://www.sarna.net/wiki/" + chassisURL
	m.IWMUrl = "https://www.ironwindmetals.com/index.php/product-listing?cid=0&searchword=" + strings.ToLower(strings.ReplaceAll(m.Chassis, " ", "+"))
	m.CatalystUrl = "https://store.catalystgamelabs.com/search?q=" + strings.ReplaceAll(m.Chassis, " ", "+")

	// Load equipment
	eqRows, err := h.DB.Query(`
		SELECT e.id, e.name, e.type, e.damage, e.heat, e.min_range,
		       e.short_range, e.medium_range, e.long_range, COALESCE(e.extreme_range,0),
		       e.tonnage, e.slots,
		       COALESCE(e.internal_name,''), e.bv, COALESCE(e.rack_size,0),
		       COALESCE(e.expected_damage,0), COALESCE(e.damage_per_ton,0), COALESCE(e.damage_per_heat,0),
		       COALESCE(e.to_hit_modifier,0),
		       COALESCE(e.effective_damage_short,0), COALESCE(e.effective_damage_medium,0), COALESCE(e.effective_damage_long,0),
		       COALESCE(e.effective_dps_ton,0), COALESCE(e.effective_dps_heat,0),
		       ve.location, ve.quantity
		FROM variant_equipment ve
		JOIN equipment e ON e.id = ve.equipment_id
		WHERE ve.variant_id = ?
		ORDER BY ve.location, e.name`, id)
	if err == nil {
		defer eqRows.Close()
		for eqRows.Next() {
			var eq models.VariantEquipment
			eqRows.Scan(&eq.ID, &eq.Name, &eq.Type, &eq.Damage, &eq.Heat, &eq.MinRange,
				&eq.ShortRange, &eq.MediumRange, &eq.LongRange, &eq.ExtremeRange,
				&eq.Tonnage, &eq.Slots,
				&eq.InternalName, &eq.BV, &eq.RackSize,
				&eq.ExpectedDamage, &eq.DamagePerTon, &eq.DamagePerHeat,
				&eq.ToHitModifier,
				&eq.EffDamageShort, &eq.EffDamageMedium, &eq.EffDamageLong,
				&eq.EffDPSTon, &eq.EffDPSHeat,
				&eq.Location, &eq.Quantity)
			m.Equipment = append(m.Equipment, eq)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}
