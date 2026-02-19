package customerio

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const trackBaseURL = "https://track.customer.io/api/v1"

// Client is a fire-and-forget Customer.io Track API client.
// All methods are safe to call when credentials are missing (no-op).
type Client struct {
	siteID string
	apiKey string
	http   *http.Client
}

// New creates a Client. If siteID or apiKey are empty, all calls become no-ops.
func New(siteID, apiKey string) *Client {
	return &Client{
		siteID: siteID,
		apiKey: apiKey,
		http:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) enabled() bool {
	return c != nil && c.siteID != "" && c.apiKey != ""
}

// Identify creates or updates a customer. PUT /customers/{id}
func (c *Client) Identify(userID string, attrs map[string]interface{}) {
	if !c.enabled() {
		return
	}
	go func() {
		url := fmt.Sprintf("%s/customers/%s", trackBaseURL, userID)
		c.do("PUT", url, attrs)
	}()
}

// Track sends a named event for a customer. POST /customers/{id}/events
func (c *Client) Track(userID string, eventName string, data map[string]interface{}) {
	if !c.enabled() {
		return
	}
	go func() {
		url := fmt.Sprintf("%s/customers/%s/events", trackBaseURL, userID)
		body := map[string]interface{}{"name": eventName}
		if len(data) > 0 {
			body["data"] = data
		}
		c.do("POST", url, body)
	}()
}

// AnonIDFromEmail returns a stable anonymous CIO identifier from an email.
func AnonIDFromEmail(email string) string {
	h := sha256.Sum256([]byte(email))
	return fmt.Sprintf("anon_%x", h[:6])
}

func (c *Client) do(method, url string, payload interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[CIO] marshal error: %v", err)
		return
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[CIO] request error: %v", err)
		return
	}
	req.SetBasicAuth(c.siteID, c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("[CIO] %s %s error: %v", method, url, err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("[CIO] %s %s status: %d", method, url, resp.StatusCode)
	}
}
