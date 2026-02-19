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

const transactionalURL = "https://api.customer.io/v1/send/email"

// Client is a fire-and-forget Customer.io Track API client.
// All methods are safe to call when credentials are missing (no-op).
type Client struct {
	siteID    string
	apiKey    string
	appAPIKey string // App API key for transactional sends
	http      *http.Client
}

// New creates a Client. If siteID or apiKey are empty, track calls become no-ops.
func New(siteID, apiKey, appAPIKey string) *Client {
	return &Client{
		siteID:    siteID,
		apiKey:    apiKey,
		appAPIKey: appAPIKey,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) enabled() bool {
	return c != nil && c.siteID != "" && c.apiKey != ""
}

func (c *Client) transactionalEnabled() bool {
	return c != nil && c.appAPIKey != ""
}

// SendTransactional sends a transactional email via Customer.io's App API.
// Fire-and-forget: runs in a goroutine.
func (c *Client) SendTransactional(to, subject, htmlBody string, identifiers map[string]string, messageData map[string]interface{}) {
	if !c.transactionalEnabled() {
		log.Printf("[CIO] transactional send skipped (no app API key)")
		return
	}
	go func() {
		payload := map[string]interface{}{
			"to":                       to,
			"transactional_message_id": "1",
			"from":                     "SLIC Command <slic@starleagueintelligencecommand.com>",
			"identifiers":              identifiers,
			"subject":                  subject,
			"body_html":                htmlBody,
		}
		if len(messageData) > 0 {
			payload["message_data"] = messageData
		}
		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[CIO] transactional marshal error: %v", err)
			return
		}
		req, err := http.NewRequest("POST", transactionalURL, bytes.NewReader(body))
		if err != nil {
			log.Printf("[CIO] transactional request error: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.appAPIKey)

		resp, err := c.http.Do(req)
		if err != nil {
			log.Printf("[CIO] transactional send error: %v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			var respBody bytes.Buffer
			respBody.ReadFrom(resp.Body)
			log.Printf("[CIO] transactional send status: %d body: %s", resp.StatusCode, respBody.String())
		} else {
			log.Printf("[CIO] transactional email sent to %s: %s", to, subject)
		}
	}()
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
