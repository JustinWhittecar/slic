package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type EquipmentHandler struct {
	DB *sql.DB
}

type EquipmentName struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (h *EquipmentHandler) Names(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var rows *sql.Rows
	var err error
	if q != "" {
		rows, err = h.DB.Query(`SELECT id, name, type FROM equipment WHERE name LIKE ? ORDER BY name LIMIT 50`, "%"+q+"%")
	} else {
		rows, err = h.DB.Query(`SELECT id, name, type FROM equipment ORDER BY name`)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	names := []EquipmentName{}
	for rows.Next() {
		var e EquipmentName
		rows.Scan(&e.ID, &e.Name, &e.Type)
		names = append(names, e)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(names)
}
