package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type PreferencesHandler struct {
	DB *sql.DB
}

type UserPreferences struct {
	ColumnVisibility json.RawMessage `json:"column_visibility,omitempty"`
	ColumnOrder      json.RawMessage `json:"column_order,omitempty"`
	ActiveFilters    json.RawMessage `json:"active_filters,omitempty"`
	UpdatedAt        string          `json:"updated_at,omitempty"`
}

func (h *PreferencesHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var prefs UserPreferences
	var cv, co, af sql.NullString
	err := h.DB.QueryRow(
		`SELECT column_visibility, column_order, active_filters, updated_at FROM user_preferences WHERE user_id = ?`,
		user.ID,
	).Scan(&cv, &co, &af, &prefs.UpdatedAt)

	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if cv.Valid {
		prefs.ColumnVisibility = json.RawMessage(cv.String)
	}
	if co.Valid {
		prefs.ColumnOrder = json.RawMessage(co.String)
	}
	if af.Valid {
		prefs.ActiveFilters = json.RawMessage(af.String)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}

func (h *PreferencesHandler) Put(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var prefs UserPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var cvStr, coStr, afStr sql.NullString
	if len(prefs.ColumnVisibility) > 0 {
		cvStr = sql.NullString{String: string(prefs.ColumnVisibility), Valid: true}
	}
	if len(prefs.ColumnOrder) > 0 {
		coStr = sql.NullString{String: string(prefs.ColumnOrder), Valid: true}
	}
	if len(prefs.ActiveFilters) > 0 {
		afStr = sql.NullString{String: string(prefs.ActiveFilters), Valid: true}
	}

	_, err := h.DB.Exec(
		`INSERT INTO user_preferences (user_id, column_visibility, column_order, active_filters, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   column_visibility = excluded.column_visibility,
		   column_order = excluded.column_order,
		   active_filters = excluded.active_filters,
		   updated_at = excluded.updated_at`,
		user.ID, cvStr, coStr, afStr, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

func (h *PreferencesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.DB.Exec(`DELETE FROM user_preferences WHERE user_id = ?`, user.ID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}
