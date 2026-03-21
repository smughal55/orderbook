package model

import "testing"

// --- AlertRule.Validate() ---

func TestAlertRule_Validate_ValidSingleRule(t *testing.T) {
	r := AlertRule{
		Name:    "high bid",
		Source:  "wss",
		Enabled: true,
		Type:    RuleTypeSingle,
		Condition: RuleCondition{
			Field:     "best_bid",
			Operator:  OperatorGT,
			Threshold: 100,
		},
		CooldownSeconds: 30,
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestAlertRule_Validate_ValidTrendRule(t *testing.T) {
	r := AlertRule{
		Name:    "avg bid rising",
		Source:  "wss",
		Enabled: true,
		Type:    RuleTypeTrend,
		Condition: RuleCondition{
			Field:     "best_bid",
			Operator:  OperatorGT,
			Threshold: 99,
		},
		TrendConfig: TrendConfig{
			WindowSeconds: 60,
			Aggregate:     AggregateAvg,
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestAlertRule_Validate_EmptyName(t *testing.T) {
	r := AlertRule{
		Name:   "",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "best_bid",
			Operator: OperatorGT,
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestAlertRule_Validate_InvalidFieldName_Empty(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "",
			Operator: OperatorGT,
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty field name")
	}
}

func TestAlertRule_Validate_InvalidFieldName_StartsWithDigit(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "1invalid",
			Operator: OperatorGT,
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for field name starting with digit")
	}
}

func TestAlertRule_Validate_InvalidFieldName_SpecialChars(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "field-name",
			Operator: OperatorGT,
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for field name with hyphen")
	}
}

func TestAlertRule_Validate_ValidFieldName_WithDotAndUnderscore(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "order.best_bid",
			Operator: OperatorGT,
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected nil for valid dotted field name, got: %v", err)
	}
}

func TestAlertRule_Validate_UnknownOperator(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "best_bid",
			Operator: "between",
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for unknown operator")
	}
}

func TestAlertRule_Validate_AllOperatorsValid(t *testing.T) {
	operators := []Operator{OperatorGT, OperatorGTE, OperatorLT, OperatorLTE, OperatorEQ, OperatorNEQ}
	for _, op := range operators {
		r := AlertRule{
			Name:   "test",
			Source: "wss",
			Type:   RuleTypeSingle,
			Condition: RuleCondition{
				Field:    "best_bid",
				Operator: op,
			},
		}
		if err := r.Validate(); err != nil {
			t.Fatalf("expected nil for operator %q, got: %v", op, err)
		}
	}
}

func TestAlertRule_Validate_TrendRule_ZeroWindow(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeTrend,
		Condition: RuleCondition{
			Field:    "best_bid",
			Operator: OperatorGT,
		},
		TrendConfig: TrendConfig{
			WindowSeconds: 0,
			Aggregate:     AggregateAvg,
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for trend rule with zero window")
	}
}

func TestAlertRule_Validate_TrendRule_NegativeWindow(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeTrend,
		Condition: RuleCondition{
			Field:    "best_bid",
			Operator: OperatorGT,
		},
		TrendConfig: TrendConfig{
			WindowSeconds: -10,
			Aggregate:     AggregateAvg,
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for trend rule with negative window")
	}
}

func TestAlertRule_Validate_TrendRule_UnknownAggregate(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeTrend,
		Condition: RuleCondition{
			Field:    "best_bid",
			Operator: OperatorGT,
		},
		TrendConfig: TrendConfig{
			WindowSeconds: 60,
			Aggregate:     "median",
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for unknown aggregate")
	}
}

func TestAlertRule_Validate_TrendRule_AllAggregatesValid(t *testing.T) {
	aggregates := []Aggregate{
		AggregateAvg, AggregateMin, AggregateMax,
		AggregateSum, AggregateCount, AggregateRateOfChange,
	}
	for _, agg := range aggregates {
		r := AlertRule{
			Name:   "test",
			Source: "wss",
			Type:   RuleTypeTrend,
			Condition: RuleCondition{
				Field:    "best_bid",
				Operator: OperatorGT,
			},
			TrendConfig: TrendConfig{
				WindowSeconds: 60,
				Aggregate:     agg,
			},
		}
		if err := r.Validate(); err != nil {
			t.Fatalf("expected nil for aggregate %q, got: %v", agg, err)
		}
	}
}

func TestAlertRule_Validate_UnknownRuleType(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   "rolling",
		Condition: RuleCondition{
			Field:    "best_bid",
			Operator: OperatorGT,
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for unknown rule type")
	}
}

func TestAlertRule_Validate_ZeroValue(t *testing.T) {
	var r AlertRule
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for zero-value AlertRule")
	}
}

func TestAlertRule_Validate_SingleRule_TrendConfigIgnored(t *testing.T) {
	// TrendConfig fields on a single-type rule are not validated — single rules
	// do not require window or aggregate configuration.
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "best_bid",
			Operator: OperatorGT,
		},
		TrendConfig: TrendConfig{
			WindowSeconds: 0,       // would fail for a trend rule
			Aggregate:     "bogus", // would fail for a trend rule
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected nil: single rule should not validate TrendConfig, got: %v", err)
	}
}

func TestAlertRule_Validate_ValidFieldName_SingleUnderscore(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "_",
			Operator: OperatorGT,
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected nil for single-underscore field name, got: %v", err)
	}
}

func TestAlertRule_Validate_ValidFieldName_SingleLetter(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "a",
			Operator: OperatorGT,
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected nil for single-letter field name, got: %v", err)
	}
}

func TestAlertRule_Validate_InvalidFieldName_Space(t *testing.T) {
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeSingle,
		Condition: RuleCondition{
			Field:    "best bid",
			Operator: OperatorGT,
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for field name containing a space")
	}
}

func TestAlertRule_Validate_TrendRule_EmptyAggregate(t *testing.T) {
	// Empty string is not a valid aggregate; distinct from an unknown word.
	r := AlertRule{
		Name:   "test",
		Source: "wss",
		Type:   RuleTypeTrend,
		Condition: RuleCondition{
			Field:    "best_bid",
			Operator: OperatorGT,
		},
		TrendConfig: TrendConfig{
			WindowSeconds: 60,
			Aggregate:     "",
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty aggregate string")
	}
}

// --- ChannelConfig.Validate() ---

func TestChannelConfig_Validate_ValidWebhook(t *testing.T) {
	c := ChannelConfig{Type: ChannelTypeWebhook, Enabled: true}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected nil for webhook, got: %v", err)
	}
}

func TestChannelConfig_Validate_ValidEmail(t *testing.T) {
	c := ChannelConfig{Type: ChannelTypeEmail, Enabled: true}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected nil for email, got: %v", err)
	}
}

func TestChannelConfig_Validate_ValidSMS(t *testing.T) {
	c := ChannelConfig{Type: ChannelTypeSMS, Enabled: true}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected nil for sms, got: %v", err)
	}
}

func TestChannelConfig_Validate_UnknownType(t *testing.T) {
	c := ChannelConfig{Type: "slack"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for unknown channel type")
	}
}

func TestChannelConfig_Validate_EmptyType(t *testing.T) {
	c := ChannelConfig{}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty channel type")
	}
}

func TestChannelConfig_Validate_DisabledWithValidType(t *testing.T) {
	// Enabled=false is not a validation constraint; a disabled channel
	// with a valid type must still pass Validate().
	c := ChannelConfig{Type: ChannelTypeWebhook, Enabled: false}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected nil for disabled-but-valid channel, got: %v", err)
	}
}
