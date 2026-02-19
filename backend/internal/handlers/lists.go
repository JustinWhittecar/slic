package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
)

type ListsHandler struct {
	DB *sql.DB
}

type UserList struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	Budget    int             `json:"budget"`
	ShareCode string          `json:"share_code,omitempty"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	Entries   []UserListEntry `json:"entries,omitempty"`
}

type UserListEntry struct {
	ID        int64 `json:"id"`
	VariantID int   `json:"variant_id"`
	Gunnery   int   `json:"gunnery"`
	Piloting  int   `json:"piloting"`
}

func (h *ListsHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	rows, err := h.DB.Query(
		`SELECT id, name, budget, COALESCE(share_code,''), created_at, updated_at FROM user_lists WHERE user_id = ? ORDER BY updated_at DESC`,
		user.ID,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	lists := []UserList{}
	for rows.Next() {
		var l UserList
		rows.Scan(&l.ID, &l.Name, &l.Budget, &l.ShareCode, &l.CreatedAt, &l.UpdatedAt)
		lists = append(lists, l)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lists)
}

func (h *ListsHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	var req struct {
		Name   string `json:"name"`
		Budget int    `json:"budget"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "Untitled List"
	}
	if req.Budget <= 0 {
		req.Budget = 7000
	}

	res, err := h.DB.Exec(`INSERT INTO user_lists (user_id, name, budget) VALUES (?, ?, ?)`,
		user.ID, req.Name, req.Budget)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func (h *ListsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var l UserList
	var ownerID int64
	err = h.DB.QueryRow(
		`SELECT id, user_id, name, budget, COALESCE(share_code,''), created_at, updated_at FROM user_lists WHERE id = ?`, id,
	).Scan(&l.ID, &ownerID, &l.Name, &l.Budget, &l.ShareCode, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Check access
	user := UserFromContext(r.Context())
	shareCode := r.URL.Query().Get("share_code")
	if (user == nil || user.ID != ownerID) && (l.ShareCode == "" || l.ShareCode != shareCode) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Load entries
	rows, err := h.DB.Query(`SELECT id, variant_id, gunnery, piloting FROM user_list_entries WHERE list_id = ?`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e UserListEntry
			rows.Scan(&e.ID, &e.VariantID, &e.Gunnery, &e.Piloting)
			l.Entries = append(l.Entries, e)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l)
}

func (h *ListsHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Verify ownership
	var ownerID int64
	if err := h.DB.QueryRow(`SELECT user_id FROM user_lists WHERE id = ?`, id).Scan(&ownerID); err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if user.ID != ownerID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		Name      string           `json:"name"`
		Budget    int              `json:"budget"`
		ShareCode *string          `json:"share_code"` // null = generate, "" = remove
		Entries   *[]UserListEntry `json:"entries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name != "" {
		h.DB.Exec(`UPDATE user_lists SET name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, req.Name, id)
	}
	if req.Budget > 0 {
		h.DB.Exec(`UPDATE user_lists SET budget=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, req.Budget, id)
	}
	if req.ShareCode != nil {
		sc := *req.ShareCode
		if sc == "" {
			// Generate
			b := make([]byte, 6)
			rand.Read(b)
			sc = hex.EncodeToString(b)
		}
		h.DB.Exec(`UPDATE user_lists SET share_code=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sc, id)
	}

	if req.Entries != nil {
		h.DB.Exec(`DELETE FROM user_list_entries WHERE list_id = ?`, id)
		for _, e := range *req.Entries {
			h.DB.Exec(`INSERT INTO user_list_entries (list_id, variant_id, gunnery, piloting) VALUES (?, ?, ?, ?)`,
				id, e.VariantID, e.Gunnery, e.Piloting)
		}
		h.DB.Exec(`UPDATE user_lists SET updated_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ListsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var ownerID int64
	if err := h.DB.QueryRow(`SELECT user_id FROM user_lists WHERE id = ?`, id).Scan(&ownerID); err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if user.ID != ownerID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	h.DB.Exec(`DELETE FROM user_lists WHERE id = ?`, id)
	w.WriteHeader(http.StatusOK)
}

func (h *ListsHandler) SharedView(w http.ResponseWriter, r *http.Request) {
	shareCode := r.PathValue("shareCode")
	if shareCode == "" {
		http.Error(w, "Missing share code", http.StatusBadRequest)
		return
	}

	var l UserList
	err := h.DB.QueryRow(
		`SELECT id, name, budget, share_code, created_at, updated_at FROM user_lists WHERE share_code = ?`, shareCode,
	).Scan(&l.ID, &l.Name, &l.Budget, &l.ShareCode, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	rows, err := h.DB.Query(`SELECT id, variant_id, gunnery, piloting FROM user_list_entries WHERE list_id = ?`, l.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e UserListEntry
			rows.Scan(&e.ID, &e.VariantID, &e.Gunnery, &e.Piloting)
			l.Entries = append(l.Entries, e)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l)
}
