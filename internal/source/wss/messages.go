package wss

import (
	"encoding/json"
	"fmt"
)

// PriceLevel represents a single price/amount entry in a snapshot or update.
type PriceLevel struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

// SnapshotMessage carries a full replacement of the order book.
type SnapshotMessage struct {
	Type string       `json:"type"`
	Bids []PriceLevel `json:"bids"`
	Asks []PriceLevel `json:"asks"`
}

// UpdateMessage carries a single incremental change to the order book.
type UpdateMessage struct {
	Type   string  `json:"type"`
	Side   string  `json:"side"`
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

// ParseMessage inspects the "type" field and returns
// either a *SnapshotMessage or an *UpdateMessage.
func ParseMessage(data []byte) (any, error) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}

	switch envelope.Type {
	case "snapshot":
		var msg SnapshotMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return &msg, nil
	case "update":
		var msg UpdateMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return &msg, nil
	default:
		return nil, fmt.Errorf("unknown message type: %q", envelope.Type)
	}
}
