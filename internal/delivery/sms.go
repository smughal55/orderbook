package delivery

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shazmughal/orderbook/internal/model"
)

// SMSHandler delivers alerts via SMS (stub — plug in Twilio).
type SMSHandler struct {
	MaxRetries int
}

func NewSMSHandler() *SMSHandler {
	return &SMSHandler{MaxRetries: 3}
}

func (h *SMSHandler) Type() string { return "sms" }

func (h *SMSHandler) Send(ctx context.Context, alert *model.TriggeredAlert, target string) error {
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

		if err := sendSMS(target, alert); err != nil {
			lastErr = err
			log.Printf("sms: attempt %d to %s failed: %v", attempt+1, target, err)
			continue
		}
		return nil
	}
	return fmt.Errorf("sms: all %d attempts failed: %w", h.MaxRetries+1, lastErr)
}

func sendSMS(to string, alert *model.TriggeredAlert) error {
	// TODO: integrate with Twilio
	log.Printf("sms: [STUB] sending alert %q to %s (matched: %s)", alert.RuleName, to, alert.MatchedValue)
	return nil
}
