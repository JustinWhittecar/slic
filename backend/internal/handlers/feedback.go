package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type FeedbackHandler struct {
	mu       sync.Mutex
	lastSeen map[string]time.Time // simple rate limit by IP
}

func NewFeedbackHandler() *FeedbackHandler {
	return &FeedbackHandler{lastSeen: make(map[string]time.Time)}
}

type feedbackRequest struct {
	Type    string `json:"type"`    // bug, feature, general
	Message string `json:"message"`
	Contact string `json:"contact"` // optional email/name
}

func (h *FeedbackHandler) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limit: 1 submission per IP per 60 seconds
	ip := r.Header.Get("Fly-Client-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	h.mu.Lock()
	if last, ok := h.lastSeen[ip]; ok && time.Since(last) < 60*time.Second {
		h.mu.Unlock()
		http.Error(w, "Please wait a minute before submitting again", http.StatusTooManyRequests)
		return
	}
	h.lastSeen[ip] = time.Now()
	// Clean old entries
	for k, v := range h.lastSeen {
		if time.Since(v) > 5*time.Minute {
			delete(h.lastSeen, k)
		}
	}
	h.mu.Unlock()

	// Parse request
	body, err := io.ReadAll(io.LimitReader(r.Body, 10_000))
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var req feedbackRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}
	if len(req.Message) > 5000 {
		http.Error(w, "Message too long", http.StatusBadRequest)
		return
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		// No token — just log it
		fmt.Printf("[FEEDBACK] type=%s contact=%s message=%s\n", req.Type, req.Contact, req.Message)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "logged"})
		return
	}

	// Create GitHub issue
	typeLabel := req.Type
	if typeLabel == "" {
		typeLabel = "general"
	}

	title := fmt.Sprintf("[Feedback] %s: %s", typeLabel, truncate(req.Message, 80))

	issueBody := fmt.Sprintf("**Type:** %s\n\n**Message:**\n%s", typeLabel, req.Message)
	if req.Contact != "" {
		issueBody += fmt.Sprintf("\n\n**Contact:** %s", req.Contact)
	}

	ghReq := map[string]interface{}{
		"title":  title,
		"body":   issueBody,
		"labels": []string{"feedback"},
	}
	ghBody, _ := json.Marshal(ghReq)

	httpReq, err := http.NewRequest("POST", "https://api.github.com/repos/JustinWhittecar/slic/issues", bytes.NewReader(ghBody))
	if err != nil {
		fmt.Printf("[FEEDBACK ERROR] failed to create request: %v\n", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Authorization", "token "+token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		fmt.Printf("[FEEDBACK ERROR] GitHub API error: %v\n", err)
		http.Error(w, "Failed to submit feedback", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("[FEEDBACK ERROR] GitHub API status %d: %s\n", resp.StatusCode, string(respBody))
		http.Error(w, "Failed to submit feedback", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
