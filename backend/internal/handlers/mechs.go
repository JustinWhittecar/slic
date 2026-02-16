package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/JustinWhittecar/slic/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MechHandler struct {
	DB *pgxpool.Pool
}

func (h *MechHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := `
		SELECT v.id, v.model_code, v.name, c.name, c.tonnage, c.tech_base,
		       v.battle_value, v.intro_year, v.era, v.role
		FROM variants v
		JOIN chassis c ON c.id = v.chassis_id
		WHERE 1=1`

	args := []any{}
	argN := 0

	nextArg := func() string {
		argN++
		return "$" + strconv.Itoa(argN)
	}

	if v := r.URL.Query().Get("tonnage_min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND c.tonnage >= " + nextArg()
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("tonnage_max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " AND c.tonnage <= " + nextArg()
			args = append(args, n)
		}
	}
	if v := r.URL.Query().Get("era"); v != "" {
		query += " AND v.era = " + nextArg()
		args = append(args, v)
	}
	if v := r.URL.Query().Get("role"); v != "" {
		query += " AND v.role = " + nextArg()
		args = append(args, v)
	}
	if v := r.URL.Query().Get("name"); v != "" {
		query += " AND (v.name ILIKE " + nextArg() + " OR c.name ILIKE " + nextArg() + ")"
		args = append(args, "%"+v+"%", "%"+v+"%")
		argN++ // extra arg
	}
	if v := r.URL.Query().Get("faction"); v != "" {
		query += ` AND EXISTS (
			SELECT 1 FROM variant_era_factions vef
			JOIN factions f ON f.id = vef.faction_id
			WHERE vef.variant_id = v.id AND (f.abbreviation = ` + nextArg() + ` OR f.name = ` + nextArg() + `))`
		args = append(args, v, v)
		argN++
	}

	query += " ORDER BY c.tonnage, c.name, v.model_code LIMIT 200"

	rows, err := h.DB.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, "query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	mechs := []models.MechListItem{}
	for rows.Next() {
		var m models.MechListItem
		if err := rows.Scan(&m.ID, &m.ModelCode, &m.Name, &m.Chassis, &m.Tonnage,
			&m.TechBase, &m.BV, &m.IntroYear, &m.Era, &m.Role); err != nil {
			http.Error(w, "scan error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		mechs = append(mechs, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mechs)
}

func (h *MechHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var m models.MechDetail
	err = h.DB.QueryRow(ctx, `
		SELECT v.id, v.model_code, v.name, c.name, c.tonnage, c.tech_base,
		       v.battle_value, v.intro_year, v.era, v.role, c.sarna_url
		FROM variants v
		JOIN chassis c ON c.id = v.chassis_id
		WHERE v.id = $1`, id).Scan(
		&m.ID, &m.ModelCode, &m.Name, &m.Chassis, &m.Tonnage, &m.TechBase,
		&m.BV, &m.IntroYear, &m.Era, &m.Role, &m.SarnaURL)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Load stats
	var stats models.VariantStats
	err = h.DB.QueryRow(ctx, `
		SELECT walk_mp, run_mp, jump_mp, armor_total, internal_structure_total,
		       heat_sink_count, heat_sink_type, engine_type, engine_rating
		FROM variant_stats WHERE variant_id = $1`, id).Scan(
		&stats.WalkMP, &stats.RunMP, &stats.JumpMP, &stats.ArmorTotal, &stats.ISTotal,
		&stats.HeatSinkCount, &stats.HeatSinkType, &stats.EngineType, &stats.EngineRating)
	if err == nil {
		m.Stats = &stats
	}

	// Load equipment
	rows, err := h.DB.Query(ctx, `
		SELECT e.id, e.name, e.type, e.damage, e.heat, e.min_range,
		       e.short_range, e.medium_range, e.long_range, e.tonnage, e.slots,
		       ve.location, ve.quantity
		FROM variant_equipment ve
		JOIN equipment e ON e.id = ve.equipment_id
		WHERE ve.variant_id = $1
		ORDER BY ve.location, e.name`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var eq models.VariantEquipment
			rows.Scan(&eq.ID, &eq.Name, &eq.Type, &eq.Damage, &eq.Heat, &eq.MinRange,
				&eq.ShortRange, &eq.MediumRange, &eq.LongRange, &eq.Tonnage, &eq.Slots,
				&eq.Location, &eq.Quantity)
			m.Equipment = append(m.Equipment, eq)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}
