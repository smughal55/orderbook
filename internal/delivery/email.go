package delivery

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shazmughal/orderbook/internal/model"
)

// EmailHandler delivers alerts via email (stub — plug in SendGrid/SES).
type EmailHandler struct {
	MaxRetries int
}

func NewEmailHandler() *EmailHandler {
	return &EmailHandler{MaxRetries: 3}
}

func (h *EmailHandler) Type() string { return "email" }

func (h *EmailHandler) Send(ctx context.Context, alert *model.TriggeredAlert, target string) error {
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

		if err := sendEmail(target, alert); err != nil {
			lastErr = err
			log.Printf("email: attempt %d to %s failed: %v", attempt+1, target, err)
			continue
		}
		return nil
	}
	return fmt.Errorf("email: all %d attempts failed: %w", h.MaxRetries+1, lastErr)
}

func sendEmail(to string, alert *model.TriggeredAlert) error {
	// TODO: integrate with SendGrid or AWS SES
	log.Printf("email: [STUB] sending alert %q to %s (matched: %s)", alert.RuleName, to, alert.MatchedValue)
	return nil
}
