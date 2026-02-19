package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
)

type ModelsHandler struct {
	DB *sql.DB // mech DB (read-only)
}

type PhysicalModel struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	SKU          string `json:"sku,omitempty"`
	SourceURL    string `json:"source_url,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
	InPrint      bool   `json:"in_print"`
}

type ChassisModels struct {
	ChassisID   int             `json:"chassis_id"`
	ChassisName string          `json:"chassis_name"`
	Tonnage     int             `json:"tonnage"`
	TechBase    string          `json:"tech_base"`
	HasModel    bool            `json:"has_model"`
	Models      []PhysicalModel `json:"models"`
}

func (h *ModelsHandler) List(w http.ResponseWriter, r *http.Request) {
	query := `SELECT pm.id, pm.chassis_id, pm.name, pm.manufacturer, COALESCE(pm.sku,''),
	                 COALESCE(pm.source_url,''), COALESCE(pm.image_url,''), COALESCE(pm.in_print, 1),
	                 c.name, c.tonnage, c.tech_base
	          FROM physical_models pm
	          JOIN chassis c ON c.id = pm.chassis_id`
	args := []any{}

	if v := r.URL.Query().Get("chassis_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			query += " WHERE pm.chassis_id = ?"
			args = append(args, n)
		}
	}

	query += " ORDER BY c.tonnage, c.name, pm.manufacturer, pm.name"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	chassisMap := map[int]*ChassisModels{}
	order := []int{}

	for rows.Next() {
		var pm PhysicalModel
		var chassisID, tonnage int
		var chassisName, techBase string
		var inPrint int

		rows.Scan(&pm.ID, &chassisID, &pm.Name, &pm.Manufacturer, &pm.SKU,
			&pm.SourceURL, &pm.ImageURL, &inPrint,
			&chassisName, &tonnage, &techBase)
		pm.InPrint = inPrint != 0

		cm, ok := chassisMap[chassisID]
		if !ok {
			cm = &ChassisModels{
				ChassisID:   chassisID,
				ChassisName: chassisName,
				Tonnage:     tonnage,
				TechBase:    techBase,
				HasModel:    true,
				Models:      []PhysicalModel{},
			}
			chassisMap[chassisID] = cm
			order = append(order, chassisID)
		}
		cm.Models = append(cm.Models, pm)
	}

	result := make([]ChassisModels, 0, len(order))
	for _, id := range order {
		result = append(result, *chassisMap[id])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
