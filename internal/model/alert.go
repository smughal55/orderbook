package model

import "time"

// AlertStatus represents the delivery state of a triggered alert.
type AlertStatus string

const (
	AlertStatusPending   AlertStatus = "pending"
	AlertStatusDelivered AlertStatus = "delivered"
	AlertStatusFailed    AlertStatus = "failed"
)

// Alert is a single triggered alert instance.
type Alert struct {
	ID           string
	RuleID       string
	RuleName     string
	Source       string
	TriggerValue float64
	Threshold    float64
	Status       AlertStatus
	CreatedAt    time.Time
	DeliveredAt  *time.Time
}
