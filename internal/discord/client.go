package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type payload struct {
	Username string  `json:"username"`
	Embeds   []embed `json:"embeds"`
}

type embed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

// BuildPayload creates a compact JSON payload for Discord Webhook.
func BuildPayload(from, subject, body string) []byte {
	excerpt := summarizeBody(body)
	description := excerpt
	if from != "" {
		description = fmt.Sprintf("From: %s\n\n%s", from, excerpt)
	}
	if subject == "" {
		subject = "(no subject)"
	}
	p := payload{
		Username: "mail-relay",
		Embeds: []embed{{
			Title:       subject,
			Description: description,
			Color:       5814783,
		}},
	}
	b, err := json.Marshal(p)
	if err != nil {
		return []byte(`{"content":"` + strings.ReplaceAll(description, "\n", " ") + `"}`)
	}
	return b
}

func summarizeBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "(empty body)"
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	if len(body) <= 300 {
		return body
	}
	return body[:297] + "..."
}

// SendWithRetry sends a payload to the webhook URL and retries on transient failures.
func SendWithRetry(url string, payload []byte, maxRetries int, backoff time.Duration) error {
	if maxRetries < 0 {
		maxRetries = 0
	}
	if backoff <= 0 {
		backoff = 1 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("webhook returned status %s", resp.Status)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return lastErr
			}
		}

		if attempt == maxRetries {
			break
		}
		time.Sleep(backoff * time.Duration(attempt+1))
	}
	return lastErr
}
