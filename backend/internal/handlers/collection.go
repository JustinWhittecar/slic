package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
)

type CollectionHandler struct {
	DB *sql.DB
}

type CollectionItem struct {
	ID        int64  `json:"id"`
	ChassisID int    `json:"chassis_id"`
	Quantity  int    `json:"quantity"`
	Notes     string `json:"notes"`
}

func (h *CollectionHandler) List(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	rows, err := h.DB.Query(
		`SELECT id, chassis_id, quantity, COALESCE(notes,'') FROM user_collections WHERE user_id = ? ORDER BY chassis_id`,
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
		rows.Scan(&item.ID, &item.ChassisID, &item.Quantity, &item.Notes)
		items = append(items, item)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *CollectionHandler) Put(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	chassisID, err := strconv.Atoi(r.PathValue("chassisId"))
	if err != nil {
		http.Error(w, "Invalid chassis ID", http.StatusBadRequest)
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
		req.Quantity = 1
	}

	_, err = h.DB.Exec(
		`INSERT INTO user_collections (user_id, chassis_id, quantity, notes) VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, chassis_id) DO UPDATE SET quantity=excluded.quantity, notes=excluded.notes`,
		user.ID, chassisID, req.Quantity, req.Notes,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *CollectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	chassisID, err := strconv.Atoi(r.PathValue("chassisId"))
	if err != nil {
		http.Error(w, "Invalid chassis ID", http.StatusBadRequest)
		return
	}

	h.DB.Exec(`DELETE FROM user_collections WHERE user_id = ? AND chassis_id = ?`, user.ID, chassisID)
	w.WriteHeader(http.StatusOK)
}
