package model

import (
	"encoding/json"
	"time"
)

// Severity indicates the importance of a triggered alert.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// TriggeredAlert is produced when a rule matches and cooldown passes.
type TriggeredAlert struct {
	AlertID          string            `json:"alert_id"`
	RuleName         string            `json:"rule_name"`
	SourceID         string            `json:"source_id"`
	RecordID         string            `json:"record_id"`
	TriggeredAt      int64             `json:"triggered_at"`
	Severity         Severity          `json:"severity"`
	MatchedValue     string            `json:"matched_value"`
	Condition        string            `json:"condition"`
	DeliveryChannels []DeliveryChannel `json:"delivery_channels"`
}

func NewTriggeredAlert(ruleID, ruleName, sourceID, recordID, matchedValue, condition string, channels []DeliveryChannel) TriggeredAlert {
	return TriggeredAlert{
		AlertID:          ruleID,
		RuleName:         ruleName,
		SourceID:         sourceID,
		RecordID:         recordID,
		TriggeredAt:      time.Now().UnixMilli(),
		Severity:         SeverityWarning,
		MatchedValue:     matchedValue,
		Condition:        condition,
		DeliveryChannels: channels,
	}
}

func (a *TriggeredAlert) Marshal() ([]byte, error) {
	return json.Marshal(a)
}

func UnmarshalTriggeredAlert(data []byte) (*TriggeredAlert, error) {
	var a TriggeredAlert
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}
