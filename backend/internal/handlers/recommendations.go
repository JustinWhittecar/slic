package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/JustinWhittecar/slic/internal/models"
)

type RecommendationsHandler struct {
	DB *sql.DB
}

func (h *RecommendationsHandler) Recommend(w http.ResponseWriter, r *http.Request) {
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

	// Budget filter (required for meaningful recommendations)
	if v := r.URL.Query().Get("budget"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			query += " AND v.battle_value <= ?"
			args = append(args, n)
		}
	}

	// Tech base filter
	if v := r.URL.Query().Get("tech_base"); v != "" && v != "All" {
		query += " AND c.tech_base = ?"
		args = append(args, v)
	}

	// Weight class filter
	if v := r.URL.Query().Get("weight_class"); v != "" && v != "All" {
		switch v {
		case "Light":
			query += " AND COALESCE(vs.tonnage, c.tonnage) BETWEEN 20 AND 35"
		case "Medium":
			query += " AND COALESCE(vs.tonnage, c.tonnage) BETWEEN 40 AND 55"
		case "Heavy":
			query += " AND COALESCE(vs.tonnage, c.tonnage) BETWEEN 60 AND 75"
		case "Assault":
			query += " AND COALESCE(vs.tonnage, c.tonnage) BETWEEN 80 AND 100"
		}
	}

	// Exclude IDs already in the list
	if v := r.URL.Query().Get("exclude"); v != "" {
		idStrs := strings.Split(v, ",")
		placeholders := []string{}
		for _, s := range idStrs {
			s = strings.TrimSpace(s)
			if n, err := strconv.Atoi(s); err == nil {
				placeholders = append(placeholders, "?")
				args = append(args, n)
			}
		}
		if len(placeholders) > 0 {
			query += " AND v.id NOT IN (" + strings.Join(placeholders, ",") + ")"
		}
	}

	// Order by combat rating descending, limit results
	query += " ORDER BY COALESCE(vs.combat_rating, 0) DESC"

	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	query += " LIMIT ?"
	args = append(args, limit)

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
