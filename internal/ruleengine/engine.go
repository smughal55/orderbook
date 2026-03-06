package ruleengine

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shazmughal/orderbook/internal/evaluator"
	"github.com/shazmughal/orderbook/internal/model"
)

// Engine applies cooldown/suppression and enriches candidate alerts
// into fully formed TriggeredAlerts.
type Engine struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Engine {
	return &Engine{rdb: rdb}
}

// Process takes candidate alerts, applies cooldown, and returns those that should fire.
func (e *Engine) Process(ctx context.Context, candidates []evaluator.CandidateAlert) []model.TriggeredAlert {
	var triggered []model.TriggeredAlert

	for _, c := range candidates {
		if !e.checkCooldown(ctx, c.Rule) {
			continue
		}

		alert := model.NewTriggeredAlert(
			c.Rule.ID,
			c.Rule.Name,
			c.Record.SourceID,
			c.Record.RecordID,
			c.MatchedValue,
			c.Rule.Condition,
			c.Rule.DeliveryChannels,
		)

		triggered = append(triggered, alert)
	}

	return triggered
}

// checkCooldown returns true if the rule is NOT in cooldown (i.e., OK to fire).
// It also sets the cooldown key if the check passes.
func (e *Engine) checkCooldown(ctx context.Context, rule *model.AlertRule) bool {
	if rule.CooldownSeconds <= 0 {
		return true
	}

	key := fmt.Sprintf("cooldown:%s", rule.ID)
	ttl := time.Duration(rule.CooldownSeconds) * time.Second

	ok, err := e.rdb.SetNX(ctx, key, 1, ttl).Result()
	if err != nil {
		return true
	}
	return ok
}
