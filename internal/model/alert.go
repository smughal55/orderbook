package model

// EvalType determines whether an alert fires on a single record or a trend.
type EvalType string

const (
	EvalTypeSingle EvalType = "single"
	EvalTypeTrend  EvalType = "trend"
)

// Aggregation is the function applied over a time window for trend alerts.
type Aggregation string

const (
	AggregationAvg          Aggregation = "avg"
	AggregationMax          Aggregation = "max"
	AggregationMin          Aggregation = "min"
	AggregationRateOfChange Aggregation = "rate_of_change"
)

// DeliveryChannel describes where an alert should be sent.
type DeliveryChannel struct {
	Type   string `json:"type"`   // "email", "sms", "webhook"
	Target string `json:"target"` // email address, phone number, or URL
}

// AlertRule is the in-memory representation of a user-configured alert.
type AlertRule struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	SourceFilter     string            `json:"source_filter"` // source_id pattern or "*"
	EvalType         EvalType          `json:"eval_type"`
	Condition        string            `json:"condition"`        // expression, e.g. "payload.price > 150"
	WindowSeconds    int               `json:"window_seconds"`   // trend only
	Aggregation      Aggregation       `json:"aggregation"`      // trend only
	Field            string            `json:"field"`            // trend only: which payload field to aggregate
	DeliveryChannels []DeliveryChannel `json:"delivery_channels"`
	CooldownSeconds  int               `json:"cooldown_seconds"`
	Enabled          bool              `json:"enabled"`
}

// MatchesSource returns true if the rule applies to the given source.
func (r *AlertRule) MatchesSource(sourceID string) bool {
	return r.SourceFilter == "*" || r.SourceFilter == sourceID
}
