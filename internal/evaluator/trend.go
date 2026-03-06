package evaluator

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/redis/go-redis/v9"
	"github.com/shazmughal/orderbook/internal/model"
)

// TrendEvaluator maintains sliding windows in Redis and evaluates trend-based rules.
type TrendEvaluator struct {
	mu       sync.RWMutex
	rdb      *redis.Client
	programs map[string]*vm.Program
	rules    map[string]*model.AlertRule
}

func NewTrendEvaluator(rdb *redis.Client) *TrendEvaluator {
	return &TrendEvaluator{
		rdb:      rdb,
		programs: make(map[string]*vm.Program),
		rules:    make(map[string]*model.AlertRule),
	}
}

// LoadRules compiles and caches trend alert rule expressions.
func (e *TrendEvaluator) LoadRules(rules []model.AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	newPrograms := make(map[string]*vm.Program, len(rules))
	newRules := make(map[string]*model.AlertRule, len(rules))

	for i := range rules {
		r := &rules[i]
		if r.EvalType != model.EvalTypeTrend || !r.Enabled {
			continue
		}

		prog, err := expr.Compile(r.Condition, expr.AsBool())
		if err != nil {
			return fmt.Errorf("compile trend rule %q (%s): %w", r.Name, r.Condition, err)
		}
		newPrograms[r.ID] = prog
		newRules[r.ID] = r
	}

	e.programs = newPrograms
	e.rules = newRules
	return nil
}

// Evaluate adds the record's value to the sliding window and checks trend rules.
func (e *TrendEvaluator) Evaluate(ctx context.Context, record *model.DataRecord) []CandidateAlert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var candidates []CandidateAlert

	for id, rule := range e.rules {
		if !rule.MatchesSource(record.SourceID) {
			continue
		}

		fieldVal, ok := extractFloat(record.Payload, rule.Field)
		if !ok {
			continue
		}

		windowKey := fmt.Sprintf("trend:%s:%s:%s", rule.ID, record.SourceID, rule.Field)
		now := time.Now().UnixMilli()
		windowStart := now - int64(rule.WindowSeconds)*1000

		e.rdb.ZAdd(ctx, windowKey, redis.Z{
			Score:  float64(now),
			Member: fmt.Sprintf("%f:%d", fieldVal, now),
		})
		e.rdb.ZRemRangeByScore(ctx, windowKey, "-inf", strconv.FormatInt(windowStart, 10))
		e.rdb.Expire(ctx, windowKey, time.Duration(rule.WindowSeconds*2)*time.Second)

		aggValue, err := e.computeAggregation(ctx, windowKey, rule.Aggregation, windowStart)
		if err != nil {
			continue
		}

		env := map[string]interface{}{
			"source_id": record.SourceID,
			"timestamp": record.Timestamp,
			"payload":   record.Payload,
			"value":     aggValue,
			"window":    rule.WindowSeconds,
		}

		prog := e.programs[id]
		result, err := expr.Run(prog, env)
		if err != nil {
			continue
		}

		matched, ok := result.(bool)
		if !ok || !matched {
			continue
		}

		candidates = append(candidates, CandidateAlert{
			Rule:         rule,
			Record:       record,
			MatchedValue: fmt.Sprintf("agg(%s)=%.4f over %ds", rule.Aggregation, aggValue, rule.WindowSeconds),
		})
	}

	return candidates
}

func (e *TrendEvaluator) computeAggregation(ctx context.Context, key string, agg model.Aggregation, windowStart int64) (float64, error) {
	members, err := e.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: strconv.FormatInt(windowStart, 10),
		Max: "+inf",
	}).Result()
	if err != nil || len(members) == 0 {
		return 0, fmt.Errorf("no data in window")
	}

	values := make([]float64, 0, len(members))
	for _, m := range members {
		v, err := parseMemberValue(m)
		if err != nil {
			continue
		}
		values = append(values, v)
	}

	if len(values) == 0 {
		return 0, fmt.Errorf("no valid values")
	}

	switch agg {
	case model.AggregationAvg:
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values)), nil
	case model.AggregationMax:
		max := math.Inf(-1)
		for _, v := range values {
			if v > max {
				max = v
			}
		}
		return max, nil
	case model.AggregationMin:
		min := math.Inf(1)
		for _, v := range values {
			if v < min {
				min = v
			}
		}
		return min, nil
	case model.AggregationRateOfChange:
		if len(values) < 2 {
			return 0, nil
		}
		first := values[0]
		last := values[len(values)-1]
		if first == 0 {
			return 0, nil
		}
		return (last - first) / first, nil
	default:
		return 0, fmt.Errorf("unknown aggregation: %s", agg)
	}
}

func parseMemberValue(member string) (float64, error) {
	for i := len(member) - 1; i >= 0; i-- {
		if member[i] == ':' {
			return strconv.ParseFloat(member[:i], 64)
		}
	}
	return strconv.ParseFloat(member, 64)
}

func extractFloat(payload map[string]interface{}, field string) (float64, bool) {
	val, ok := payload[field]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
