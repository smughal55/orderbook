package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shazmughal/orderbook/internal/model"
)

// WebhookHandler delivers alerts via HTTP POST webhook.
type WebhookHandler struct {
	client     *http.Client
	MaxRetries int
}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		client:     &http.Client{Timeout: 10 * time.Second},
		MaxRetries: 3,
	}
}

func (h *WebhookHandler) Type() string { return "webhook" }

func (h *WebhookHandler) Send(ctx context.Context, alert *model.TriggeredAlert, target string) error {
	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("webhook marshal: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= h.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("webhook request build: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := h.client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("webhook: attempt %d to %s failed: %v", attempt+1, target, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("webhook: HTTP %d from %s", resp.StatusCode, target)
		log.Printf("webhook: attempt %d to %s returned %d", attempt+1, target, resp.StatusCode)
	}
	return fmt.Errorf("webhook: all %d attempts failed: %w", h.MaxRetries+1, lastErr)
}
