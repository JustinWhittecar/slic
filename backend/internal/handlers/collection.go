package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
)

type CollectionHandler struct {
	DB    *sql.DB // user DB (writable)
	MecDB *sql.DB // mech DB (read-only, has physical_models)
}

type CollectionItem struct {
	ID              int64  `json:"id"`
	PhysicalModelID int    `json:"physical_model_id"`
	Quantity        int    `json:"quantity"`
	Notes           string `json:"notes"`
	ModelName       string `json:"model_name"`
	Manufacturer    string `json:"manufacturer"`
	SKU             string `json:"sku"`
	SourceURL       string `json:"source_url,omitempty"`
	ChassisID       int    `json:"chassis_id"`
	ChassisName     string `json:"chassis_name"`
	Tonnage         int    `json:"tonnage"`
}

type CollectionSummary struct {
	ChassisID     int    `json:"chassis_id"`
	ChassisName   string `json:"chassis_name"`
	TotalQuantity int    `json:"total_quantity"`
}

func (h *CollectionHandler) List(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	rows, err := h.DB.Query(
		`SELECT id, physical_model_id, quantity, COALESCE(notes,'') FROM user_collections WHERE user_id = ? ORDER BY physical_model_id`,
		user.ID,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []CollectionItem{}
	for rows.Next() {
		var item CollectionItem
		rows.Scan(&item.ID, &item.PhysicalModelID, &item.Quantity, &item.Notes)

		// Enrich from mech DB
		h.MecDB.QueryRow(
			`SELECT pm.name, pm.manufacturer, COALESCE(pm.sku,''), COALESCE(pm.source_url,''),
			        c.id, c.name, c.tonnage
			 FROM physical_models pm
			 JOIN chassis c ON c.id = pm.chassis_id
			 WHERE pm.id = ?`, item.PhysicalModelID,
		).Scan(&item.ModelName, &item.Manufacturer, &item.SKU, &item.SourceURL,
			&item.ChassisID, &item.ChassisName, &item.Tonnage)

		items = append(items, item)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *CollectionHandler) Put(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	modelID, err := strconv.Atoi(r.PathValue("modelId"))
	if err != nil {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Quantity int    `json:"quantity"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		// quantity 0 or negative = remove
		h.DB.Exec(`DELETE FROM user_collections WHERE user_id = ? AND physical_model_id = ?`, user.ID, modelID)
		w.WriteHeader(http.StatusOK)
		return
	}

	_, err = h.DB.Exec(
		`INSERT INTO user_collections (user_id, physical_model_id, quantity, notes) VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, physical_model_id) DO UPDATE SET quantity=excluded.quantity, notes=excluded.notes`,
		user.ID, modelID, req.Quantity, req.Notes,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *CollectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	modelID, err := strconv.Atoi(r.PathValue("modelId"))
	if err != nil {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	h.DB.Exec(`DELETE FROM user_collections WHERE user_id = ? AND physical_model_id = ?`, user.ID, modelID)
	w.WriteHeader(http.StatusOK)
}

func (h *CollectionHandler) Summary(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	rows, err := h.DB.Query(
		`SELECT physical_model_id, quantity FROM user_collections WHERE user_id = ?`,
		user.ID,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Group by chassis
	chassisMap := map[int]*CollectionSummary{}
	for rows.Next() {
		var pmID, qty int
		rows.Scan(&pmID, &qty)

		var chassisID int
		var chassisName string
		h.MecDB.QueryRow(
			`SELECT c.id, c.name FROM physical_models pm JOIN chassis c ON c.id = pm.chassis_id WHERE pm.id = ?`,
			pmID,
		).Scan(&chassisID, &chassisName)

		if chassisID == 0 {
			continue
		}
		if s, ok := chassisMap[chassisID]; ok {
			s.TotalQuantity += qty
		} else {
			chassisMap[chassisID] = &CollectionSummary{
				ChassisID:     chassisID,
				ChassisName:   chassisName,
				TotalQuantity: qty,
			}
		}
	}

	summaries := make([]CollectionSummary, 0, len(chassisMap))
	for _, s := range chassisMap {
		summaries = append(summaries, *s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}
