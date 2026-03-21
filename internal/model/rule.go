package model

import (
	"errors"
	"regexp"
	"time"
)

// RuleType distinguishes single-record rules from trend-based rules.
type RuleType string

const (
	RuleTypeSingle RuleType = "single"
	RuleTypeTrend  RuleType = "trend"
)

// Operator is the comparison operator used in a rule condition.
type Operator string

const (
	OperatorGT  Operator = "gt"
	OperatorGTE Operator = "gte"
	OperatorLT  Operator = "lt"
	OperatorLTE Operator = "lte"
	OperatorEQ  Operator = "eq"
	OperatorNEQ Operator = "neq"
)

var validOperators = map[Operator]struct{}{
	OperatorGT:  {},
	OperatorGTE: {},
	OperatorLT:  {},
	OperatorLTE: {},
	OperatorEQ:  {},
	OperatorNEQ: {},
}

// Aggregate is the aggregation function applied over a sliding window.
type Aggregate string

const (
	AggregateAvg           Aggregate = "avg"
	AggregateMin           Aggregate = "min"
	AggregateMax           Aggregate = "max"
	AggregateSum           Aggregate = "sum"
	AggregateCount         Aggregate = "count"
	AggregateRateOfChange  Aggregate = "rate_of_change"
)

var validAggregates = map[Aggregate]struct{}{
	AggregateAvg:          {},
	AggregateMin:          {},
	AggregateMax:          {},
	AggregateSum:          {},
	AggregateCount:        {},
	AggregateRateOfChange: {},
}

// fieldNameRe validates that a field name is a safe identifier.
var fieldNameRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

// RuleCondition specifies the field, operator, and threshold for evaluation.
// Value type — no pointer indirection needed.
type RuleCondition struct {
	Field     string
	Operator  Operator
	Threshold float64
}

// TrendConfig specifies the sliding window parameters for trend-type rules.
// Value type — no pointer indirection needed.
type TrendConfig struct {
	WindowSeconds int
	Aggregate     Aggregate
}

// AlertRule is a user-defined rule that triggers alerts when its condition is met.
type AlertRule struct {
	ID              string
	Name            string
	Source          string
	Enabled         bool
	Type            RuleType
	Condition       RuleCondition
	TrendConfig     TrendConfig
	ChannelIDs      []string
	CooldownSeconds int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Validate returns an error if the AlertRule is not valid for use.
func (r AlertRule) Validate() error {
	if r.Name == "" {
		return errors.New("name must not be empty")
	}
	if !fieldNameRe.MatchString(r.Condition.Field) {
		return errors.New("condition.field must match ^[a-zA-Z_][a-zA-Z0-9_.]*$")
	}
	if _, ok := validOperators[r.Condition.Operator]; !ok {
		return errors.New("condition.operator must be one of: gt, gte, lt, lte, eq, neq")
	}
	if r.Type != RuleTypeSingle && r.Type != RuleTypeTrend {
		return errors.New("type must be one of: single, trend")
	}
	if r.Type == RuleTypeTrend {
		if r.TrendConfig.WindowSeconds <= 0 {
			return errors.New("trend_config.window_seconds must be greater than zero")
		}
		if _, ok := validAggregates[r.TrendConfig.Aggregate]; !ok {
			return errors.New("trend_config.aggregate must be one of: avg, min, max, sum, count, rate_of_change")
		}
	}
	return nil
}
