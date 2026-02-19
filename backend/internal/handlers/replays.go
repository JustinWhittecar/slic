package handlers

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"io"
	"net/http"
	"strconv"
)

type ReplayHandler struct {
	DB *sql.DB
}

func (h *ReplayHandler) GetReplay(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var data []byte
	err = h.DB.QueryRow(`SELECT replay_data FROM variant_replays WHERE variant_id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		http.Error(w, "no replay available", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// Decompress gzip
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		http.Error(w, "decompress error", http.StatusInternalServerError)
		return
	}
	defer gz.Close()

	jsonData, err := io.ReadAll(gz)
	if err != nil {
		http.Error(w, "decompress error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(jsonData)
}
