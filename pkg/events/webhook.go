package events

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// WebhookSink POSTs each event as its JSON envelope to a fixed URL. One
// retry, 5s timeout, then the event is dropped with a warning. Best-effort
// by design (ADR-007 §2).
type WebhookSink struct {
	URL    string
	Client *http.Client
}

// NewWebhookSink builds a webhook sink with a 5s-timeout client.
func NewWebhookSink(url string) WebhookSink {
	return WebhookSink{URL: url, Client: &http.Client{Timeout: 5 * time.Second}}
}

// Deliver implements Sink.
func (s WebhookSink) Deliver(e Event) {
	body, err := json.Marshal(e)
	if err != nil {
		slog.Warn("events webhook: marshal", "event", e.Type, "err", err)
		return
	}
	for attempt := 0; attempt < 2; attempt++ {
		if s.post(body) {
			return
		}
	}
	slog.Warn("events webhook: dropped after retry", "event", e.Type, "url", s.URL)
}

func (s WebhookSink) post(body []byte) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
