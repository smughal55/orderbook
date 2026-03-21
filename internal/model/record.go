package model

import "time"

// DataRecord is a normalized data unit emitted by any source adapter.
type DataRecord struct {
	ID        string
	Source    string
	Timestamp time.Time
	Fields    map[string]float64
	Meta      map[string]string
}
