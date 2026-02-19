package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type EventsHandler struct {
	DB *sql.DB

	mu      sync.Mutex
	ipCount map[string]*rateBucket
}

type rateBucket struct {
	count  int
	window time.Time
}

func NewEventsHandler(db *sql.DB) *EventsHandler {
	return &EventsHandler{
		DB:      db,
		ipCount: make(map[string]*rateBucket),
	}
}

func (h *EventsHandler) checkRate(ip string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	b, ok := h.ipCount[ip]
	if !ok || now.Sub(b.window) > time.Minute {
		h.ipCount[ip] = &rateBucket{count: 1, window: now}
		return true
	}
	b.count++
	return b.count <= 100
}

type eventPayload struct {
	Events []eventItem `json:"events"`
}

type eventItem struct {
	Name       string         `json:"name"`
	Properties map[string]any `json:"properties"`
	PageURL    string         `json:"page_url"`
	Referrer   string         `json:"referrer"`
}

func (h *EventsHandler) Track(w http.ResponseWriter, r *http.Request) {
	// Extract IP
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ip = strings.Split(ip, ",")[0]
		ip = strings.TrimSpace(ip)
	} else {
		ip = r.RemoteAddr
	}

	if !h.checkRate(ip) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	var payload eventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(payload.Events) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if len(payload.Events) > 50 {
		payload.Events = payload.Events[:50]
	}

	// Session ID from cookie
	sessionID := ""
	if c, err := r.Cookie("slic_sid"); err == nil && c.Value != "" {
		sessionID = c.Value
	} else {
		sessionID = uuid.New().String()
		http.SetCookie(w, &http.Cookie{
			Name:     "slic_sid",
			Value:    sessionID,
			Path:     "/",
			MaxAge:   30 * 24 * 60 * 60,
			HttpOnly: false,
			Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
			SameSite: http.SameSiteLaxMode,
		})
	}

	// User ID from auth context (optional)
	var userID *int64
	if u := UserFromContext(r.Context()); u != nil {
		userID = &u.ID
	}

	ua := r.Header.Get("User-Agent")

	// Batch insert
	tx, err := h.DB.Begin()
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	stmt, err := tx.Prepare(`INSERT INTO events (user_id, session_id, event_name, properties, page_url, referrer, user_agent, ip) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	defer stmt.Close()

	for _, ev := range payload.Events {
		var propsJSON *string
		if ev.Properties != nil {
			b, _ := json.Marshal(ev.Properties)
			s := string(b)
			propsJSON = &s
		}
		stmt.Exec(userID, sessionID, ev.Name, propsJSON, ev.PageURL, ev.Referrer, ua, ip)
	}
	tx.Commit()

	w.WriteHeader(http.StatusNoContent)
}

func (h *EventsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	// Simple admin check: require any logged-in user
	if UserFromContext(r.Context()) == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	type topEvent struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	type topQuery struct {
		Query string `json:"query"`
		Count int    `json:"count"`
	}
	type topMech struct {
		MechName string `json:"mech_name"`
		Count    int    `json:"count"`
	}
	type dauEntry struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	type statsResp struct {
		TopEvents  []topEvent `json:"top_events"`
		TopQueries []topQuery `json:"top_queries"`
		TopMechs   []topMech  `json:"top_mechs"`
		DAU        []dauEntry `json:"dau"`
	}

	resp := statsResp{}

	// Top events last 7 days
	rows, err := h.DB.Query(`SELECT event_name, COUNT(*) as cnt FROM events WHERE created_at > datetime('now', '-7 days') GROUP BY event_name ORDER BY cnt DESC LIMIT 20`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e topEvent
			rows.Scan(&e.Name, &e.Count)
			resp.TopEvents = append(resp.TopEvents, e)
		}
	}

	// Top search queries
	rows2, err := h.DB.Query(`SELECT json_extract(properties, '$.query') as q, COUNT(*) as cnt FROM events WHERE event_name = 'search' AND created_at > datetime('now', '-7 days') AND json_extract(properties, '$.query') IS NOT NULL GROUP BY q ORDER BY cnt DESC LIMIT 20`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var q topQuery
			rows2.Scan(&q.Query, &q.Count)
			resp.TopQueries = append(resp.TopQueries, q)
		}
	}

	// Top mech detail views
	rows3, err := h.DB.Query(`SELECT json_extract(properties, '$.mech_name') as mn, COUNT(*) as cnt FROM events WHERE event_name = 'mech_view' AND created_at > datetime('now', '-7 days') AND json_extract(properties, '$.mech_name') IS NOT NULL GROUP BY mn ORDER BY cnt DESC LIMIT 20`)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var m topMech
			rows3.Scan(&m.MechName, &m.Count)
			resp.TopMechs = append(resp.TopMechs, m)
		}
	}

	// DAU last 7 days
	rows4, err := h.DB.Query(`SELECT date(created_at) as d, COUNT(DISTINCT session_id) as cnt FROM events WHERE created_at > datetime('now', '-7 days') GROUP BY d ORDER BY d`)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			var d dauEntry
			rows4.Scan(&d.Date, &d.Count)
			resp.DAU = append(resp.DAU, d)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
